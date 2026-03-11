from __future__ import annotations

import os
from typing import Final

import uvicorn


DEFAULT_HOST: Final[str] = "0.0.0.0"
DEFAULT_PORT: Final[int] = 8080


def _host() -> str:
    return os.getenv("HOST", DEFAULT_HOST)


def _port() -> int:
    value = os.getenv("PORT")
    if value is None:
        return DEFAULT_PORT
    return int(value)


def run() -> None:
    config = uvicorn.Config("app:app", host=_host(), port=_port(), log_level="info")
    server = uvicorn.Server(config)
    try:
        server.run()
    except KeyboardInterrupt:
        # Uvicorn has already printed the normal shutdown lines; avoid a second traceback dump.
        return


def main() -> None:
    run()


if __name__ == "__main__":
    main()
