from __future__ import annotations

from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from typing import Any


@dataclass(slots=True)
class VectorPoint:
    id: str
    vector: list[float]
    payload: dict[str, Any] = field(default_factory=dict)


@dataclass(slots=True)
class SearchResult:
    id: str
    score: float
    payload: dict[str, Any] = field(default_factory=dict)


@dataclass(slots=True)
class StoredPoint:
    id: str
    payload: dict[str, Any] = field(default_factory=dict)


class VectorStore(ABC):
    @abstractmethod
    async def healthcheck(self) -> bool:
        raise NotImplementedError

    @abstractmethod
    async def upsert(self, collection: str, points: list[VectorPoint]) -> None:
        raise NotImplementedError

    @abstractmethod
    async def ensure_collection(self, collection: str, vector_size: int) -> None:
        raise NotImplementedError

    @abstractmethod
    async def query(
        self,
        collection: str,
        vector: list[float],
        filters: dict[str, Any],
        top_k: int,
    ) -> list[SearchResult]:
        raise NotImplementedError

    @abstractmethod
    async def query_lexical(
        self,
        collection: str,
        query_text: str,
        filters: dict[str, Any],
        top_k: int,
    ) -> list[SearchResult]:
        raise NotImplementedError

    @abstractmethod
    async def update_payload(
        self,
        collection: str,
        ids: list[str],
        payload: dict[str, Any],
    ) -> None:
        raise NotImplementedError

    @abstractmethod
    async def delete(self, collection: str, ids: list[str]) -> None:
        raise NotImplementedError

    @abstractmethod
    async def delete_collection(self, collection: str) -> None:
        raise NotImplementedError

    @abstractmethod
    async def list_collections(self) -> list[str]:
        raise NotImplementedError

    @abstractmethod
    async def count(self, collection: str, filters: dict[str, Any]) -> int:
        raise NotImplementedError

    @abstractmethod
    async def scroll(
        self,
        collection: str,
        filters: dict[str, Any],
        limit: int = 1000,
    ) -> list[StoredPoint]:
        raise NotImplementedError

    async def close(self) -> None:
        return None
