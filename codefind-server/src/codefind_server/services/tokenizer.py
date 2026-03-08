from __future__ import annotations

from threading import Lock

from transformers import AutoTokenizer


class TokenizerService:
    def __init__(self, model_name: str) -> None:
        self.model_name = model_name
        self._tokenizer = None
        self._lock = Lock()

    def tokenize(self, text: str) -> list[str]:
        tokenizer = self._get_tokenizer()
        return tokenizer.tokenize(text)

    def _get_tokenizer(self):
        if self._tokenizer is not None:
            return self._tokenizer
        with self._lock:
            if self._tokenizer is None:
                self._tokenizer = AutoTokenizer.from_pretrained(self.model_name)
        return self._tokenizer
