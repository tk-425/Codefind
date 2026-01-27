from pydantic import BaseModel
from typing import List, Optional, Dict
from datetime import datetime


# ChunkMetadata model
class ChunkMetadata(BaseModel):
    repo_id: str
    project_name: str
    file_path: str
    language: str
    start_line: int
    end_line: int
    content_hash: str
    chunk_tokens: int
    model_id: str
    indexed_at: datetime
    symbol_name: Optional[str] = None
    symbol_kind: Optional[str] = None
    is_split: bool = False
    split_index: Optional[int] = None
    status: str = "active"  # "active" or "deleted"
    deleted_at: Optional[datetime] = None
    deleted_in_commit: Optional[str] = None


# Chunk model
class Chunk(BaseModel):
    id: Optional[str] = None  # Stable chunk ID (SHA256-based, optional)
    content: str
    metadata: ChunkMetadata


# Tokenize
class TokenizeRequest(BaseModel):
    model: str
    input: List[str]


class TokenizeResponse(BaseModel):
    tokens: List[int]
    error: Optional[str] = None


# Embed
class EmbedRequest(BaseModel):
    model: str
    input: List[str]


class EmbedResponse(BaseModel):
    embeddings: List[List[float]]
    error: Optional[str] = None


# Index
class IndexRequest(BaseModel):
    auth_key: str
    chunks: List[Chunk]
    collection: str


class IndexResponse(BaseModel):
    inserted_count: int
    updated_count: int
    error: Optional[str] = None


# Query
class QueryRequest(BaseModel):
    query: str
    collection: Optional[str] = None
    top_k: int = 10
    filters: Optional[Dict[str, str]] = None
    page: Optional[int] = 1
    page_size: Optional[int] = 20


class QueryResult(BaseModel):
    chunk_id: str
    content: str
    metadata: ChunkMetadata
    distance: float
    similarity: float


class QueryResponse(BaseModel):
    results: List[QueryResult]
    total_count: int
    page: int
    page_size: int
    error: Optional[str] = None


# Health
class HealthResponse(BaseModel):
    status: str
    ollama_status: str
    chromadb_status: str
    timestamp: datetime
    error: Optional[str] = None
