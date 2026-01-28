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


# Chunk Status Update (for tombstone mode)
class ChunkStatusRequest(BaseModel):
    auth_key: str
    collection: str
    file_paths: List[str]  # Files whose chunks should be marked deleted
    status: str = "deleted"  # "active" or "deleted"
    deleted_in_commit: Optional[str] = None


class ChunkStatusResponse(BaseModel):
    updated_count: int
    error: Optional[str] = None


# Purge (Cleanup Deleted Chunks)
class PurgeRequest(BaseModel):
    auth_key: Optional[str] = None
    collection: str
    older_than: int = 0  # Days since deletion
    cutoff_date: Optional[str] = None  # RFC3339 date
    dry_run: bool = False


class DeletedFileInfo(BaseModel):
    file_path: str
    chunk_count: int
    deleted_at: str  # RFC3339 format


class PurgeResponse(BaseModel):
    chunks_found: int
    chunks_removed: int
    bytes_freed: int = 0
    files: List[DeletedFileInfo] = []  # Per-file details for --list
    error: Optional[str] = None


# Query
class QueryRequest(BaseModel):
    query: str
    collection: Optional[str] = None
    top_k: int = 10
    filters: Optional[Dict[str, str]] = None
    page: Optional[int] = 1
    page_size: Optional[int] = 20
    # New filter fields for Phase 2B
    languages: Optional[List[str]] = None  # e.g., ["python", "go"]
    path_prefix: Optional[str] = None  # e.g., "internal/"
    exclude_path: Optional[str] = None  # e.g., "vendor|test"
    # Phase 2C: Tombstone mode filters
    include_deleted: bool = False  # Include deleted chunks in results
    deleted_only: bool = False  # Show only deleted chunks


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
