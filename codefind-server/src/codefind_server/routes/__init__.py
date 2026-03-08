from .admin import router as admin_router
from .auth import router as auth_router
from .collections import router as collections_router
from .health import router as health_router
from .orgs import router as orgs_router
from .query import router as query_router
from .stats import router as stats_router
from .tokenize import router as tokenize_router

__all__ = [
    "admin_router",
    "auth_router",
    "collections_router",
    "health_router",
    "orgs_router",
    "query_router",
    "stats_router",
    "tokenize_router",
]
