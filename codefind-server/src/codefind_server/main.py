import json

from .config import get_settings


def main() -> None:
    settings = get_settings()
    print(json.dumps({"app": "codefind-server", "environment": settings.environment}))
