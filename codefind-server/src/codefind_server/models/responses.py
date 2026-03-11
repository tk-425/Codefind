from datetime import UTC, datetime

from pydantic import BaseModel, Field


class HealthResponse(BaseModel):
    status: str
    ollama_status: str = "unknown"
    qdrant_status: str = "unknown"
    timestamp: datetime = Field(default_factory=lambda: datetime.now(UTC))


class OrgSummary(BaseModel):
    organization_id: str
    organization_name: str | None = None
    organization_slug: str | None = None
    role: str


class OrgListResponse(BaseModel):
    data: list[OrgSummary]
    total_count: int


class OrganizationMemberResponse(BaseModel):
    user_id: str
    membership_id: str | None = None
    role: str
    first_name: str | None = None
    last_name: str | None = None
    email_address: str | None = None
    profile_image_url: str | None = None


class OrganizationMemberListResponse(BaseModel):
    data: list[OrganizationMemberResponse]
    total_count: int


class OrganizationInvitationResponse(BaseModel):
    id: str
    invitation_id: str
    email_address: str
    role: str
    status: str
    organization_id: str
    created_at: int | None = None
    updated_at: int | None = None
    expires_at: int | None = None
    inviter_user_id: str | None = None


class OrganizationInvitationListResponse(BaseModel):
    data: list[OrganizationInvitationResponse]
    total_count: int


class CollectionSummaryResponse(BaseModel):
    repo_id: str


class CollectionListResponse(BaseModel):
    data: list[CollectionSummaryResponse]
    total_count: int


class RepoStatsResponse(BaseModel):
    repo_id: str
    chunk_count: int


class StatsResponse(BaseModel):
    repo_id: str | None = None
    repo_count: int
    chunk_count: int
    active_chunks: int
    deleted_chunks: int
    total_chunks: int
    overhead_percent: float
    repos: list[RepoStatsResponse]


class QueryResultResponse(BaseModel):
    id: str
    score: float
    repo_id: str
    project: str | None = None
    language: str | None = None
    path: str | None = None
    snippet: str | None = None
    content: str | None = None
    page: int | None = None
    start_line: int | None = None
    end_line: int | None = None


class QueryResponse(BaseModel):
    data: list[QueryResultResponse]
    total_count: int
    page: int
    page_size: int
    has_more: bool


class TokenizeResponse(BaseModel):
    model: str
    tokens: list[str]
    token_count: int


class IndexResponse(BaseModel):
    status: str
    repo_id: str
    indexed_count: int = 0
    accepted: bool = False
    detail: str | None = None


class ChunkStatusUpdateResponse(BaseModel):
    status: str
    repo_id: str
    updated_count: int
    detail: str | None = None


class TombstonedChunkSummaryResponse(BaseModel):
    path: str
    chunk_count: int
    tombstoned_at: str | None = None


class TombstonedChunkListResponse(BaseModel):
    status: str
    repo_id: str
    found_count: int
    files: list[TombstonedChunkSummaryResponse]
    detail: str | None = None


class ChunkPurgeResponse(BaseModel):
    status: str
    repo_id: str
    found_count: int = 0
    purged_count: int
    files: list[TombstonedChunkSummaryResponse] = []
    detail: str | None = None


class RepoClearResponse(BaseModel):
    status: str
    repo_id: str
    cleared: bool
    detail: str | None = None
