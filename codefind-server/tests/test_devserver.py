from __future__ import annotations

import pytest

from codefind_server import devserver


def test_run_swallows_keyboard_interrupt(monkeypatch):
    calls: list[tuple[str, object]] = []

    class FakeServer:
        def __init__(self, config) -> None:
            calls.append(("server_init", config))

        def run(self) -> None:
            calls.append(("server_run", None))
            raise KeyboardInterrupt()

    monkeypatch.setattr(devserver.uvicorn, "Config", lambda *args, **kwargs: ("config", args, kwargs))
    monkeypatch.setattr(devserver.uvicorn, "Server", FakeServer)

    devserver.run()

    assert calls[0][0] == "server_init"
    assert calls[1] == ("server_run", None)


def test_run_propagates_non_interrupt_failures(monkeypatch):
    class FakeServer:
        def __init__(self, config) -> None:
            self.config = config

        def run(self) -> None:
            raise RuntimeError("boom")

    monkeypatch.setattr(devserver.uvicorn, "Config", lambda *args, **kwargs: ("config", args, kwargs))
    monkeypatch.setattr(devserver.uvicorn, "Server", FakeServer)

    with pytest.raises(RuntimeError, match="boom"):
        devserver.run()
