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
