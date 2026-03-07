from __future__ import annotations

import json
import logging
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

from .config import Settings
from .middleware.request_context import get_request_context
from .security import redact_mapping


AUDIT_LOGGER_NAME = "codefind.audit"


class JsonLineFormatter(logging.Formatter):
    def format(self, record: logging.LogRecord) -> str:
        return json.dumps(record.msg, sort_keys=True)


def _ensure_directory(path: Path) -> None:
    path.mkdir(parents=True, exist_ok=True)


def _configure_audit_logger(settings: Settings) -> None:
    logger = logging.getLogger(AUDIT_LOGGER_NAME)
    logger.handlers.clear()
    logger.propagate = False
    logger.setLevel(logging.INFO)

    if settings.audit_log_file is None:
        logger.disabled = True
        return

    logger.disabled = False
    audit_dir = settings.audit_log_file.parent
    _ensure_directory(audit_dir)
    handler = logging.FileHandler(settings.audit_log_file)
    handler.setFormatter(JsonLineFormatter())
    logger.addHandler(handler)


def configure_logging(*, settings: Settings) -> None:
    logging.basicConfig(level=logging.INFO)
    logger = logging.getLogger("codefind")
    logger.info("logging configured for %s", settings.environment)
    _configure_audit_logger(settings)


def emit_audit_event(
    *,
    event_type: str,
    result: str,
    metadata: dict[str, Any] | None = None,
    org_id: str | None = None,
    user_id: str | None = None,
    user_role: str | None = None,
    repo_id: str | None = None,
    target: str | None = None,
) -> None:
    context = get_request_context()
    payload: dict[str, Any] = {
        "timestamp": datetime.now(UTC).isoformat(),
        "request_id": context.request_id if context else None,
        "event_type": event_type,
        "result": result,
        "org_id": org_id or (context.org_id if context else None),
        "user_id": user_id or (context.user_id if context else None),
        "user_role": user_role or (context.user_role if context else None),
        "repo_id": repo_id,
        "target": target,
        "client_ip": context.client_ip if context else None,
        "metadata": redact_mapping(metadata or {}),
    }
    logging.getLogger(AUDIT_LOGGER_NAME).info(payload)
