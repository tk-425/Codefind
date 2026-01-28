"""
Rate limiting module for codefind-server.

Provides IP-based rate limiting for auth failures and other endpoints.
"""

import time
from collections import defaultdict
from dataclasses import dataclass, field
from typing import Optional
from audit import get_audit_logger


@dataclass
class RateLimitEntry:
    """Tracks rate limit data for a key."""

    attempts: list = field(default_factory=list)  # List of timestamps
    alerted: bool = False  # Whether we've alerted for this key


class RateLimiter:
    """In-memory rate limiter with sliding window."""

    # Rate limit configurations
    AUTH_FAIL_LIMIT = 5  # Max failed auth attempts
    AUTH_FAIL_WINDOW = 60  # Per minute (60 seconds)
    AUTH_ALERT_THRESHOLD = 10  # Alert after this many failures

    def __init__(self):
        self._entries: dict[str, RateLimitEntry] = defaultdict(RateLimitEntry)

    def _cleanup_old_attempts(self, entry: RateLimitEntry, window_seconds: int) -> None:
        """Remove attempts older than the window."""
        cutoff = time.time() - window_seconds
        entry.attempts = [ts for ts in entry.attempts if ts > cutoff]

    def check_auth_rate_limit(self, ip: str) -> bool:
        """Check if IP has exceeded auth failure rate limit.

        Args:
            ip: Client IP address

        Returns:
            True if rate limited (should block), False if allowed
        """
        key = f"auth:{ip}"
        entry = self._entries[key]

        # Clean up old attempts
        self._cleanup_old_attempts(entry, self.AUTH_FAIL_WINDOW)

        # Check if over limit
        return len(entry.attempts) >= self.AUTH_FAIL_LIMIT

    def record_auth_failure(self, ip: str, endpoint: str) -> None:
        """Record an authentication failure.

        Args:
            ip: Client IP address
            endpoint: The endpoint that was accessed
        """
        key = f"auth:{ip}"
        entry = self._entries[key]

        # Record the attempt
        entry.attempts.append(time.time())

        # Clean up old attempts for accurate count
        self._cleanup_old_attempts(entry, self.AUTH_FAIL_WINDOW)

        # Log the failure
        audit = get_audit_logger()
        audit.log_auth_fail(endpoint=endpoint, ip=ip)

        # Alert if threshold reached and not already alerted
        total_attempts = len(entry.attempts)
        if total_attempts >= self.AUTH_ALERT_THRESHOLD and not entry.alerted:
            entry.alerted = True
            audit.log(
                "AUTH_ALERT",
                ip=ip,
                attempts=total_attempts,
                message="Possible brute force attack detected",
            )
            print(f"⚠️  SECURITY ALERT: {total_attempts} failed auth attempts from {ip}")

    def reset_auth_failures(self, ip: str) -> None:
        """Reset auth failure count for an IP (e.g., after successful auth)."""
        key = f"auth:{ip}"
        if key in self._entries:
            del self._entries[key]


class BootstrapGuard:
    """Guards the bootstrap endpoint to ensure one-time use only."""

    def __init__(self):
        self._bootstrapped = False

    def is_bootstrapped(self) -> bool:
        """Check if bootstrap has already been called."""
        return self._bootstrapped

    def mark_bootstrapped(self) -> None:
        """Mark bootstrap as complete."""
        self._bootstrapped = True


# Global instances
_rate_limiter: Optional[RateLimiter] = None
_bootstrap_guard: Optional[BootstrapGuard] = None


def get_rate_limiter() -> RateLimiter:
    """Get or create the global rate limiter instance."""
    global _rate_limiter
    if _rate_limiter is None:
        _rate_limiter = RateLimiter()
    return _rate_limiter


def get_bootstrap_guard() -> BootstrapGuard:
    """Get or create the global bootstrap guard instance."""
    global _bootstrap_guard
    if _bootstrap_guard is None:
        _bootstrap_guard = BootstrapGuard()
    return _bootstrap_guard
