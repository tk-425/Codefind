from __future__ import annotations

from fastapi import APIRouter, Request
import httpx

from ..models.responses import HealthResponse


router = APIRouter(tags=["health"])


@router.get("/health", response_model=HealthResponse)
async def health(request: Request) -> HealthResponse:
    vector_store = request.app.state.vector_store
    settings = request.app.state.settings

    qdrant_ok = await vector_store.healthcheck()
    ollama_ok = await _check_ollama(settings.ollama_url)
    status_value = "ok" if qdrant_ok and ollama_ok else "degraded"
    return HealthResponse(
        status=status_value,
        ollama_status="ok" if ollama_ok else "unavailable",
        qdrant_status="ok" if qdrant_ok else "unavailable",
    )


async def _check_ollama(base_url: str) -> bool:
    try:
        async with httpx.AsyncClient(timeout=2.0) as client:
            response = await client.get(f"{base_url.rstrip('/')}/api/version")
            response.raise_for_status()
    except httpx.HTTPError:
        return False
    return True
