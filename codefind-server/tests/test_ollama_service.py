from __future__ import annotations

import asyncio

import httpx
import pytest

from codefind_server.services.ollama import (
    EmbeddingResponse,
    OllamaError,
    OllamaService,
)


class DummyResponse:
    def __init__(self, status_code: int, payload: dict[str, object]) -> None:
        self.status_code = status_code
        self._payload = payload

    def json(self) -> dict[str, object]:
        return self._payload


class DummyAsyncClient:
    def __init__(self, responses: list[object]) -> None:
        self.responses = list(responses)
        self.calls: list[tuple[str, dict[str, object]]] = []

    async def post(self, path: str, json: dict[str, object]) -> DummyResponse:
        self.calls.append((path, json))
        response = self.responses.pop(0)
        if isinstance(response, Exception):
            raise response
        return response

    async def aclose(self) -> None:
        return None


def test_embed_many_retries_once_after_read_timeout(
    monkeypatch: pytest.MonkeyPatch,
    caplog: pytest.LogCaptureFixture,
):
    service = OllamaService(
        "http://localhost:11434",
        "nomic-embed-text",
        max_attempts=3,
        retry_backoff_seconds=1.0,
    )
    service._client = DummyAsyncClient(
        [
            httpx.ReadTimeout("timed out"),
            DummyResponse(200, {"embeddings": [[0.1, 0.2], [0.3, 0.4]]}),
        ]
    )

    sleep_calls: list[float] = []

    async def fake_sleep(delay: float) -> None:
        sleep_calls.append(delay)

    monkeypatch.setattr(asyncio, "sleep", fake_sleep)
    caplog.set_level("WARNING", logger="codefind")

    responses = asyncio.run(service.embed_many(["first", "second"]))

    assert responses == [
        EmbeddingResponse(embedding=[0.1, 0.2]),
        EmbeddingResponse(embedding=[0.3, 0.4]),
    ]
    assert service._client.calls == [
        ("/api/embed", {"model": "nomic-embed-text", "input": ["first", "second"]}),
        ("/api/embed", {"model": "nomic-embed-text", "input": ["first", "second"]}),
    ]
    assert sleep_calls == [1.0]
    assert any("embed timeout" in record.getMessage() for record in caplog.records)


def test_embed_many_raises_after_retry_exhaustion(monkeypatch: pytest.MonkeyPatch):
    max_attempts = 3
    service = OllamaService(
        "http://localhost:11434",
        "nomic-embed-text",
        max_attempts=max_attempts,
        retry_backoff_seconds=1.0,
    )
    service._client = DummyAsyncClient(
        [httpx.ReadTimeout("timed out") for _ in range(max_attempts)]
    )

    async def fake_sleep(_: float) -> None:
        return None

    monkeypatch.setattr(asyncio, "sleep", fake_sleep)

    with pytest.raises(OllamaError, match="timed out after"):
        asyncio.run(service.embed_many(["only-once"]))

    assert len(service._client.calls) == max_attempts


def test_embed_many_does_not_retry_non_timeout_http_errors(monkeypatch: pytest.MonkeyPatch):
    service = OllamaService(
        "http://localhost:11434",
        "nomic-embed-text",
        max_attempts=3,
        retry_backoff_seconds=1.0,
    )
    service._client = DummyAsyncClient([httpx.ConnectError("boom")])

    sleep_calls: list[float] = []

    async def fake_sleep(delay: float) -> None:
        sleep_calls.append(delay)

    monkeypatch.setattr(asyncio, "sleep", fake_sleep)

    with pytest.raises(OllamaError, match="request failed"):
        asyncio.run(service.embed_many(["only-once"]))

    assert len(service._client.calls) == 1
    assert sleep_calls == []


def test_embed_many_uses_configured_backoff(monkeypatch: pytest.MonkeyPatch):
    service = OllamaService(
        "http://localhost:11434",
        "nomic-embed-text",
        max_attempts=2,
        retry_backoff_seconds=2.5,
    )
    service._client = DummyAsyncClient(
        [
            httpx.ReadTimeout("timed out"),
            DummyResponse(200, {"embeddings": [[0.1, 0.2]]}),
        ]
    )

    sleep_calls: list[float] = []

    async def fake_sleep(delay: float) -> None:
        sleep_calls.append(delay)

    monkeypatch.setattr(asyncio, "sleep", fake_sleep)

    responses = asyncio.run(service.embed_many(["only-once"]))

    assert responses == [EmbeddingResponse(embedding=[0.1, 0.2])]
    assert sleep_calls == [2.5]
