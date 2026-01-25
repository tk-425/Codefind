from fastapi import APIRouter, HTTPException
from pydantic import BaseModel
from typing import List
import uuid

from services.chromadb_service import ChromaDBService
from services.ollama_service import OllamaService

router = APIRouter()

class ChunkMetadata(BaseModel):
    repo_id: str
    project_name: str
    file_path: str
    language: str
    start_line: int
    end_line: int
    content_hash: str
    model_id: str
    indexed_at: str
    chunk_tokens: int
    status: str = "active"
    symbol_name: str = ""
    symbol_kind: str = ""
    is_split: bool = False
    split_index: int = 0

class ChunkRequest(BaseModel):
    content: str
    metadata: ChunkMetadata

class IndexRequest(BaseModel):
    auth_key: str
    chunks: List[ChunkRequest]
    collection: str

class IndexResponse(BaseModel):
    inserted_count: int
    updated_count: int
    error: str = ""

@router.post("/index", response_model=IndexResponse)
async def index_chunks(request: IndexRequest):
    """Index chunks: embed via Ollama and store in ChromaDB"""

    # Step 1: Authenticate
    # TODO: Implement auth_key validation (Phase 3A)

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

            # Convert metadata to dict (using model_dump instead of deprecated dict())
            metadata_dict = chunk.metadata.model_dump()
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