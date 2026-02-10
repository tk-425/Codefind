"""
Audit logging module for codefind-server.

Logs operations to ~/.codefind-server/audit.log with log rotation.
"""

import os
import logging
from logging.handlers import TimedRotatingFileHandler
from pathlib import Path
from datetime import datetime, timezone
from typing import Optional


class AuditLogger:
    """Audit logger with file rotation support."""

    # Operation types
    INDEX = "INDEX"
    QUERY = "QUERY"
    CLEANUP = "CLEANUP"
    DELETE = "DELETE"
    CLEAR = "CLEAR"
    AUTH_FAIL = "AUTH_FAIL"
    ADMIN_ADD = "ADMIN_ADD"
    ADMIN_REMOVE = "ADMIN_REMOVE"
    BOOTSTRAP = "BOOTSTRAP"

    def __init__(self, log_dir: Optional[str] = None):
        """Initialize audit logger.

        Args:
            log_dir: Directory for log files. Defaults to ~/.codefind-server
        """
        if log_dir is None:
            log_dir = os.path.join(os.path.expanduser("~"), ".codefind-server")

        self.log_dir = Path(log_dir)
        self.log_dir.mkdir(parents=True, exist_ok=True)

        # Create logs subdirectory
        self.logs_dir = self.log_dir / "logs"
        self.logs_dir.mkdir(parents=True, exist_ok=True)

        self.log_file = self.logs_dir / "audit.log"

        # Create logger
        self.logger = logging.getLogger("codefind.audit")
        self.logger.setLevel(logging.INFO)

        # Avoid duplicate handlers
        if not self.logger.handlers:
            # Create rotating file handler
            # Rotate daily, keep 30 days of logs
            handler = TimedRotatingFileHandler(
                filename=str(self.log_file),
                when="midnight",
                interval=1,
                backupCount=30,
                encoding="utf-8",
            )

            # Format: timestamp [OPERATION] key=value pairs
            formatter = logging.Formatter("%(message)s")
            handler.setFormatter(formatter)

            # Compress rotated logs (adds .gz suffix)
            handler.rotator = self._gzip_rotator
            handler.namer = self._gzip_namer

            self.logger.addHandler(handler)

    def _gzip_namer(self, name: str) -> str:
        """Name rotated files with .gz extension."""
        return name + ".gz"

    def _gzip_rotator(self, source: str, dest: str) -> None:
        """Compress rotated log files with gzip."""
        import gzip
        import shutil

        with open(source, "rb") as f_in:
            with gzip.open(dest, "wb") as f_out:
                shutil.copyfileobj(f_in, f_out)
        os.remove(source)

    def _format_log(self, operation: str, **kwargs) -> str:
        """Format a log entry.

        Args:
            operation: Operation type (INDEX, QUERY, etc.)
            **kwargs: Key-value pairs to log

        Returns:
            Formatted log string
        """
        timestamp = datetime.now(timezone.utc).strftime("[%Y-%m-%d %H:%M:%S]")

        # Build key=value pairs
        pairs = []
        for key, value in kwargs.items():
            if value is not None:
                # Quote strings with spaces
                if isinstance(value, str) and " " in value:
                    pairs.append(f'{key}="{value}"')
                else:
                    pairs.append(f"{key}={value}")

        pairs_str = " ".join(pairs)
        return f"{timestamp} [{operation}] {pairs_str}"

    def log(self, operation: str, **kwargs) -> None:
        """Log an operation.

        Args:
            operation: Operation type
            **kwargs: Key-value pairs to log
        """
        message = self._format_log(operation, **kwargs)
        self.logger.info(message)

    def log_index(
        self,
        repo: str,
        files: int,
        chunks: int,
        method: str = "hybrid",
        user: Optional[str] = None,
    ) -> None:
        """Log an index operation."""
        self.log(
            self.INDEX, repo=repo, files=files, chunks=chunks, method=method, user=user
        )

    def log_query(
        self, collection: str, query: str, results: int, user: Optional[str] = None
    ) -> None:
        """Log a query operation."""
        # Truncate query if too long
        if len(query) > 50:
            query = query[:47] + "..."
        self.log(
            self.QUERY, collection=collection, query=query, results=results, user=user
        )

    def log_cleanup(self, repo: str, purged: int, user: Optional[str] = None) -> None:
        """Log a cleanup/purge operation."""
        self.log(self.CLEANUP, repo=repo, purged=purged, user=user)

    def log_delete(
        self, repo: str, files: int, chunks: int, user: Optional[str] = None
    ) -> None:
        """Log a soft delete operation."""
        self.log(self.DELETE, repo=repo, files=files, chunks=chunks, user=user)

    def log_clear(self, repo: str, user: Optional[str] = None) -> None:
        """Log a collection clear operation."""
        self.log(self.CLEAR, repo=repo, user=user)

    def log_auth_fail(
        self, endpoint: str, ip: Optional[str] = None, reason: str = "invalid_key"
    ) -> None:
        """Log an authentication failure."""
        self.log(self.AUTH_FAIL, endpoint=endpoint, ip=ip, reason=reason)

    def log_admin_add(self, email: str, user: Optional[str] = None) -> None:
        """Log adding a new admin."""
        self.log(self.ADMIN_ADD, email=email, added_by=user)

    def log_admin_remove(self, email: str, user: Optional[str] = None) -> None:
        """Log removing an admin."""
        self.log(self.ADMIN_REMOVE, email=email, removed_by=user)

    def log_bootstrap(self, email: str, ip: Optional[str] = None) -> None:
        """Log bootstrap operation."""
        self.log(self.BOOTSTRAP, email=email, ip=ip)


# Global audit logger instance
_audit_logger: Optional[AuditLogger] = None


def get_audit_logger() -> AuditLogger:
    """Get or create the global audit logger instance."""
    global _audit_logger
    if _audit_logger is None:
        _audit_logger = AuditLogger()
    return _audit_logger
