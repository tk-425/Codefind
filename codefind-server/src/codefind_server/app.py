from contextlib import asynccontextmanager

from fastapi import FastAPI

from .config import get_settings
from .logging import configure_logging
from .routes.health import router as health_router


@asynccontextmanager
async def lifespan(app: FastAPI):
    settings = get_settings()
    configure_logging(settings=settings)
    app.state.settings = settings
    yield


def create_app() -> FastAPI:
    app = FastAPI(title="Code-Find Server", lifespan=lifespan)
    app.include_router(health_router)
    return app
