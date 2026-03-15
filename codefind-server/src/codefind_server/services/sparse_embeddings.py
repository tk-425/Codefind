from __future__ import annotations

import asyncio
from collections.abc import Iterable
from dataclasses import dataclass
from pathlib import Path
from typing import cast

from fastembed import SparseTextEmbedding

from ..adapters.base import SparseVectorData


class SparseEmbeddingError(RuntimeError):
    """Raised when sparse embedding generation fails."""


@dataclass(slots=True, frozen=True)
class SparseEmbeddingResponse:
    indices: list[int]
    values: list[float]

    def to_vector_data(self) -> SparseVectorData:
        return SparseVectorData(indices=list(self.indices), values=list(self.values))


class SparseEmbeddingService:
    def __init__(
        self,
        *,
        model_name: str,
        cache_dir: str | None = None,
        batch_size: int = 16,
    ) -> None:
        self.model_name = model_name
        self.cache_dir = str(Path(cache_dir).expanduser().resolve()) if cache_dir else None
        self.batch_size = batch_size
        self._model: SparseTextEmbedding | None = None

    def _get_model(self) -> SparseTextEmbedding:
        if self._model is None:
            try:
                self._model = SparseTextEmbedding(
                    model_name=self.model_name,
                    cache_dir=self.cache_dir,
                )
            except Exception as error:
                raise SparseEmbeddingError(
                    f"failed to initialize sparse embedding model '{self.model_name}': {error}"
                ) from error
        return self._model

    async def warmup(self, text: str = "codefind sparse retrieval warmup") -> None:
        await self.query_embed(text)

    async def embed_many(self, texts: list[str]) -> list[SparseEmbeddingResponse]:
        if not texts:
            return []
        return await asyncio.to_thread(self._embed_documents_sync, texts)

    async def query_embed(self, text: str) -> SparseEmbeddingResponse:
        responses = await asyncio.to_thread(self._embed_queries_sync, [text])
        return responses[0]

    def _embed_documents_sync(self, texts: list[str]) -> list[SparseEmbeddingResponse]:
        model = self._get_model()
        try:
            embeddings = list(model.embed(texts, batch_size=self.batch_size))
        except Exception as error:
            raise SparseEmbeddingError(f"sparse document embedding failed: {error}") from error
        return self._normalize_embeddings(embeddings)

    def _embed_queries_sync(self, texts: list[str]) -> list[SparseEmbeddingResponse]:
        model = self._get_model()
        try:
            embeddings = list(model.query_embed(texts))
        except Exception as error:
            raise SparseEmbeddingError(f"sparse query embedding failed: {error}") from error
        return self._normalize_embeddings(embeddings)

    def _normalize_embeddings(self, embeddings: list[object]) -> list[SparseEmbeddingResponse]:
        responses: list[SparseEmbeddingResponse] = []
        for embedding in embeddings:
            indices = getattr(embedding, "indices", None)
            values = getattr(embedding, "values", None)
            if isinstance(indices, str) or not isinstance(indices, Iterable):
                raise SparseEmbeddingError("sparse embedding response was missing indices or values")
            if isinstance(values, str) or not isinstance(values, Iterable):
                raise SparseEmbeddingError("sparse embedding response was missing indices or values")
            try:
                index_values = list(indices)
                weight_values = list(values)
            except TypeError as error:
                raise SparseEmbeddingError("sparse embedding response was missing indices or values") from error
            responses.append(
                SparseEmbeddingResponse(
                    indices=[int(value) for value in cast(list[object], index_values)],
                    values=[float(value) for value in cast(list[object], weight_values)],
                )
            )
        return responses
