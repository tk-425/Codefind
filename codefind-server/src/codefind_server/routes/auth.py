from __future__ import annotations

from urllib.parse import urlencode, urljoin, urlparse

from fastapi import APIRouter, HTTPException, Query, status
from fastapi.responses import RedirectResponse

from ..config import get_settings

CLI_CALLBACK_PATH = "/callback"
CLI_CALLBACK_HOST = "127.0.0.1"

router = APIRouter(prefix="/auth", tags=["auth"])


def validate_cli_redirect_uri(value: str) -> str:
    parsed = urlparse(value)
    if (
        parsed.scheme != "http"
        or parsed.hostname != CLI_CALLBACK_HOST
        or parsed.path != CLI_CALLBACK_PATH
        or not parsed.port
        or parsed.params
        or parsed.fragment
        or parsed.username
        or parsed.password
    ):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="redirect_uri must match http://127.0.0.1:<port>/callback",
        )
    return value


def build_signin_redirect(web_app_url: str, redirect_uri: str | None) -> str:
    sign_in_url = urljoin(web_app_url.rstrip("/") + "/", "signin")
    params = []
    if redirect_uri is not None:
        params.append(("redirect_uri", redirect_uri))
    if not params:
        return sign_in_url
    return f"{sign_in_url}?{urlencode(params)}"


@router.get("/signin")
async def auth_signin(
    redirect_uri: str | None = Query(default=None),
):
    settings = get_settings()
    validated_redirect_uri = (
        validate_cli_redirect_uri(redirect_uri) if redirect_uri is not None else None
    )
    target = build_signin_redirect(settings.web_app_url, validated_redirect_uri)
    response = RedirectResponse(url=target, status_code=status.HTTP_307_TEMPORARY_REDIRECT)
    response.headers["Cache-Control"] = "no-store"
    return response
