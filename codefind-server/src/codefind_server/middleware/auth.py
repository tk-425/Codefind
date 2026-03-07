from __future__ import annotations

from dataclasses import dataclass
from functools import lru_cache
from typing import Any

import jwt
from fastapi import Header, HTTPException, Request, status
from jwt import PyJWKClient

from ..config import Settings, get_settings
from .request_context import set_request_identity


@dataclass(slots=True, frozen=True)
class OrgContext:
    org_id: str
    org_role: str
    user_id: str


class TokenVerificationError(ValueError):
    """Raised when a Clerk token cannot be verified."""


def extract_bearer(authorization: str | None) -> str:
    if not authorization:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Missing Authorization header.",
        )
    scheme, _, token = authorization.partition(" ")
    if scheme.lower() != "bearer" or not token:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid Authorization header.",
        )
    return token


@lru_cache(maxsize=1)
def get_jwk_client(jwks_url: str) -> PyJWKClient:
    return PyJWKClient(jwks_url)


def verify_clerk_token(token: str, settings: Settings) -> dict[str, Any]:
    signing_key = get_jwk_client(settings.clerk_jwks_url).get_signing_key_from_jwt(token)
    claims = jwt.decode(
        token,
        signing_key.key,
        algorithms=["RS256"],
        issuer=settings.clerk_iss,
        options={"require": ["exp", "iss", "sub"]},
    )
    authorized_party = claims.get("azp")
    if authorized_party != settings.clerk_azp:
        raise TokenVerificationError("Invalid authorized party.")
    return claims


async def require_auth(
    request: Request,
    authorization: str | None = Header(default=None),
) -> OrgContext:
    settings = get_settings()
    token = extract_bearer(authorization)
    try:
        claims = verify_clerk_token(token, settings)
    except HTTPException:
        raise
    except jwt.PyJWTError as error:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid or expired token.",
        ) from error
    except TokenVerificationError as error:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail=str(error),
        ) from error

    org_id = claims.get("org_id")
    org_role = claims.get("org_role")
    user_id = claims.get("sub")
    if not org_id:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="No active organization.",
        )
    if org_role not in {"org:admin", "org:member"}:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="No valid organization role.",
        )
    if not user_id:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Token missing subject.",
        )
    context = OrgContext(org_id=org_id, org_role=org_role, user_id=user_id)
    request.state.org_context = context
    set_request_identity(org_id=org_id, user_id=user_id, user_role=org_role)
    return context


async def require_admin(
    request: Request,
    authorization: str | None = Header(default=None),
) -> OrgContext:
    context = await require_auth(request=request, authorization=authorization)
    if context.org_role != "org:admin":
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Admin role required.",
        )
    return context
