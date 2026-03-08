from __future__ import annotations

from fastapi import APIRouter, Depends, Request

from ..middleware.auth import require_auth
from ..models.requests import TokenizeRequest
from ..models.responses import TokenizeResponse
from ..services import TokenizerService


router = APIRouter(prefix="/tokenize", tags=["tokenize"])


def get_tokenizer_service(request: Request) -> TokenizerService:
    return request.app.state.tokenizer


@router.post("", response_model=TokenizeResponse)
async def tokenize(
    payload: TokenizeRequest,
    _context=Depends(require_auth),
    tokenizer: TokenizerService = Depends(get_tokenizer_service),
) -> TokenizeResponse:
    tokens = tokenizer.tokenize(payload.text)
    return TokenizeResponse(
        model=tokenizer.model_name,
        tokens=tokens,
        token_count=len(tokens),
    )
