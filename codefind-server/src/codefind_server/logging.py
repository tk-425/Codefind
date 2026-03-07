from __future__ import annotations

import logging

from .config import Settings


def configure_logging(*, settings: Settings) -> None:
    logging.basicConfig(level=logging.INFO)
    logger = logging.getLogger("codefind")
    logger.info("logging configured for %s", settings.environment)
