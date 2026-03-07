from pydantic import BaseModel


class HealthRequest(BaseModel):
    detail: bool = False
