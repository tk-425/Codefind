from __future__ import annotations

import asyncio
from pathlib import Path

from .config import get_settings
from .services import SparseEmbeddingService


def _cache_has_files(cache_dir: str | None) -> bool:
    if not cache_dir:
        return False
    cache_path = Path(cache_dir).expanduser().resolve()
    if not cache_path.exists():
        return False
    return any(path.is_file() for path in cache_path.rglob("*"))


async def _run() -> None:
    settings = get_settings()
    if not settings.sparse_retrieval_enabled:
        print("[SPARSE] Sparse retrieval disabled; skipping model warmup.")
        return

    cache_dir = settings.sparse_embed_cache_dir
    if cache_dir:
        cache_path = Path(cache_dir).expanduser().resolve()
        cache_path.mkdir(parents=True, exist_ok=True)
        if _cache_has_files(cache_dir):
            print(f"[SPARSE] Cache detected at {cache_path}. Verifying sparse model...")
        else:
            print(f"[SPARSE] Sparse model cache not found at {cache_path}.")
            print("[SPARSE] Downloading sparse model now. Please wait until download completes before running index.")
    else:
        print("[SPARSE] No dedicated sparse cache dir configured. Verifying model with default cache path...")

    service = SparseEmbeddingService(
        model_name=settings.sparse_embed_model,
        cache_dir=cache_dir,
        batch_size=settings.sparse_embed_batch_size,
    )
    await service.warmup()
    if cache_dir:
        print(f"[SPARSE] Sparse model is ready in {Path(cache_dir).expanduser().resolve()}.")
    else:
        print("[SPARSE] Sparse model is ready.")


def main() -> None:
    asyncio.run(_run())


if __name__ == "__main__":
    main()
