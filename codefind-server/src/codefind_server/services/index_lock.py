from __future__ import annotations

import asyncio


class IndexJobLockManager:
    def __init__(self) -> None:
        self._guard = asyncio.Lock()
        self._active: set[str] = set()

    async def acquire(self, key: str) -> bool:
        async with self._guard:
            if key in self._active:
                return False
            self._active.add(key)
            return True

    async def release(self, key: str) -> None:
        async with self._guard:
            self._active.discard(key)
