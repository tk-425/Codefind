# Codefind API Server

FastAPI server for Code-Search that handles embeddings, indexing, and semantic search.

**Important**: This server runs on the **server machine** (not the client machine). Deploy to `~/.codefind-server/api-server/`.

## Setup

### Prerequisites
- Python 3.12+
- `uv` package manager
- Ollama running on `localhost:11434` (on same server)
- ChromaDB running on `localhost:8000` (on same server)

### Installation

1. Create the directory structure on server:
```bash
mkdir -p ~/.codefind-server/api-server
cd ~/.codefind-server/api-server
```

2. Copy these files to `~/.codefind-server/api-server/`:
   - `app.py`
   - `auth.py`
   - `models.py`
   - `pyproject.toml`

3. Install dependencies:
```bash
uv sync
```

## Running the Server

```bash
cd ~/.codefind-server/api-server
uv run app.py
```

Server will start on `http://0.0.0.0:8080` and will be accessible to clients over the Tailscale network.

Alternatively, if you've activated the virtual environment:
```bash
cd ~/.codefind-server/api-server
source .venv/bin/activate
python app.py
```

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
