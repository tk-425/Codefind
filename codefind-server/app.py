from fastapi import FastAPI, HTTPException, Header, Request
from datetime import datetime, timezone
import requests
import uuid
import os
from typing import Optional
from transformers import AutoTokenizer
from dotenv import load_dotenv

from models import (
    TokenizeRequest,
    TokenizeResponse,
    EmbedRequest,
    EmbedResponse,
    IndexRequest,
    IndexResponse,
    ChunkStatusRequest,
    ChunkStatusResponse,
    PurgeRequest,
    PurgeResponse,
    DeletedFileInfo,
    StatsResponse,
    QueryRequest,
    QueryResponse,
    QueryResult,
    HealthResponse,
)
from auth import (
    validate_auth_key,
    validate_admin,
    create_first_manager,
    add_manager,
    remove_manager,
    list_managers,
)
from services.chromadb_service import ChromaDBService
from services.ollama_service import OllamaService
from audit import get_audit_logger
from ratelimit import get_rate_limiter, get_bootstrap_guard

# Load environment variables
load_dotenv()

app = FastAPI(title="Codefind API Server")

OLLAMA_URL = os.getenv("OLLAMA_URL")
CHROMADB_URL = os.getenv("CHROMADB_URL")

# Fail explicitly if environment variables not set
if not OLLAMA_URL:
    raise ValueError("OLLAMA_URL environment variable not set. Check .env file.")
if not CHROMADB_URL:
    raise ValueError("CHROMADB_URL environment variable not set. Check .env file.")

# Cache tokenizer at startup (loading is slow, cache once)
print("Loading tokenizer...")
_cached_tokenizer = AutoTokenizer.from_pretrained(
    "bert-base-uncased", trust_remote_code=True
)
print("Tokenizer loaded.")

# --- Health Endpoint ---


