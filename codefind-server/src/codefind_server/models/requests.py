from pydantic import BaseModel, Field


class HealthRequest(BaseModel):
    detail: bool = Field(default=False)
