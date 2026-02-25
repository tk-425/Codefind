import os
import sys
import unittest
from unittest.mock import patch

from fastapi.testclient import TestClient


CURRENT_DIR = os.path.dirname(__file__)
SERVER_DIR = os.path.abspath(os.path.join(CURRENT_DIR, ".."))
if SERVER_DIR not in sys.path:
    sys.path.insert(0, SERVER_DIR)

os.environ.setdefault("OLLAMA_URL", "http://localhost:11434")
os.environ.setdefault("CHROMADB_URL", "http://localhost:8000")

with patch("transformers.AutoTokenizer.from_pretrained", return_value=object()):
    import app as app_module


class FakeChromaOK:
    def list_collections(self):
        return ["repo-a", "repo-b"]


class FakeChromaError:
    def list_collections(self):
        raise Exception("chromadb unavailable")


class TestCollectionsEndpoint(unittest.TestCase):
    def setUp(self):
        self.client = TestClient(app_module.app)

    def test_list_collections_success(self):
        with patch.object(app_module, "ChromaDBService", return_value=FakeChromaOK()):
            response = self.client.get("/collections")

        self.assertEqual(response.status_code, 200)
        data = response.json()
        self.assertEqual(data["collections"], ["repo-a", "repo-b"])
        self.assertEqual(data["error"], "")

    def test_list_collections_failure(self):
        with patch.object(app_module, "ChromaDBService", return_value=FakeChromaError()):
            response = self.client.get("/collections")

        self.assertEqual(response.status_code, 200)
        data = response.json()
        self.assertEqual(data["collections"], [])
        self.assertIn("chromadb unavailable", data["error"])


if __name__ == "__main__":
    unittest.main()

