#!/usr/bin/env python3
"""Integration test for /index endpoint."""

import requests
import json
import os
from datetime import datetime
from dotenv import load_dotenv

# Load environment variables from .env file
load_dotenv()

SERVER_URL = os.getenv("SERVER_URL")
AUTH_KEY = os.getenv("AUTH_KEY")

# Fail explicitly if environment variables not set
if not SERVER_URL:
    raise ValueError("SERVER_URL environment variable not set. Check .env file.")
if not AUTH_KEY:
    raise ValueError("AUTH_KEY environment variable not set. Check .env file.")


def test_single_chunk():
    """Test: Send single chunk and verify response."""
    print("\n[TEST 1] Sending single chunk...")

    payload = {
        "auth_key": AUTH_KEY,
        "collection": "test-repo-single",
        "chunks": [
            {
                "content": "def factorial(n):\n    return 1 if n <= 1 else n * factorial(n-1)",
                "metadata": {
                    "repo_id": "test-repo-single",
                    "project_name": "Test Project",
                    "file_path": "src/math.py",
                    "language": "python",
                    "start_line": 1,
                    "end_line": 2,
                    "content_hash": "hash001",
                    "model_id": "bge-m3",
                    "indexed_at": datetime.now().isoformat(),
                    "chunk_tokens": 20
                }
            }
        ]
    }

    response = requests.post(
        f"{SERVER_URL}/index",
        json=payload,
        headers={"X-Auth-Key": AUTH_KEY}
    )

    print(f"  Status: {response.status_code}")
    print(f"  Response: {json.dumps(response.json(), indent=2)}")

    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    data = response.json()
    assert data["inserted_count"] == 1, f"Expected 1 chunk, got {data['inserted_count']}"
    print("  ✅ PASSED\n")


def test_multiple_chunks():
    """Test: Send batch of 5 chunks and verify all stored."""
    print("[TEST 2] Sending batch of 5 chunks...")

    chunks_data = [
        ("class Calculator:\n    def add(x, y):\n        return x + y", 1, 3),
        ("    def subtract(x, y):\n        return x - y", 4, 5),
        ("    def multiply(x, y):\n        return x * y", 6, 8),
        ("    def divide(x, y):\n        return x / y", 9, 11),
        ("calc = Calculator()\nresult = calc.add(5, 3)", 12, 13),
    ]

    chunks = []
    for i, (content, start, end) in enumerate(chunks_data):
        chunks.append({
            "content": content,
            "metadata": {
                "repo_id": "test-repo-batch",
                "project_name": "Test Project",
                "file_path": "src/calculator.py",
                "language": "python",
                "start_line": start,
                "end_line": end,
                "content_hash": f"hash{i:03d}",
                "model_id": "bge-m3",
                "indexed_at": datetime.now().isoformat(),
                "chunk_tokens": 15 + i
            }
        })

    payload = {
        "auth_key": AUTH_KEY,
        "collection": "test-repo-batch",
        "chunks": chunks
    }

    response = requests.post(
        f"{SERVER_URL}/index",
        json=payload,
        headers={"X-Auth-Key": AUTH_KEY}
    )

    print(f"  Status: {response.status_code}")
    print(f"  Response: {json.dumps(response.json(), indent=2)}")

    assert response.status_code == 200
    data = response.json()
    assert data["inserted_count"] == 5, f"Expected 5 chunks, got {data['inserted_count']}"
    print("  ✅ PASSED\n")


def test_authentication_failure():
    """Test: Request without valid auth key should fail."""
    print("[TEST 3] Testing auth failure (invalid key)...")

    payload = {
        "auth_key": "invalid-key",
        "collection": "test-repo",
        "chunks": [
            {
                "content": "test",
                "metadata": {
                    "repo_id": "test-repo",
                    "project_name": "Test",
                    "file_path": "test.txt",
                    "language": "text",
                    "start_line": 1,
                    "end_line": 1,
                    "content_hash": "xyz",
                    "model_id": "bge-m3",
                    "indexed_at": datetime.now().isoformat(),
                    "chunk_tokens": 1
                }
            }
        ]
    }

    response = requests.post(
        f"{SERVER_URL}/index",
        json=payload,
        headers={"X-Auth-Key": "invalid-key"}
    )

    print(f"  Status: {response.status_code}")
    print(f"  Response: {json.dumps(response.json(), indent=2)}")

    assert response.status_code == 401, f"Expected 401, got {response.status_code}"
    print("  ✅ PASSED (correctly rejected)\n")


def test_collection_creation():
    """Test: Verify collection is auto-created if not exists."""
    print("[TEST 4] Testing auto-collection creation...")

    new_collection = f"test-collection-{datetime.now().timestamp()}"

    payload = {
        "auth_key": AUTH_KEY,
        "collection": new_collection,
        "chunks": [
            {
                "content": "Auto-created collection test",
                "metadata": {
                    "repo_id": new_collection,
                    "project_name": "Test",
                    "file_path": "test.txt",
                    "language": "text",
                    "start_line": 1,
                    "end_line": 1,
                    "content_hash": "abc",
                    "model_id": "bge-m3",
                    "indexed_at": datetime.now().isoformat(),
                    "chunk_tokens": 5
                }
            }
        ]
    }

    response = requests.post(
        f"{SERVER_URL}/index",
        json=payload,
        headers={"X-Auth-Key": AUTH_KEY}
    )

    print(f"  Status: {response.status_code}")
    print(f"  Collection: {new_collection}")
    print(f"  Response: {json.dumps(response.json(), indent=2)}")

    assert response.status_code == 200
    print("  ✅ PASSED (collection auto-created)\n")


def run_all_tests():
    """Run all tests."""
    print("=" * 70)
    print("Integration Tests for /index Endpoint")
    print("=" * 70)

    try:
        test_single_chunk()
        test_multiple_chunks()
        test_authentication_failure()
        test_collection_creation()

        print("=" * 70)
        print("✅ ALL TESTS PASSED")
        print("=" * 70)
        return True

    except AssertionError as e:
        print(f"\n❌ TEST FAILED: {e}")
        return False
    except Exception as e:
        print(f"\n❌ ERROR: {e}")
        return False


if __name__ == "__main__":
    import sys
    success = run_all_tests()
    sys.exit(0 if success else 1)
