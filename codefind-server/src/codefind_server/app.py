from contextlib import asynccontextmanager

from fastapi import FastAPI

from .adapters.qdrant import QdrantAdapter
from .config import get_settings
from .logging import configure_logging
from .middleware import request_context_middleware
from .middleware.rate_limit import RateLimitMiddleware
from .routes.health import router as health_router
from .security import init_sentry, request_body_limit_middleware


@asynccontextmanager
async def lifespan(app: FastAPI):
    settings = get_settings()
    configure_logging(settings=settings)
    init_sentry(settings)
    if settings.vector_store != "qdrant":
        raise RuntimeError(f"Unsupported VECTOR_STORE: {settings.vector_store}")
    vector_store = QdrantAdapter(url=settings.qdrant_url)
    app.state.settings = settings
    app.state.vector_store = vector_store
    yield
    await vector_store.close()


def create_app() -> FastAPI:
    app = FastAPI(title="Code-Find Server", lifespan=lifespan)
    settings = get_settings()
    app.middleware("http")(request_context_middleware)
    app.middleware("http")(request_body_limit_middleware(settings.max_request_body_bytes))
    app.add_middleware(RateLimitMiddleware, settings=settings)
    app.include_router(health_router)
    return app
