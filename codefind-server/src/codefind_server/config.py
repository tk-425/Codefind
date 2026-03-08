import os
from dataclasses import dataclass
from functools import lru_cache
from urllib.parse import urlparse
from pathlib import Path


class SettingsError(ValueError):
    """Raised when server configuration is invalid."""


REPO_ROOT = Path(__file__).resolve().parents[3]
SERVER_ROOT = Path(__file__).resolve().parents[2]


@dataclass(frozen=True)
class Settings:
    environment: str
    web_app_url: str
    vector_store: str
    qdrant_url: str
    ollama_url: str
    clerk_iss: str
    clerk_azp: str
    clerk_jwks_url: str
    clerk_secret_key: str
    ollama_embed_model: str = "nomic-embed-text"
    tokenizer_model: str = "bert-base-uncased"
    audit_log_path: str | None = None
    sentry_dsn: str | None = None
    sentry_traces_sample_rate: float = 0.0
    max_request_body_bytes: int = 1_048_576
    rate_limit_window_seconds: int = 60
    rate_limit_auth_per_window: int = 10
    rate_limit_admin_per_window: int = 30
    rate_limit_query_per_window: int = 120
    rate_limit_default_per_window: int = 300

    @property
    def audit_log_dir(self) -> Path | None:
        if not self.audit_log_path:
            return None
        return Path(self.audit_log_path).expanduser().resolve().parent

    @property
    def audit_log_file(self) -> Path | None:
        if not self.audit_log_path:
            return None
        return Path(self.audit_log_path).expanduser().resolve()

    @classmethod
    def from_env(cls) -> "Settings":
        return cls(
            environment=os.getenv("ENVIRONMENT", "development"),
            web_app_url=os.getenv("WEB_APP_URL", "http://localhost:5173"),
            vector_store=os.getenv("VECTOR_STORE", ""),
            qdrant_url=os.getenv("QDRANT_URL", ""),
            ollama_url=os.getenv("OLLAMA_URL", ""),
            ollama_embed_model=os.getenv("OLLAMA_EMBED_MODEL", "nomic-embed-text"),
            tokenizer_model=os.getenv("TOKENIZER_MODEL", "bert-base-uncased"),
            clerk_iss=os.getenv("CLERK_ISS", ""),
            clerk_azp=os.getenv("CLERK_AZP", ""),
            clerk_jwks_url=os.getenv("CLERK_JWKS_URL", ""),
            clerk_secret_key=os.getenv("CLERK_SECRET_KEY", ""),
            audit_log_path=os.getenv("AUDIT_LOG_PATH") or None,
            sentry_dsn=os.getenv("SENTRY_DSN") or None,
            sentry_traces_sample_rate=float(
                os.getenv("SENTRY_TRACES_SAMPLE_RATE", "0.0")
            ),
            max_request_body_bytes=int(
                os.getenv("MAX_REQUEST_BODY_BYTES", "1048576")
            ),
            rate_limit_window_seconds=int(
                os.getenv("RATE_LIMIT_WINDOW_SECONDS", "60")
            ),
            rate_limit_auth_per_window=int(
                os.getenv("RATE_LIMIT_AUTH_PER_WINDOW", "10")
            ),
            rate_limit_admin_per_window=int(
                os.getenv("RATE_LIMIT_ADMIN_PER_WINDOW", "30")
            ),
            rate_limit_query_per_window=int(
                os.getenv("RATE_LIMIT_QUERY_PER_WINDOW", "120")
            ),
            rate_limit_default_per_window=int(
                os.getenv("RATE_LIMIT_DEFAULT_PER_WINDOW", "300")
            ),
        )

    def validate_required(self) -> "Settings":
        required = {
            "VECTOR_STORE": self.vector_store,
            "QDRANT_URL": self.qdrant_url,
            "OLLAMA_URL": self.ollama_url,
            "CLERK_ISS": self.clerk_iss,
            "CLERK_AZP": self.clerk_azp,
            "CLERK_JWKS_URL": self.clerk_jwks_url,
            "CLERK_SECRET_KEY": self.clerk_secret_key,
        }
        missing = [name for name, value in required.items() if not value]
        if missing:
            joined = ", ".join(sorted(missing))
            raise SettingsError(f"Missing required environment variables: {joined}")
        if self.vector_store != "qdrant":
            raise SettingsError(
                f"Unsupported VECTOR_STORE '{self.vector_store}'. Expected 'qdrant'."
            )
        self._validate_web_app_url(self.web_app_url)
        if self.audit_log_file is not None:
            self._validate_audit_log_path(self.audit_log_file)
        if not 0.0 <= self.sentry_traces_sample_rate <= 1.0:
            raise SettingsError("SENTRY_TRACES_SAMPLE_RATE must be between 0.0 and 1.0.")
        for name, value in (
            ("MAX_REQUEST_BODY_BYTES", self.max_request_body_bytes),
            ("RATE_LIMIT_WINDOW_SECONDS", self.rate_limit_window_seconds),
            ("RATE_LIMIT_AUTH_PER_WINDOW", self.rate_limit_auth_per_window),
            ("RATE_LIMIT_ADMIN_PER_WINDOW", self.rate_limit_admin_per_window),
            ("RATE_LIMIT_QUERY_PER_WINDOW", self.rate_limit_query_per_window),
            ("RATE_LIMIT_DEFAULT_PER_WINDOW", self.rate_limit_default_per_window),
        ):
            if value <= 0:
                raise SettingsError(f"{name} must be a positive integer.")
        return self

    @staticmethod
    def _validate_audit_log_path(path: Path) -> None:
        if path.is_dir():
            raise SettingsError("AUDIT_LOG_PATH must point to a file, not a directory.")
        if path.is_relative_to(REPO_ROOT) or path.is_relative_to(SERVER_ROOT):
            raise SettingsError(
                "AUDIT_LOG_PATH must live outside the repository working tree."
            )

    @staticmethod
    def _validate_web_app_url(value: str) -> None:
        parsed = urlparse(value)
        if parsed.scheme not in {"http", "https"} or not parsed.netloc:
            raise SettingsError("WEB_APP_URL must be a valid http(s) URL.")


@lru_cache(maxsize=1)
def get_settings() -> Settings:
    return Settings.from_env().validate_required()
