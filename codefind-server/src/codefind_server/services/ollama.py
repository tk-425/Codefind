from __future__ import annotations

from dataclasses import dataclass

import httpx


class OllamaError(RuntimeError):
    """Raised when Ollama embedding requests fail."""


@dataclass(slots=True, frozen=True)
class EmbeddingResponse:
    embedding: list[float]


class OllamaService:
    def __init__(self, base_url: str, embed_model: str) -> None:
        self.base_url = base_url.rstrip("/")
        self.embed_model = embed_model
        self._client = httpx.AsyncClient(base_url=self.base_url, timeout=15.0)

    async def embed(self, text: str) -> EmbeddingResponse:
        try:
            response = await self._client.post(
                "/api/embed",
                json={
                    "model": self.embed_model,
                    "input": text,
                },
            )
        except httpx.HTTPError as error:
            raise OllamaError(f"ollama request failed: {error}") from error

        if response.status_code != httpx.codes.OK:
            raise OllamaError(
                f"ollama embed failed with status {response.status_code}"
            )

        payload = response.json()
        embeddings = payload.get("embeddings")
        if not isinstance(embeddings, list) or not embeddings:
            raise OllamaError("ollama embed response missing embeddings")
        first = embeddings[0]
        if not isinstance(first, list) or not all(
            isinstance(value, int | float) for value in first
        ):
            raise OllamaError("ollama embed response contained invalid embedding data")
        return EmbeddingResponse(embedding=[float(value) for value in first])

    async def close(self) -> None:
        await self._client.aclose()
