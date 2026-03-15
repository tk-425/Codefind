## Code-Find Server

Phase 1 scaffold for the Code-Find v2 multi-tenant backend.

Key boundaries established in this scaffold:

- `adapters/base.py` exists before vector-store logic grows
- `middleware/auth.py` exists before protected routes are implemented
- `config.py` and `logging.py` exist before runtime behavior spreads
- runtime logs are configured outside the repository

## Local Development

Start the dev server with the wrapper script:

```bash
bash scripts/dev-server.sh
```

The wrapper will:

- load `codefind-server/.env` automatically when it exists
- start `uvicorn` on `0.0.0.0:8080` by default
- preserve normal signal handling by `exec`-ing the server process

## Environment

The server now uses both dense and sparse embeddings for hybrid retrieval. In addition to the existing Ollama settings, local and server environments should define:

- `SPARSE_RETRIEVAL_ENABLED` — enable or disable sparse retrieval wiring
- `SPARSE_EMBED_MODEL` — FastEmbed sparse model name
- `SPARSE_EMBED_CACHE_DIR` — dedicated cache directory for sparse model files
- `SPARSE_EMBED_BATCH_SIZE` — sparse document embedding batch size during indexing

Use [`codefind-server/.env.example`](/Users/terrykang/Documents/Programming/+Projects/Code-Find-v2/codefind-server/.env.example) as the reference template.

When sparse retrieval is enabled, `scripts/dev-server.sh` now warms the sparse model before starting the server. If the cache directory is empty, the script prints a wait message, downloads the model first, and only then starts `uvicorn`.

Optional: add a `zsh` alias on the server machine so the dev server can be started from any directory.

Add this to `~/.zshrc`:

```bash
alias codefind-server='bash /home/tk-macmini/codefind-server/scripts/dev-server.sh'
```

Then reload your shell:

```bash
source ~/.zshrc
```
