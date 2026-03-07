from __future__ import annotations

from collections.abc import Mapping
from typing import Any

from fastapi import HTTPException, Request, Response, status
import sentry_sdk
from sentry_sdk.integrations.fastapi import FastApiIntegration
from sentry_sdk.integrations.starlette import StarletteIntegration

from .config import Settings

REDACTED = "[REDACTED]"
SENSITIVE_HEADERS = {"authorization", "cookie", "x-api-key"}
SENSITIVE_KEYS = {
    "access_token",
    "authorization",
    "bearer",
    "clerk_secret_key",
    "code",
    "cookie",
    "embedding",
    "file_content",
    "jwt",
    "password",
    "query",
    "query_text",
    "secret",
    "source",
    "token",
}


def redact_value(value: Any) -> Any:
    if isinstance(value, Mapping):
        return redact_mapping(value)
    if isinstance(value, list):
        return [redact_value(item) for item in value]
    return value


def redact_mapping(mapping: Mapping[str, Any]) -> dict[str, Any]:
    redacted: dict[str, Any] = {}
    for key, value in mapping.items():
        lowered = key.lower()
        if lowered in SENSITIVE_HEADERS or lowered in SENSITIVE_KEYS:
            redacted[key] = REDACTED
            continue
        redacted[key] = redact_value(value)
    return redacted


def scrub_sentry_event(event: dict[str, Any], _hint: dict[str, Any]) -> dict[str, Any]:
    request_data = event.get("request")
    if isinstance(request_data, Mapping):
        scrubbed_request = redact_mapping(request_data)
        headers = request_data.get("headers")
        if isinstance(headers, Mapping):
            scrubbed_request["headers"] = redact_mapping(headers)
        data = request_data.get("data")
        if isinstance(data, Mapping):
            scrubbed_request["data"] = redact_mapping(data)
        event["request"] = scrubbed_request

    user = event.get("user")
    if isinstance(user, Mapping):
        safe_user = dict(user)
        safe_user.pop("ip_address", None)
        event["user"] = safe_user

    contexts = event.get("contexts")
    if isinstance(contexts, Mapping):
        event["contexts"] = redact_mapping(contexts)

    extra = event.get("extra")
    if isinstance(extra, Mapping):
        event["extra"] = redact_mapping(extra)

    breadcrumbs = event.get("breadcrumbs", {})
    if isinstance(breadcrumbs, Mapping):
        values = breadcrumbs.get("values")
        if isinstance(values, list):
            for breadcrumb in values:
                data = breadcrumb.get("data")
                if isinstance(data, Mapping):
                    breadcrumb["data"] = redact_mapping(data)

    return event


def init_sentry(settings: Settings) -> None:
    if not settings.sentry_dsn:
        return

    sentry_sdk.init(
        dsn=settings.sentry_dsn,
        environment=settings.environment,
        send_default_pii=False,
        traces_sample_rate=settings.sentry_traces_sample_rate,
        before_send=scrub_sentry_event,
        integrations=[
            StarletteIntegration(
                transaction_style="endpoint",
                failed_request_status_codes=[403, range(500, 600)],
            ),
            FastApiIntegration(transaction_style="endpoint"),
        ],
    )


def request_body_limit_middleware(max_bytes: int):
    async def middleware(request: Request, call_next) -> Response:
        content_length = request.headers.get("content-length")
        if content_length is not None and int(content_length) > max_bytes:
            raise HTTPException(
                status_code=status.HTTP_413_REQUEST_ENTITY_TOO_LARGE,
                detail="Request body too large.",
            )

        body = await request.body()
        if len(body) > max_bytes:
            raise HTTPException(
                status_code=status.HTTP_413_REQUEST_ENTITY_TOO_LARGE,
                detail="Request body too large.",
            )
        return await call_next(request)

    return middleware
