from __future__ import annotations

import json
import logging
from pathlib import Path

from codefind_server.config import Settings
from codefind_server.logging import AUDIT_LOGGER_NAME, configure_logging, emit_audit_event


def test_emit_audit_event_writes_redacted_jsonl(tmp_path: Path):
    audit_path = tmp_path / "audit" / "events.jsonl"
    settings = Settings(
        environment="test",
        web_app_url="http://localhost:5173",
        vector_store="qdrant",
        qdrant_url="http://localhost:6333",
        ollama_url="http://localhost:11434",
        clerk_iss="https://clerk.example.com",
        clerk_azp="http://localhost:3000",
        clerk_jwks_url="https://clerk.example.com/jwks",
        clerk_secret_key="secret",
        audit_log_path=str(audit_path),
    )

    configure_logging(settings=settings)
    emit_audit_event(
        event_type="auth.failed",
        result="denied",
        metadata={
            "authorization": "Bearer secret-token",
            "query": "sensitive query",
            "safe": "value",
        },
        org_id="org_123",
        user_id="user_123",
        user_role="org:admin",
    )

    entries = audit_path.read_text().strip().splitlines()
    assert len(entries) == 1
    payload = json.loads(entries[0])
    assert payload["event_type"] == "auth.failed"
    assert payload["result"] == "denied"
    assert payload["org_id"] == "org_123"
    assert payload["metadata"]["authorization"] == "[REDACTED]"
    assert payload["metadata"]["query"] == "[REDACTED]"
    assert payload["metadata"]["safe"] == "value"

    logging.getLogger(AUDIT_LOGGER_NAME).handlers.clear()
