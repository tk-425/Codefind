# Configuration Options

Complete reference for codefind configuration.

---

## Config File Location

```
~/.codefind/config.json
```

---

## Configuration Options

### server_url

**Description:** URL of the codefind server  
**Type:** String  
**Default:** None (must be set)

```bash
# View
codefind config get server_url

# Set
codefind config set server_url http://x.x.x.x:8080
```

**Example values:**

- `http://localhost:8080` (local server)
- `http://x.x.x.x:8080` (Tailscale IP)

---

### auth_key

**Description:** Authentication key for server access  
**Type:** String (stored securely in keychain)  
**Default:** None

```bash
# Set via login
codefind auth login

# Check status
codefind auth status
```

---

## Manifest Files

Each indexed project has a manifest file:

```
~/.codefind/manifests/<project-id>.json
```

**Contents:**

- Project name and path
- Last indexed timestamp
- Chunk count and file list
- Git commit hash (if applicable)

---

## Environment Variables

### CODEFIND_SERVER_URL

Override server URL:

```bash
export CODEFIND_SERVER_URL=http://localhost:8080
codefind query "test"  # Uses env var
```

### CODEFIND_DEBUG

Enable debug output:

```bash
export CODEFIND_DEBUG=1
codefind index  # Shows debug info
```

---

## Server Configuration

Server-side configuration is in `.env`:

```bash
# ~/.codefind-server/api-server/.env
OLLAMA_URL=http://localhost:11434
CHROMADB_URL=http://localhost:8000
```

---

## Chunking Configuration

Chunking behavior is controlled by flags:

```bash
# Default: Hybrid (LSP + Window fallback)
codefind index

# Force window-only
codefind index --window-only
```

**Window chunking parameters** (hardcoded defaults):

- Chunk size: 512 tokens
- Overlap: 64 tokens
- Chars per token: 4

---

## Query Options

| Option          | Description           | Default |
| --------------- | --------------------- | ------- |
| `--limit=N`     | Number of results     | 10      |
| `--lang=LANG`   | Filter by language    | All     |
| `--path=PREFIX` | Filter by path prefix | All     |
| `--page=N`      | Page number           | 1       |
| `--page-size=N` | Results per page      | 20      |

---

## LSP Configuration

LSPs are auto-discovered from PATH. Supported:

| Language   | LSP                        | Install                                          |
| ---------- | -------------------------- | ------------------------------------------------ |
| Go         | gopls                      | `go install golang.org/x/tools/gopls@latest`     |
| Python     | Pyright                    | `npm install -g pyright`                         |
| TypeScript | TypeScript Language Server | `npm install -g typescript-language-server`      |
| Java       | Eclipse JDT LS             | Manual install                                   |
| Swift      | SourceKit-LSP              | Included with Xcode                              |
| Rust       | rust-analyzer              | `rustup component add rust-analyzer`             |
| OCaml      | OCaml LSP                  | `opam install ocaml-lsp-server`                  |

---

## Resetting Configuration

```bash
# Remove all config
rm -rf ~/.codefind

# Reinitialize
codefind init
```
