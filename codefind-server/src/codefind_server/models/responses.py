from datetime import UTC, datetime

from pydantic import BaseModel, Field


class HealthResponse(BaseModel):
    status: str
    ollama_status: str = "unknown"
    qdrant_status: str = "unknown"
    timestamp: datetime = Field(default_factory=lambda: datetime.now(UTC))
