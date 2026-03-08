from __future__ import annotations

import re


REPO_ID_PATTERN = re.compile(r"^[a-zA-Z0-9_-]+$")


def validate_repo_id(repo_id: str) -> str:
    if not REPO_ID_PATTERN.fullmatch(repo_id):
        raise ValueError(
            "repo_id must match [a-zA-Z0-9_-] and must not contain path separators."
        )
    return repo_id


def collection_name_for(org_id: str, repo_id: str) -> str:
    return f"{org_id}_{validate_repo_id(repo_id)}"


def repo_id_from_collection(org_id: str, collection_name: str) -> str | None:
    prefix = f"{org_id}_"
    if not collection_name.startswith(prefix):
        return None
    repo_id = collection_name.removeprefix(prefix)
    if not repo_id:
        return None
    try:
        return validate_repo_id(repo_id)
    except ValueError:
        return None
