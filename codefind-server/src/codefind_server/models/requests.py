from typing import Literal

from pydantic import BaseModel, Field


class HealthRequest(BaseModel):
    detail: bool = Field(default=False)


class CreateOrganizationInvitationRequest(BaseModel):
    email_address: str
    role: Literal["org:admin", "org:member"] = "org:member"


class QueryRequest(BaseModel):
    query_text: str = Field(min_length=1, max_length=2000)
    repo_id: str | None = None
    project: str | None = Field(default=None, max_length=255)
    language: str | None = Field(default=None, max_length=128)
    page: int = Field(default=1, ge=1)
    page_size: int = Field(default=10, ge=1, le=50)
    top_k: int = Field(default=10, ge=1)


class TokenizeRequest(BaseModel):
    text: str = Field(min_length=1, max_length=2000)


class IndexRequest(BaseModel):
    repo_id: str = Field(min_length=1, max_length=255)
    repo_path: str = Field(min_length=1, max_length=4096)
    force: bool = False
    window: bool = False
    retry_lsp: bool = False
    concurrency: int = Field(default=1, ge=1, le=64)


class ChunkStatusUpdateRequest(BaseModel):
    repo_id: str = Field(min_length=1, max_length=255)
    chunk_ids: list[str] = Field(min_length=1)
    status: Literal["active", "tombstoned"]


class ChunkPurgeRequest(BaseModel):
    repo_id: str = Field(min_length=1, max_length=255)
    older_than_days: int = Field(ge=1, le=3650)
