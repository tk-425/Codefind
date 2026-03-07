from __future__ import annotations

from contextvars import ContextVar, Token
from dataclasses import dataclass, replace
from time import perf_counter
from uuid import uuid4

from fastapi import Request, Response


REQUEST_ID_HEADER = "X-Request-ID"


@dataclass(slots=True, frozen=True)
class RequestContext:
    request_id: str
    method: str
    path: str
    client_ip: str | None
    org_id: str | None = None
    user_id: str | None = None
    user_role: str | None = None


_request_context: ContextVar[RequestContext | None] = ContextVar(
    "request_context",
    default=None,
)


def _client_ip(request: Request) -> str | None:
    forwarded_for = request.headers.get("x-forwarded-for")
    if forwarded_for:
        return forwarded_for.split(",")[0].strip()
    if request.client is None:
        return None
    return request.client.host


def get_request_context() -> RequestContext | None:
    return _request_context.get()


def set_request_identity(*, org_id: str, user_id: str, user_role: str) -> None:
    context = get_request_context()
    if context is None:
        return
    _request_context.set(
        replace(
            context,
            org_id=org_id,
            user_id=user_id,
            user_role=user_role,
        )
    )


async def request_context_middleware(request: Request, call_next) -> Response:
    request_id = request.headers.get(REQUEST_ID_HEADER, str(uuid4()))
    context = RequestContext(
        request_id=request_id,
        method=request.method,
        path=request.url.path,
        client_ip=_client_ip(request),
    )
    request.state.request_context = context
    token: Token[RequestContext | None] = _request_context.set(context)
    start = perf_counter()
    try:
        response = await call_next(request)
    finally:
        _request_context.reset(token)
    response.headers[REQUEST_ID_HEADER] = request_id
    response.headers["X-Process-Time"] = f"{perf_counter() - start:.6f}"
    return response
