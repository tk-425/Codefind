from .admin import router as admin_router
from .auth import router as auth_router
from .health import router as health_router
from .orgs import router as orgs_router

__all__ = ["admin_router", "auth_router", "health_router", "orgs_router"]
