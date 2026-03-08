from typing import Literal

from pydantic import BaseModel, Field


class HealthRequest(BaseModel):
    detail: bool = Field(default=False)


class CreateOrganizationInvitationRequest(BaseModel):
    email_address: str
    role: Literal["org:admin", "org:member"] = "org:member"
