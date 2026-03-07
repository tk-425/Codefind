import os
from dataclasses import dataclass
from functools import lru_cache
from pathlib import Path


class SettingsError(ValueError):
    """Raised when server configuration is invalid."""


@dataclass(frozen=True)
class Settings:
    environment: str
    vector_store: str
    qdrant_url: str
    ollama_url: str
    clerk_iss: str
    clerk_azp: str
    clerk_jwks_url: str
    clerk_secret_key: str
    audit_log_path: str | None = None
    sentry_dsn: str | None = None

    @property
    def audit_log_dir(self) -> Path | None:
        if not self.audit_log_path:
            return None
        return Path(self.audit_log_path).expanduser().resolve().parent

    @classmethod
    def from_env(cls) -> "Settings":
        return cls(
            environment=os.getenv("ENVIRONMENT", "development"),
            vector_store=os.getenv("VECTOR_STORE", ""),
            qdrant_url=os.getenv("QDRANT_URL", ""),
            ollama_url=os.getenv("OLLAMA_URL", ""),
            clerk_iss=os.getenv("CLERK_ISS", ""),
            clerk_azp=os.getenv("CLERK_AZP", ""),
            clerk_jwks_url=os.getenv("CLERK_JWKS_URL", ""),
            clerk_secret_key=os.getenv("CLERK_SECRET_KEY", ""),
            audit_log_path=os.getenv("AUDIT_LOG_PATH") or None,
            sentry_dsn=os.getenv("SENTRY_DSN") or None,
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
        return self


@lru_cache(maxsize=1)
def get_settings() -> Settings:
    return Settings.from_env().validate_required()
