from __future__ import annotations

from dataclasses import dataclass

import httpx


class OllamaError(RuntimeError):
    """Raised when Ollama embedding requests fail."""


OLLAMA_EMBED_TIMEOUT_SECONDS = 180.0


@dataclass(slots=True, frozen=True)
class EmbeddingResponse:
    embedding: list[float]


class OllamaService:
    def __init__(self, base_url: str, embed_model: str) -> None:
        self.base_url = base_url.rstrip("/")
        self.embed_model = embed_model
        self._client = httpx.AsyncClient(
            base_url=self.base_url,
            timeout=OLLAMA_EMBED_TIMEOUT_SECONDS,
        )

    async def embed_many(self, texts: list[str]) -> list[EmbeddingResponse]:
        if not texts:
            return []

        try:
            response = await self._client.post(
                "/api/embed",
                json={
                    "model": self.embed_model,
                    "input": texts,
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
        if len(embeddings) != len(texts):
            raise OllamaError("ollama embed response count did not match request count")

        responses: list[EmbeddingResponse] = []
        for embedding in embeddings:
            if not isinstance(embedding, list) or not all(
                isinstance(value, int | float) for value in embedding
            ):
                raise OllamaError("ollama embed response contained invalid embedding data")
            responses.append(EmbeddingResponse(embedding=[float(value) for value in embedding]))
        return responses

    async def embed(self, text: str) -> EmbeddingResponse:
        responses = await self.embed_many([text])
        return responses[0]

    async def close(self) -> None:
        await self._client.aclose()
