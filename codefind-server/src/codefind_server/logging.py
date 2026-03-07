from __future__ import annotations

import logging

from .config import Settings


def configure_logging(*, settings: Settings) -> None:
    logging.basicConfig(level=logging.INFO)
    logger = logging.getLogger("codefind")
    logger.debug("logging configured", extra={"environment": settings.environment})
