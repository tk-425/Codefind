import json
import hashlib
import os
from typing import Dict
from datetime import datetime

MANAGERS_FILE = os.path.expanduser("~/.codefind-server/managers.json")


def load_managers() -> Dict:
    """Load managers from JSON file."""
    if not os.path.exists(MANAGERS_FILE):
        return {}
    with open(MANAGERS_FILE, "r") as f:
        return json.load(f)


def save_managers(managers: Dict):
    """Save managers to JSON file."""
    os.makedirs(os.path.dirname(MANAGERS_FILE), exist_ok=True)
    with open(MANAGERS_FILE, "w") as f:
        json.dump(managers, f, indent=2, default=str)


def hash_auth_key(auth_key: str) -> str:
    """Hash auth key using SHA256."""
    return hashlib.sha256(auth_key.encode("utf-8")).hexdigest()


def validate_auth_key(auth_key: str, email: str = None) -> bool:
    """Check if auth key is valid.

    If email is provided, validates that the key belongs to that specific manager.
    If email is not provided (backward compatibility), checks if key exists for any manager.
    """
    managers = load_managers()
    key_hash = hash_auth_key(auth_key)

    if email:
        # Validate email + key pair
        if email not in managers:
            return False
        data = managers[email]
        if data.get("auth_key_hash") != key_hash:
            return False
        # Update last_used_at
        data["last_used_at"] = datetime.now().isoformat()
        save_managers(managers)
        return True
    else:
        # Legacy: check if key exists in any manager
        for manager_email, data in managers.items():
            if data.get("auth_key_hash") == key_hash:
                # Update last_used_at
                data["last_used_at"] = datetime.now().isoformat()
                save_managers(managers)
                return True
        return False


def create_first_manager(email: str, auth_key: str) -> bool:
    """Create first manager (bootstrap only). First manager is always an admin."""
    managers = load_managers()

    # Fail if managers already exist
    if managers:
        return False

    # Create first manager as admin
    managers[email] = {
        "auth_key_hash": hash_auth_key(auth_key),
        "role": "admin",
        "created_at": datetime.now().isoformat(),
        "last_used_at": None,
    }
    save_managers(managers)
    return True


def add_manager(email: str, auth_key: str, role: str = "manager") -> bool:
    """Add new manager (requires existing auth).

    Args:
        email: Email address for the new manager
        auth_key: Auth key for the new manager
        role: Role for the new manager ('admin' or 'manager')
    """
    managers = load_managers()

    # Fail if manager already exists
    if email in managers:
        return False

    # Validate role
    if role not in ("admin", "manager"):
        role = "manager"

    # Add manager
    managers[email] = {
        "auth_key_hash": hash_auth_key(auth_key),
        "role": role,
        "created_at": datetime.now().isoformat(),
        "last_used_at": None,
    }
    save_managers(managers)
    return True


def remove_manager(email: str) -> bool:
    """Remove manager."""
    managers = load_managers()

    if email not in managers:
        return False

    del managers[email]
    save_managers(managers)
    return True


def list_managers() -> Dict:
    """List all managers."""
    return load_managers()


def get_manager_role(email: str) -> str | None:
    """Get role for a manager. Returns None if manager doesn't exist."""
    managers = load_managers()
    if email not in managers:
        return None
    return managers[email].get(
        "role", "manager"
    )  # Default to manager for backwards compat


def validate_admin(auth_key: str, email: str = None) -> bool:
    """Check if auth key belongs to an admin.

    If email is provided, validates that the specific email+key belongs to an admin.
    If email is not provided, checks if key exists for any admin.
    """
    managers = load_managers()
    key_hash = hash_auth_key(auth_key)

    if email:
        # Validate email + key pair and check admin role
        if email not in managers:
            return False
        data = managers[email]
        if data.get("auth_key_hash") != key_hash:
            return False
        if data.get("role", "manager") != "admin":
            return False
        # Update last_used_at
        data["last_used_at"] = datetime.now().isoformat()
        save_managers(managers)
        return True
    else:
        # Legacy: check if key exists for any admin
        for manager_email, data in managers.items():
            if data.get("auth_key_hash") == key_hash:
                if data.get("role", "manager") == "admin":
                    data["last_used_at"] = datetime.now().isoformat()
                    save_managers(managers)
                    return True
        return False
