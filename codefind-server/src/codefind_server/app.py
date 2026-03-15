from contextlib import asynccontextmanager

from fastapi import APIRouter, FastAPI

from .adapters.qdrant import QdrantAdapter
from .config import get_settings
from .logging import configure_logging
from .middleware import request_context_middleware
from .middleware.rate_limit import RateLimitMiddleware
from .routes.admin import router as admin_router
from .routes.auth import router as auth_router
from .routes.collections import router as collections_router
from .routes.health import router as health_router
from .routes.index import router as index_router
from .routes.orgs import router as orgs_router
from .routes.query import router as query_router
from .routes.stats import router as stats_router
from .routes.tokenize import router as tokenize_router
from .security import init_sentry, request_body_limit_middleware
from .services import IndexJobLockManager, OllamaService, SparseEmbeddingService, TokenizerService


@asynccontextmanager
async def lifespan(app: FastAPI):
    settings = get_settings()
    configure_logging(settings=settings)
    init_sentry(settings)
    if settings.vector_store != "qdrant":
        raise RuntimeError(f"Unsupported VECTOR_STORE: {settings.vector_store}")
    vector_store = QdrantAdapter(url=settings.qdrant_url)
    ollama = OllamaService(
        base_url=settings.ollama_url,
        embed_model=settings.ollama_embed_model,
        timeout_seconds=settings.ollama_embed_timeout_seconds,
        max_attempts=settings.ollama_embed_max_attempts,
        retry_backoff_seconds=settings.ollama_embed_retry_backoff_seconds,
    )
    sparse_embeddings = (
        SparseEmbeddingService(
            model_name=settings.sparse_embed_model,
            cache_dir=settings.sparse_embed_cache_dir,
            batch_size=settings.sparse_embed_batch_size,
        )
        if settings.sparse_retrieval_enabled
        else None
    )
    tokenizer = TokenizerService(model_name=settings.tokenizer_model)
    index_locks = IndexJobLockManager()
    app.state.settings = settings
    app.state.vector_store = vector_store
    app.state.ollama = ollama
    app.state.sparse_embeddings = sparse_embeddings
    app.state.tokenizer = tokenizer
    app.state.index_locks = index_locks
    yield
    await ollama.close()
    await vector_store.close()


def create_app() -> FastAPI:
    app = FastAPI(title="Code-Find Server", lifespan=lifespan)
    settings = get_settings()
    app.middleware("http")(request_context_middleware)
    app.middleware("http")(request_body_limit_middleware(settings.max_request_body_bytes))
    app.add_middleware(RateLimitMiddleware, settings=settings)
    api_router = APIRouter(prefix="/api")
    api_router.include_router(admin_router)
    api_router.include_router(auth_router)
    api_router.include_router(collections_router)
    api_router.include_router(health_router)
    api_router.include_router(index_router)
    api_router.include_router(orgs_router)
    api_router.include_router(query_router)
    api_router.include_router(stats_router)
    api_router.include_router(tokenize_router)
    app.include_router(api_router)
    return app