@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Check if Ollama and ChromaDB are reachable."""
    ollama_ok = False
    chromadb_ok = False
    error_msg = None

    try:
        resp = requests.get(f"{OLLAMA_URL}/api/tags", timeout=2)
        ollama_ok = resp.status_code == 200
    except Exception as e:
        error_msg = f"Ollama error: {str(e)}"

    try:
        resp = requests.get(f"{CHROMADB_URL}/api/v2/heartbeat", timeout=2)
        chromadb_ok = resp.status_code == 200
    except Exception as e:
        error_msg = f"ChromaDB error: {str(e)}"

    status = "ok" if (ollama_ok and chromadb_ok) else "error"

    return HealthResponse(
        status=status,
        ollama_status="ok" if ollama_ok else "error",
        chromadb_status="ok" if chromadb_ok else "error",
        timestamp=datetime.now(timezone.utc),
        error=error_msg,
    )


# --- Tokenize Endpoint ---


@app.post("/tokenize", response_model=TokenizeResponse)
async def tokenize(request: TokenizeRequest):
    """Count tokens using cached tokenizer."""
    try:
        token_counts = []
        for text in request.input:
            tokens = _cached_tokenizer.encode(text)
            token_counts.append(len(tokens))

        return TokenizeResponse(tokens=token_counts)

    except Exception as e:
        return TokenizeResponse(tokens=[], error=str(e))


# --- Embed Endpoint ---


@app.post("/embed", response_model=EmbedResponse)
async def embed(request: EmbedRequest):
    """Generate embeddings using Ollama."""
    try:
        ollama = OllamaService()
        embeddings = ollama.embed(model=request.model, texts=request.input)
        return EmbedResponse(embeddings=embeddings)

    except Exception as e:
        return EmbedResponse(embeddings=[], error=str(e))


# --- Index Endpoint (Protected) ---


@app.post("/index", response_model=IndexResponse)
async def index_chunks(
    request: IndexRequest,
    x_auth_key: Optional[str] = Header(None),
    x_auth_email: Optional[str] = Header(None),
):
    """Index chunks: embed via Ollama and store in ChromaDB."""

    # Step 1: Authenticate (email + key if provided, key only for backward compatibility)
    if not x_auth_key or not validate_auth_key(x_auth_key, x_auth_email):
        get_audit_logger().log_auth_fail(endpoint="/index")
        raise HTTPException(status_code=401, detail="Invalid auth key")

    # Step 2: Validate ChromaDB connection
    try:
        chroma = ChromaDBService()
        collection = chroma.get_or_create_collection(request.collection)
    except Exception as e:
        raise HTTPException(status_code=503, detail=f"ChromaDB error: {e}")

    # Step 3: Batch embed chunks via Ollama
    try:
        ollama = OllamaService()
        texts = [chunk.content for chunk in request.chunks]

        # Get embeddings from Ollama
        embeddings = ollama.embed(
            model=request.chunks[0].metadata.model_id, texts=texts
        )
    except Exception as e:
        raise HTTPException(status_code=503, detail=f"Ollama error: {e}")

    # Step 4: Store in ChromaDB
    try:
        ids = []
        documents = []
        metadatas = []

        for i, chunk in enumerate(request.chunks):
            # Use stable chunk ID if provided, otherwise generate UUID
            chunk_id = chunk.id if chunk.id else str(uuid.uuid4())
            ids.append(chunk_id)
            documents.append(chunk.content)

            # Convert metadata to dict with JSON serialization (datetime -> ISO string, exclude None values)
            metadata_dict = chunk.metadata.model_dump(mode="json", exclude_none=True)
            metadatas.append(metadata_dict)

        # Use upsert to handle re-indexing same chunks (idempotent)
        collection.upsert(
            ids=ids, documents=documents, embeddings=embeddings, metadatas=metadatas
        )

        # Audit log the index operation
        get_audit_logger().log_index(
            repo=request.collection,
            files=len(set(chunk.metadata.file_path for chunk in request.chunks)),
            chunks=len(ids),
            method="hybrid",
        )

        return IndexResponse(inserted_count=len(ids), updated_count=0, error="")
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Storage error: {e}")


# --- Soft Delete Endpoint (Authenticated) ---


@app.post("/chunks/delete")
async def soft_delete_chunks(
    repo_id: str,
    file_paths: list[str],
    x_auth_key: Optional[str] = Header(None),
    x_auth_email: Optional[str] = Header(None),
):
    """Mark chunks as deleted without removing from ChromaDB.

    This preserves history and allows recovery.
    Requires manager authentication.
    """
    # Validate auth (email + key if provided)
    if not x_auth_key or not validate_auth_key(x_auth_key, x_auth_email):
        get_audit_logger().log_auth_fail(endpoint="/chunks/delete")
        raise HTTPException(
            status_code=401,
            detail="Invalid or missing auth key",
        )

    try:
        chroma = ChromaDBService()
        collection = chroma.get_or_create_collection(repo_id)

        deleted_count = 0
        for path in file_paths:
            # Get chunks matching this file path
            results = collection.get(where={"file_path": path})

            if results and results["ids"]:
                for chunk_id in results["ids"]:
                    # Update metadata to mark as deleted
                    collection.update(
                        ids=[chunk_id],
                        metadatas=[
                            {
                                "status": "deleted",
                                "deleted_at": datetime.now(timezone.utc).isoformat(),
                            }
                        ],
                    )
                    deleted_count += 1

        # Audit log the delete operation
        get_audit_logger().log_delete(
            repo=repo_id, files=len(file_paths), chunks=deleted_count
        )

        return {"deleted_count": deleted_count, "file_paths": file_paths}
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Delete error: {e}")


# --- Clear Collection Endpoint (Authenticated) ---


@app.delete("/clear/{repo_id}")
async def clear_collection(
    repo_id: str,
    x_auth_key: Optional[str] = Header(None),
    x_auth_email: Optional[str] = Header(None),
):
    """Delete all chunks in a collection.

    This completely removes the collection from ChromaDB.
    Requires manager authentication.
    """
    # Validate auth (email + key if provided)
    if not x_auth_key or not validate_auth_key(x_auth_key, x_auth_email):
        get_audit_logger().log_auth_fail(endpoint="/clear")
        raise HTTPException(
            status_code=401,
            detail="Invalid or missing auth key",
        )

    try:
        chroma = ChromaDBService()
        # Check if collection exists first
        collection_names = [c.name for c in chroma.client.list_collections()]
        if repo_id in collection_names:
            # Delete the entire collection
            chroma.client.delete_collection(repo_id)
            get_audit_logger().log_clear(repo=repo_id)
            return {"status": "deleted", "repo_id": repo_id}
        else:
            # Collection doesn't exist - treat as success (already cleared)
            return {
                "status": "not_found",
                "repo_id": repo_id,
                "message": "Collection already deleted or never existed",
            }
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Clear error: {e}")


# --- Chunk Status Update Endpoint (Protected) ---


@app.patch("/chunks/status")
async def update_chunk_status(
    request: ChunkStatusRequest,
    x_auth_key: Optional[str] = Header(None),
    x_auth_email: Optional[str] = Header(None),
):
    """Mark chunks as deleted or active.

    This soft-deletes chunks by updating their status metadata.
    Used for tombstone mode - retains code history for querying.
    """
    # Validate auth (use header or request body, email from header)
    auth_key = x_auth_key or request.auth_key
    if not auth_key or not validate_auth_key(auth_key, x_auth_email):
        get_audit_logger().log_auth_fail(endpoint="/chunks/status")
        raise HTTPException(status_code=401, detail="Invalid or missing auth key")

    try:
        chroma = ChromaDBService()
        collection = chroma.client.get_collection(name=request.collection)

        # Get all chunks for the specified file paths
        # ChromaDB doesn't support direct metadata updates, so we need to:
        # 1. Query for chunks matching the file paths
        # 2. Delete them
        # 3. Re-add with updated metadata

        updated_count = 0
        now = datetime.now(timezone.utc)

        for file_path in request.file_paths:
            # Query for chunks from this file
            results = collection.get(
                where={"file_path": {"$eq": file_path}},
                include=["documents", "metadatas", "embeddings"],
            )

            if not results["ids"]:
                continue

            # Update metadata for each chunk
            for i, chunk_id in enumerate(results["ids"]):
                metadata = results["metadatas"][i]
                metadata["status"] = request.status
                if request.status == "deleted":
                    # Use ISO format with Z suffix for RFC3339 compatibility
                    metadata["deleted_at"] = now.strftime("%Y-%m-%dT%H:%M:%SZ")
                    if request.deleted_in_commit:
                        metadata["deleted_in_commit"] = request.deleted_in_commit
                else:
                    # Restoring - clear deletion fields
                    metadata.pop("deleted_at", None)
                    metadata.pop("deleted_in_commit", None)

            # Delete and re-add with updated metadata
            collection.delete(ids=results["ids"])
            collection.add(
                ids=results["ids"],
                documents=results["documents"],
                metadatas=results["metadatas"],
                embeddings=results["embeddings"],
            )
            updated_count += len(results["ids"])

        return ChunkStatusResponse(updated_count=updated_count)

    except Exception as e:
        return ChunkStatusResponse(updated_count=0, error=str(e))


# --- Purge Endpoint (Protected) ---


@app.delete("/chunks/purge", response_model=PurgeResponse)
async def purge_deleted_chunks(
    request: PurgeRequest,
    x_auth_key: Optional[str] = Header(None),
    x_auth_email: Optional[str] = Header(None),
):
    """Permanently remove deleted chunks older than specified date.

    Protected endpoint - requires X-Auth-Key header.
    """
    # Validate auth (use header or request body, email from header)
    auth_key = x_auth_key or request.auth_key
    if not auth_key or not validate_auth_key(auth_key, x_auth_email):
        get_audit_logger().log_auth_fail(endpoint="/chunks/purge")
        return PurgeResponse(chunks_found=0, chunks_removed=0, error="Unauthorized")

    try:
        collection_name = request.collection
        if not collection_name:
            return PurgeResponse(
                chunks_found=0, chunks_removed=0, error="Collection required"
            )

        # Get or create collection
        try:
            chroma = ChromaDBService()
            collection = chroma.client.get_collection(name=collection_name)
        except Exception:
            return PurgeResponse(
                chunks_found=0,
                chunks_removed=0,
                error=f"Collection '{collection_name}' not found",
            )

        # Query for deleted chunks
        results = collection.get(
            where={"status": {"$eq": "deleted"}}, include=["metadatas"]
        )

        if not results or not results["ids"]:
            return PurgeResponse(chunks_found=0, chunks_removed=0)

        # Filter by cutoff date if provided
        chunks_to_remove = []
        cutoff_date = None
        if request.cutoff_date:
            from datetime import datetime

            try:
                cutoff_date = datetime.fromisoformat(
                    request.cutoff_date.replace("Z", "+00:00")
                )
            except ValueError:
                pass

        for i, chunk_id in enumerate(results["ids"]):
            metadata = results["metadatas"][i] if results["metadatas"] else {}
            deleted_at_str = metadata.get("deleted_at", "")

            # If cutoff date specified, filter by age
            if cutoff_date and deleted_at_str:
                try:
                    deleted_at = datetime.fromisoformat(
                        deleted_at_str.replace("Z", "+00:00")
                    )
                    if deleted_at > cutoff_date:
                        continue  # Skip, not old enough
                except ValueError:
                    pass  # Include if can't parse date

            chunks_to_remove.append(chunk_id)

        chunks_found = len(chunks_to_remove)

        # Aggregate per-file details for --list output
        file_info_map = {}  # file_path -> {chunk_count, deleted_at}
        for i, chunk_id in enumerate(results["ids"]):
            if chunk_id not in chunks_to_remove:
                continue
            metadata = results["metadatas"][i] if results["metadatas"] else {}
            file_path = metadata.get("file_path", "unknown")
            deleted_at_str = metadata.get("deleted_at", "")

            if file_path not in file_info_map:
                file_info_map[file_path] = {
                    "chunk_count": 0,
                    "deleted_at": deleted_at_str,
                }
            file_info_map[file_path]["chunk_count"] += 1

        # Convert to list of DeletedFileInfo
        files = [
            DeletedFileInfo(
                file_path=fp,
                chunk_count=info["chunk_count"],
                deleted_at=info["deleted_at"],
            )
            for fp, info in file_info_map.items()
        ]

        # If dry run, return count and file details
        if request.dry_run:
            return PurgeResponse(
                chunks_found=chunks_found, chunks_removed=0, bytes_freed=0, files=files
            )

        # Actually delete the chunks
        if chunks_to_remove:
            collection.delete(ids=chunks_to_remove)
            # Audit log the purge operation
            get_audit_logger().log_cleanup(
                repo=collection_name, purged=len(chunks_to_remove)
            )

        return PurgeResponse(
            chunks_found=chunks_found,
            chunks_removed=len(chunks_to_remove),
            bytes_freed=0,  # ChromaDB doesn't report bytes
            files=files,
        )

    except Exception as e:
        return PurgeResponse(chunks_found=0, chunks_removed=0, error=str(e))


# --- Stats Endpoint (Public) ---


@app.get("/stats/{collection_name}", response_model=StatsResponse)
async def get_collection_stats(collection_name: str):
    """Get statistics for a collection.

    Public endpoint - no authentication required.
    """
    try:
        chroma = ChromaDBService()
        try:
            collection = chroma.client.get_collection(name=collection_name)
        except Exception:
            return StatsResponse(
                active_chunks=0,
                deleted_chunks=0,
                total_chunks=0,
                overhead_percent=0.0,
                error=f"Collection '{collection_name}' not found",
            )

        # Get all chunks and count by status
        results = collection.get(include=["metadatas"])

        active_count = 0
        deleted_count = 0

        if results and results["ids"]:
            for i, _ in enumerate(results["ids"]):
                metadata = results["metadatas"][i] if results["metadatas"] else {}
                status = metadata.get("status", "active")
                if status == "deleted":
                    deleted_count += 1
                else:
                    active_count += 1

        total = active_count + deleted_count
        overhead = (deleted_count / total * 100) if total > 0 else 0.0

        return StatsResponse(
            active_chunks=active_count,
            deleted_chunks=deleted_count,
            total_chunks=total,
            overhead_percent=round(overhead, 1),
        )

    except Exception as e:
        return StatsResponse(
            active_chunks=0,
            deleted_chunks=0,
            total_chunks=0,
            overhead_percent=0.0,
            error=str(e),
        )


# --- Query Endpoint (Public) ---


@app.post("/query", response_model=QueryResponse)
async def query(request: QueryRequest):
    """Search indexed chunks across one or more collections.

    Public endpoint - no authentication required.
    """
    try:
        # Step 1: Validate request
        top_k = request.top_k or 10
        page = request.page or 1
        page_size = request.page_size or 20

        if page < 1 or page_size < 1:
            raise ValueError("page and page_size must be >= 1")
        if top_k < 1 or top_k > 1000:
            raise ValueError("top_k must be between 1 and 1000")

        # Step 2: Embed the query
        try:
            ollama = OllamaService()
            query_embeddings = ollama.embed(
                model="unclemusclez/jina-embeddings-v2-base-code", texts=[request.query]
            )
            query_embedding = query_embeddings[0]
        except Exception as e:
            raise HTTPException(status_code=503, detail=f"Embedding error: {e}")

        # Step 3: Determine which collections to query
        chroma = ChromaDBService()

        if request.collection:
            # Query specific collection
            collections_to_query = [request.collection]
        else:
            # Query all collections (multi-project)
            try:
                collections_to_query = chroma.list_collections()
            except Exception as e:
                return QueryResponse(
                    results=[],
                    total_count=0,
                    page=page,
                    page_size=page_size,
                    error=f"Failed to list collections: {e}",
                )

        # Step 4: Build metadata filters
        where_filter = None
        filter_conditions = []

        # Handle new languages field (Phase 2B)
        if request.languages and len(request.languages) > 0:
            if len(request.languages) == 1:
                filter_conditions.append({"language": {"$eq": request.languages[0]}})
            else:
                filter_conditions.append({"language": {"$in": request.languages}})

        # Phase 2C: Handle tombstone mode filters
        if request.deleted_only:
            # Show only deleted chunks
            filter_conditions.append({"status": {"$eq": "deleted"}})
        elif request.include_deleted:
            # Include both active and deleted - no status filter
            pass
        else:
            # Default: exclude deleted chunks (only show active)
            filter_conditions.append({"status": {"$eq": "active"}})

        # Handle legacy filters dict (backward compatibility)
        if request.filters:
            for key, value in request.filters.items():
                if key == "file_path":
                    filter_conditions.append({key: {"$contains": value}})
                elif key == "language" and not request.languages:
                    # Only use if languages not specified
                    filter_conditions.append({key: {"$eq": value}})
                else:
                    filter_conditions.append({key: {"$eq": value}})

        # Combine conditions with $and if multiple
        if len(filter_conditions) == 1:
            where_filter = filter_conditions[0]
        elif len(filter_conditions) > 1:
            where_filter = {"$and": filter_conditions}

        # Step 5: Query each collection
        all_results = []

        for collection_name in collections_to_query:
            try:
                results = chroma.query(
                    collection_name=collection_name,
                    query_embedding=query_embedding,
                    top_k=top_k,
                    where=where_filter,
                )

                # Convert results to QueryResult format
                for i, (chunk_id, distance, content, metadata) in enumerate(
                    zip(
                        results["ids"],
                        results["distances"],
                        results["documents"],
                        results["metadatas"],
                    )
                ):
                    # Convert L2 distance to similarity score (0-1 range)
                    # L2 distance is unbounded [0, inf), so we use: 1 / (1 + distance)
                    # This maps: distance=0 → similarity=1, distance=inf → similarity=0
                    similarity = 1.0 / (1.0 + distance)

                    all_results.append(
                        {
                            "id": chunk_id,
                            "content": content,
                            "similarity": similarity,
                            "distance": distance,
                            "metadata": metadata,
                        }
                    )
            except Exception:
                # Collection might not exist or have different embedding dimension, skip it
                continue

        # Step 5.5: Apply path filters (post-query since ChromaDB doesn't support prefix)
        if request.path_prefix or request.exclude_path:
            import re

            filtered_results = []
            for result in all_results:
                file_path = result.get("metadata", {}).get("file_path", "")

                # Check path prefix
                if request.path_prefix and not file_path.startswith(
                    request.path_prefix
                ):
                    continue

                # Check exclusion pattern
                if request.exclude_path:
                    if re.search(request.exclude_path, file_path):
                        continue

                filtered_results.append(result)
            all_results = filtered_results

        # Step 6: Sort by similarity (descending - highest first)
        all_results.sort(key=lambda x: x["similarity"], reverse=True)

        # Step 6.5: Limit to top_k results (important for multi-collection efficiency)
        # This prevents returning hundreds of results when querying many collections
        all_results = all_results[:top_k]

        # Step 7: Apply pagination
        total_count = len(all_results)
        start_idx = (page - 1) * page_size
        end_idx = start_idx + page_size
        paginated_results = all_results[start_idx:end_idx]

        # Step 8: Convert to response format
        response_results = [
            QueryResult(
                chunk_id=r["id"],
                content=r["content"],
                similarity=r["similarity"],
                metadata=r["metadata"],
                distance=r["distance"],
            )
            for r in paginated_results
        ]

        return QueryResponse(
            results=response_results,
            total_count=total_count,
            page=page,
            page_size=page_size,
            error="",
        )

    except HTTPException:
        raise
    except Exception as e:
        return QueryResponse(
            results=[], total_count=0, page=page, page_size=page_size, error=str(e)
        )


# --- Admin Endpoints (Protected) ---


@app.post("/admin/bootstrap")
async def bootstrap(email: str, auth_key: str, request: Request):
    """Create first manager (one-time only)."""
    # Check if already bootstrapped
    guard = get_bootstrap_guard()
    if guard.is_bootstrapped():
        raise HTTPException(
            status_code=403, detail="Bootstrap already completed - endpoint disabled"
        )

    success = create_first_manager(email, auth_key)

    if not success:
        raise HTTPException(
            status_code=400, detail="Bootstrap failed - managers already exist"
        )

    # Mark as bootstrapped
    guard.mark_bootstrapped()

    client_ip = request.client.host if request.client else "unknown"
    get_audit_logger().log_bootstrap(email=email, ip=client_ip)
    return {"status": "ok", "message": f"First manager {email} created"}


@app.post("/admin/add")
async def add_admin(
    email: str,
    auth_key: str,
    request: Request,
    role: str = "manager",
    x_auth_key: Optional[str] = Header(None),
    x_auth_email: Optional[str] = Header(None),
):
    """Add new manager (requires admin auth).

    Args:
        email: Email for the new manager
        auth_key: Auth key for the new manager
        role: Role for new manager ('admin' or 'manager', defaults to 'manager')
    """
    client_ip = request.client.host if request.client else "unknown"
    rate_limiter = get_rate_limiter()

    # Check rate limit
    if rate_limiter.check_auth_rate_limit(client_ip):
        raise HTTPException(
            status_code=429, detail="Too many auth failures. Try again later."
        )

    # Require admin role to add managers
    if not x_auth_key or not validate_admin(x_auth_key, x_auth_email):
        rate_limiter.record_auth_failure(client_ip, "/admin/add")
        raise HTTPException(status_code=403, detail="Admin access required")

    success = add_manager(email, auth_key, role)

    if not success:
        raise HTTPException(status_code=400, detail="Manager already exists")

    get_audit_logger().log_admin_add(email=email)
    return {"status": "ok", "message": f"{role.capitalize()} {email} added"}


@app.get("/admin/list")
async def list_admins(
    request: Request,
    x_auth_key: Optional[str] = Header(None),
    x_auth_email: Optional[str] = Header(None),
):
    """List all managers (requires admin auth)."""
    client_ip = request.client.host if request.client else "unknown"
    rate_limiter = get_rate_limiter()

    # Check rate limit
    if rate_limiter.check_auth_rate_limit(client_ip):
        raise HTTPException(
            status_code=429, detail="Too many auth failures. Try again later."
        )

    # Require admin role to list managers
    if not x_auth_key or not validate_admin(x_auth_key, x_auth_email):
        rate_limiter.record_auth_failure(client_ip, "/admin/list")
        raise HTTPException(status_code=403, detail="Admin access required")

    managers = list_managers()
    return {"managers": managers}


@app.delete("/admin/{email}")
async def remove_admin(
    email: str,
    request: Request,
    x_auth_key: Optional[str] = Header(None),
    x_auth_email: Optional[str] = Header(None),
):
    """Remove manager (requires admin auth)."""
    client_ip = request.client.host if request.client else "unknown"
    rate_limiter = get_rate_limiter()

    # Check rate limit
    if rate_limiter.check_auth_rate_limit(client_ip):
        raise HTTPException(
            status_code=429, detail="Too many auth failures. Try again later."
        )

    # Require admin role to remove managers
    if not x_auth_key or not validate_admin(x_auth_key, x_auth_email):
        rate_limiter.record_auth_failure(client_ip, "/admin/remove")
        raise HTTPException(status_code=403, detail="Admin access required")

    success = remove_manager(email)

    if not success:
        raise HTTPException(status_code=404, detail="Manager not found")

    get_audit_logger().log_admin_remove(email=email)
    return {"status": "ok", "message": f"Manager {email} removed"}


# --- Server startup ---

if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8080, timeout_keep_alive=300)
