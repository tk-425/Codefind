# Codefind API Server

FastAPI server for Code-Search that handles embeddings, indexing, and semantic search.

**Important**: This server runs on the **server machine** (not the client machine). Deploy to `~/.codefind-server/api-server/`.

## Quick Deploy to New Server

### 1. Prerequisites

On the new server machine:

```bash
# Install uv (Python package manager)
curl -LsSf https://astral.sh/uv/install.sh | sh

# Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# Install Docker (for ChromaDB)
# Ubuntu/Debian: sudo apt install docker.io docker-compose
# Or see: https://docs.docker.com/engine/install/

# Add user to docker group (avoids permission errors)
sudo usermod -aG docker $USER
newgrp docker  # Apply immediately, or logout/login
```

### 2. Create Directory Structure

```bash
mkdir -p ~/.codefind-server/api-server
mkdir -p ~/.codefind-server/chromadb-data
mkdir -p ~/.codefind-server/ollama-models
```

### 3. Configure Ollama Model Directory

Add to `~/.bashrc` or `~/.zshrc`:

```bash
export OLLAMA_MODELS=/home/<user>/.codefind-server/ollama-models
```

Reload shell: `source ~/.bashrc`

### 4. Pull Required Embedding Models

```bash
ollama pull unclemusclez/jina-embeddings-v2-base-code
ollama pull bge-m3
ollama pull mxbai-embed-large
```

### 5. Start ChromaDB (Docker)

Create `~/.codefind-server/docker-compose.yml`:

```yaml
version: "3.8"
services:
  chromadb:
    image: chromadb/chroma:latest
    ports:
      - "8000:8000"
    volumes:
      - ./chromadb-data:/chroma/chroma
    extra_hosts:
      - "host.docker.internal:host-gateway"
    restart: unless-stopped
```

Start it:

```bash
cd ~/.codefind-server
docker-compose up -d
```

### 6. Copy API Server Files

From your dev machine, copy these files to `~/.codefind-server/api-server/`:

- `app.py`
- `auth.py`
- `models.py`
- `pyproject.toml`

### 7. Install Dependencies & Start Server

```bash
cd ~/.codefind-server/api-server
uv sync
uv run app.py
```

Server starts on `http://0.0.0.0:8080`

### 8. Update Client

On your client machine:

```bash
./codefind init --server-url http://<NEW_SERVER_IP>:8080
```

---

## Migrate Existing Server

To move from an old server to a new one:

### On Old Server

```bash
cd ~/.codefind-server
docker-compose down
killall ollama  # or: systemctl stop ollama
```

### Copy Everything

```bash
rsync -avz ~/.codefind-server new-server:~/
```

> **Note**: This copies all data including `chromadb-data` (existing index) and `ollama-models`. If you prefer a fresh start, skip copying `chromadb-data/` and re-index on the new server with `codefind index`.

### On New Server

```bash
# Set OLLAMA_MODELS in shell profile
export OLLAMA_MODELS=/home/<user>/.codefind-server/ollama-models

# Start services
ollama serve &
cd ~/.codefind-server && docker-compose up -d
cd ~/.codefind-server/api-server && uv run app.py
```

### On Client

```bash
./codefind init --server-url http://<NEW_SERVER_IP>:8080
./codefind server status  # Verify connectivity
```

---

## Running the Server

```bash
cd ~/.codefind-server/api-server
uv run app.py
```

Server will start on `http://0.0.0.0:8080` and will be accessible to clients over the Tailscale network.

## API Endpoints

### Public Endpoints

#### Health Check

```
GET /health
```

Returns status of Ollama and ChromaDB.

#### Tokenize

```bash
POST /tokenize
Content-Type: application/json

{
  "model": "unclemusclez/jina-embeddings-v2-base-code",
  "input": ["def hello(): pass"]
}
```

Forward tokenization request to Ollama.

#### Embed

```bash
POST /embed
Content-Type: application/json

{
  "model": "unclemusclez/jina-embeddings-v2-base-code",
  "input": ["def hello(): pass"]
}
```

Generate embeddings using Ollama.

#### Query

```bash
POST /query
Content-Type: application/json

{
  "query": "search term",
  "collection": "my-repo",
  "top_k": 10,
  "page": 1,
  "page_size": 20
}
```

Search indexed chunks (public, no auth required).

### Protected Endpoints (Require Auth)

All protected endpoints require the `X-Auth-Key` header with a valid authentication key.

#### Index

```bash
POST /index
X-Auth-Key: <auth-key>
Content-Type: application/json

{
  "auth_key": "<auth-key>",
  "chunks": [...],
  "collection": "my-repo"
}
```

Index chunks into ChromaDB.

### Admin Endpoints

#### Bootstrap (Create First Manager)

```bash
POST /admin/bootstrap?email=admin@example.com&auth_key=secret-key-123
```

One-time operation to create the first manager. Disabled after first use.

#### Add Manager

```bash
POST /admin/add?email=user@example.com
X-Auth-Key: <valid-auth-key>
```

Add new manager (requires valid auth key).

#### List Managers

```bash
GET /admin/list
X-Auth-Key: <valid-auth-key>
```

List all managers.

#### Remove Manager

```bash
DELETE /admin/<email>
X-Auth-Key: <valid-auth-key>
```

Remove a manager.

## Configuration

### Environment Variables

- `OLLAMA_URL`: Ollama endpoint (default: `http://localhost:11434`)
- `CHROMADB_URL`: ChromaDB endpoint (default: `http://localhost:8000`)

Edit in `app.py` if needed.

### Manager Storage

Managers are stored on the server machine in `~/.codefind-server/managers.json`

Auth keys are hashed using SHA256 (never stored in plaintext).

## Development

### File Structure (on server machine)

```
~/.codefind-server/api-server/
├── app.py          # FastAPI application
├── models.py       # Pydantic models
├── auth.py         # Authentication logic
├── pyproject.toml  # Project configuration
├── .venv/          # Virtual environment (created by uv)
├── uv.lock         # Locked dependencies
└── README.md       # Documentation
```

### TODO Items

- Implement ChromaDB upsert logic in `/index` endpoint
- Implement ChromaDB query logic in `/query` endpoint
- Generate new auth keys in `/admin/add` endpoint
