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
    with open(MANAGERS_FILE, 'r') as f:
        return json.load(f)


def save_managers(managers: Dict):
    """Save managers to JSON file."""
    os.makedirs(os.path.dirname(MANAGERS_FILE), exist_ok=True)
    with open(MANAGERS_FILE, 'w') as f:
        json.dump(managers, f, indent=2, default=str)


def hash_auth_key(auth_key: str) -> str:
    """Hash auth key using SHA256."""
    return hashlib.sha256(auth_key.encode('utf-8')).hexdigest()


def validate_auth_key(auth_key: str) -> bool:
    """Check if auth key is valid."""
    managers = load_managers()
    key_hash = hash_auth_key(auth_key)

    # Check if key exists in managers
    for email, data in managers.items():
        if data.get('auth_key_hash') == key_hash:
            # Update last_used_at
            data['last_used_at'] = datetime.now().isoformat()
            save_managers(managers)
            return True

    return False


def create_first_manager(email: str, auth_key: str) -> bool:
    """Create first manager (bootstrap only)."""
    managers = load_managers()

    # Fail if managers already exist
    if managers:
        return False

    # Create first manager
    managers[email] = {
        'auth_key_hash': hash_auth_key(auth_key),
        'created_at': datetime.now().isoformat(),
        'last_used_at': None
    }
    save_managers(managers)
    return True


def add_manager(email: str, auth_key: str) -> bool:
    """Add new manager (requires existing auth)."""
    managers = load_managers()

    # Fail if manager already exists
    if email in managers:
        return False

    # Add manager
    managers[email] = {
        'auth_key_hash': hash_auth_key(auth_key),
        'created_at': datetime.now().isoformat(),
        'last_used_at': None
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
