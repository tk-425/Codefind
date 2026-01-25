from fastapi import FastAPI, HTTPException, Header
from datetime import datetime, timezone
import requests
import uuid
import os
from typing import Optional
from transformers import AutoTokenizer
from dotenv import load_dotenv

from models import (
    TokenizeRequest, TokenizeResponse,
    EmbedRequest, EmbedResponse,
    IndexRequest, IndexResponse,
    QueryRequest, QueryResponse,
    HealthResponse
)
from auth import validate_auth_key, create_first_manager, add_manager, remove_manager, list_managers
from services.chromadb_service import ChromaDBService
from services.ollama_service import OllamaService

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
        error=error_msg
    )

# --- Tokenize Endpoint ---

@app.post("/tokenize", response_model=TokenizeResponse)
async def tokenize(request: TokenizeRequest):
    """Count tokens using transformers library."""
    try:
        # Load tokenizer for the model (defaults to BERT tokenizer for embedding models)
        tokenizer = AutoTokenizer.from_pretrained("bert-base-uncased", trust_remote_code=True)

        token_counts = []
        for text in request.input:
            tokens = tokenizer.encode(text)
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
async def index_chunks(request: IndexRequest, x_auth_key: Optional[str] = Header(None)):
    """Index chunks: embed via Ollama and store in ChromaDB."""

    # Step 1: Authenticate
    if not x_auth_key or not validate_auth_key(x_auth_key):
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
            model=request.chunks[0].metadata.model_id,
            texts=texts
        )
    except Exception as e:
        raise HTTPException(status_code=503, detail=f"Ollama error: {e}")

    # Step 4: Store in ChromaDB
    try:
        ids = []
        documents = []
        metadatas = []

        for chunk in request.chunks:
            chunk_id = str(uuid.uuid4())
            ids.append(chunk_id)
            documents.append(chunk.content)

            # Convert metadata to dict with JSON serialization (datetime -> ISO string, exclude None values)
            metadata_dict = chunk.metadata.model_dump(mode='json', exclude_none=True)
            metadatas.append(metadata_dict)

        # Add to collection
        collection.add(
            ids=ids,
            documents=documents,
            embeddings=embeddings,
            metadatas=metadatas
        )

        return IndexResponse(
            inserted_count=len(ids),
            updated_count=0,
            error=""
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Storage error: {e}")

# --- Query Endpoint (Public) ---

@app.post("/query", response_model=QueryResponse)
async def query(request: QueryRequest):
    """Search indexed chunks (public endpoint)."""
    try:
        # TODO: Implement ChromaDB query logic
        # For now, return mock results

        return QueryResponse(
            results=[],
            total_count=0,
            page=request.page or 1,
            page_size=request.page_size or 20
        )

    except Exception as e:
        return QueryResponse(
            results=[],
            total_count=0,
            page=request.page or 1,
            page_size=request.page_size or 20,
            error=str(e)
        )

# --- Admin Endpoints (Protected) ---

@app.post("/admin/bootstrap")
async def bootstrap(email: str, auth_key: str):
    """Create first manager (one-time only)."""
    success = create_first_manager(email, auth_key)

    if not success:
        raise HTTPException(status_code=400, detail="Bootstrap failed - managers already exist")

    return {"status": "ok", "message": f"First manager {email} created"}

@app.post("/admin/add")
async def add_admin(email: str, x_auth_key: Optional[str] = Header(None)):
    """Add new manager (requires auth)."""

    if not x_auth_key or not validate_auth_key(x_auth_key):
        raise HTTPException(status_code=401, detail="Invalid auth key")

    # TODO: Generate new auth key and return to caller
    success = add_manager(email, "new-auth-key-placeholder")

    if not success:
        raise HTTPException(status_code=400, detail="Manager already exists")

    return {"status": "ok", "message": f"Manager {email} added"}

@app.get("/admin/list")
async def list_admins(x_auth_key: Optional[str] = Header(None)):
    """List all managers (requires auth)."""

    if not x_auth_key or not validate_auth_key(x_auth_key):
        raise HTTPException(status_code=401, detail="Invalid auth key")

    managers = list_managers()
    return {"managers": managers}

@app.delete("/admin/{email}")
async def remove_admin(email: str, x_auth_key: Optional[str] = Header(None)):
    """Remove manager (requires auth)."""

    if not x_auth_key or not validate_auth_key(x_auth_key):
        raise HTTPException(status_code=401, detail="Invalid auth key")

    success = remove_manager(email)

    if not success:
        raise HTTPException(status_code=404, detail="Manager not found")

    return {"status": "ok", "message": f"Manager {email} removed"}

# --- Server startup ---

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8080)
