## Code-Find Server

Phase 1 scaffold for the Code-Find v2 multi-tenant backend.

Key boundaries established in this scaffold:

- `adapters/base.py` exists before vector-store logic grows
- `middleware/auth.py` exists before protected routes are implemented
- `config.py` and `logging.py` exist before runtime behavior spreads
- runtime logs are configured outside the repository
