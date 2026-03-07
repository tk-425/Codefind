from dataclasses import dataclass
from functools import lru_cache
from pathlib import Path


@dataclass(frozen=True)
class Settings:
    environment: str = "development"
    vector_store: str = "qdrant"
    audit_log_path: str | None = None

    @property
    def audit_log_dir(self) -> Path | None:
        if not self.audit_log_path:
            return None
        return Path(self.audit_log_path).expanduser().resolve().parent


@lru_cache(maxsize=1)
def get_settings() -> Settings:
    return Settings()
