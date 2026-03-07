from pydantic import BaseModel


class HealthResponse(BaseModel):
    status: str
    ollama_status: str = "unknown"
    qdrant_status: str = "unknown"
