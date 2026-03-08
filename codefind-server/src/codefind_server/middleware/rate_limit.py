from __future__ import annotations

import asyncio
from collections import defaultdict, deque
from collections.abc import Awaitable, Callable
from dataclasses import dataclass
from time import monotonic

from fastapi import Request
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.responses import JSONResponse

from ..config import Settings
from .auth import extract_bearer, extract_org_context, verify_clerk_token


@dataclass(slots=True, frozen=True)
class RateLimitIdentity:
    org_id: str | None = None
    user_id: str | None = None


IdentityResolver = Callable[[Request], Awaitable[RateLimitIdentity | None]]


class InMemoryRateLimiter:
    def __init__(self) -> None:
        self._lock = asyncio.Lock()
        self._requests: dict[str, deque[float]] = defaultdict(deque)

    async def allow(self, *, key: str, limit: int, window_seconds: int) -> tuple[bool, int]:
        async with self._lock:
            now = monotonic()
            window = self._requests[key]
            cutoff = now - window_seconds
            while window and window[0] <= cutoff:
                window.popleft()
            if len(window) >= limit:
                retry_after = max(1, int(window_seconds - (now - window[0])))
                return False, retry_after
            window.append(now)
            return True, 0


async def default_identity_resolver(request: Request) -> RateLimitIdentity | None:
    authorization = request.headers.get("authorization")
    if not authorization:
        return None
    try:
        token = extract_bearer(authorization)
        claims = verify_clerk_token(token, request.app.state.settings)
    except Exception:
        return None
    org_id, _org_role = extract_org_context(claims)
    user_id = claims.get("sub")
    if not org_id or not user_id:
        return None
    return RateLimitIdentity(org_id=org_id, user_id=user_id)


def _client_ip(request: Request) -> str:
    forwarded_for = request.headers.get("x-forwarded-for")
    if forwarded_for:
        return forwarded_for.split(",")[0].strip()
    if request.client is None:
        return "unknown"
    return request.client.host


def _category(path: str) -> str:
    if path == "/health":
        return "health"
    if path.startswith("/auth"):
        return "auth"
    if path.startswith("/admin"):
        return "admin"
    if path.startswith("/query") or path.startswith("/tokenize") or path.startswith("/search"):
        return "query"
    return "default"


def _limits(settings: Settings, category: str) -> int:
    if category == "auth":
        return settings.rate_limit_auth_per_window
    if category == "admin":
        return settings.rate_limit_admin_per_window
    if category == "query":
        return settings.rate_limit_query_per_window
    return settings.rate_limit_default_per_window


class RateLimitMiddleware(BaseHTTPMiddleware):
    def __init__(
        self,
        app,
        *,
        settings: Settings,
        limiter: InMemoryRateLimiter | None = None,
        identity_resolver: IdentityResolver = default_identity_resolver,
    ) -> None:
        super().__init__(app)
        self._settings = settings
        self._limiter = limiter or InMemoryRateLimiter()
        self._identity_resolver = identity_resolver

    async def dispatch(self, request: Request, call_next):
        category = _category(request.url.path)
        if category == "health":
            return await call_next(request)

        key = await self._key(request, category)
        allowed, retry_after = await self._limiter.allow(
            key=key,
            limit=_limits(self._settings, category),
            window_seconds=self._settings.rate_limit_window_seconds,
        )
        if not allowed:
            return JSONResponse(
                status_code=429,
                content={"detail": "Rate limit exceeded."},
                headers={"Retry-After": str(retry_after)},
            )
        return await call_next(request)

    async def _key(self, request: Request, category: str) -> str:
        client_ip = _client_ip(request)
        if category in {"auth", "admin", "default"}:
            return f"{category}:{client_ip}"

        identity = await self._identity_resolver(request)
        if identity is not None and identity.org_id and identity.user_id:
            return f"{category}:{identity.org_id}:{identity.user_id}"
        return f"{category}:{client_ip}"
