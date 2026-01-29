# CLI Commands Reference

Complete reference for all codefind commands.

---

## Core Commands

### init

Initialize codefind configuration.

```bash
codefind init
```

**What it does:**

- Creates `~/.codefind/` directory
- Prompts for server URL
- Stores configuration in `~/.codefind/config.json`

**Example session:**

```
$ codefind init
Enter server URL: http://x.x.x.x:8080
✅ Configuration saved to ~/.codefind/config.json
```

---

### index

Index the current directory for semantic search.

```bash
codefind index [options]
```

**Options:**
| Option | Description |
|--------|-------------|
| `--window-only` | Use window-based chunking only (skip LSP) |
| `--concurrency=N` | Number of concurrent batch requests (default: 2, max: 8) |

**What it does:**

- Discovers all code files in current directory
- Chunks files using LSP (or window fallback)
- Sends chunks to server for embedding
- Creates local manifest for tracking

**Example:**

```bash
$ cd ~/projects/my-app
$ codefind index

📦 Chunking mode: Hybrid (LSP when available)
🔀 Concurrency: 2 parallel requests
✓ Found 42 files
📤 Sending chunks to server (parallel)...
  Progress: 20/20 batches ✓ (100%)
✅ Indexing complete!

# With custom concurrency
$ codefind index --concurrency=4   # Higher for powerful servers
$ codefind index --concurrency=1   # Serial mode for weak servers
```

**Tips:**

- Run from project root directory
- Re-run after code changes (incremental update)
- Use `--window-only` if LSP is causing issues
- Adjust `--concurrency` based on your server capacity

---

### query

Search indexed code semantically.

```bash
codefind query "<search term>" [options]
```

**Options:**
| Option | Description | Default |
|--------|-------------|---------|
| `--limit=N` | Number of results | 10 |
| `--lang=LANG` | Filter by language | All |
| `--path=PREFIX` | Filter by path prefix | All |
| `--page=N` | Page number | 1 |
| `--page-size=N` | Results per page | 20 |
| `--all` | Search all projects | Current only |
| `--projects="A,B"` | Search specific projects | Current only |

**Examples:**

```bash
# Basic search
codefind query "error handling"

# Limit results
codefind query "authentication" --limit=5

# Filter by language
codefind query "api endpoints" --lang=go

# Filter by path
codefind query "validation" --path=internal/

# Search all projects
codefind query "database connection" --all

# Search specific projects
codefind query "config" --projects="backend,api-server"
```

**Example output:**

```
Results for "error handling":

[1] internal/api/handler.go:45-67 (0.89)
    func handleError(w http.ResponseWriter, err error) {
        if err == nil {
            return
        }
        log.Printf("Error: %v", err)
        ...

[2] internal/service/user.go:112-130 (0.82)
    // HandleValidationError processes validation errors
    func HandleValidationError(err ValidationError) Response {
        ...

Use 'codefind open <id>' to open in editor
```

**Tips:**

- Use natural language descriptions
- More specific queries = better results
- Combine filters for precision

---

### list

List all indexed projects.

```bash
codefind list
```

**Example output:**

```
Indexed Projects:
PROJECT                                  REPO ID      CHUNKS   INDEXED AT
--------------------------------------------------------------------------------
my-app                                   a1b2c3d4     156      2026-01-28 15:30
api-server                               e5f6g7h8     89       2026-01-27 10:15
frontend                                 i9j0k1l2     234      2026-01-26 09:00
```

---

### stats

Show statistics for the current project.

```bash
codefind stats
```

**Example output:**

```
Project: my-app
┌─────────────────┬────────┐
│ Active Chunks   │    156 │
│ Deleted Chunks  │     12 │
│ Total Chunks    │    168 │
│ Storage Overhead│   7.1% │
└─────────────────┴────────┘
```

**Tips:**

- High storage overhead? Run `codefind cleanup`
- Deleted chunks = stale data from modified files

---

### health

Check server connectivity and component status.

```bash
codefind health
```

**Example output (healthy):**

```
✅ Server: OK
✅ Ollama: OK
✅ ChromaDB: OK
```

**Example output (unhealthy):**

```
✅ Server: OK
❌ Ollama: ERROR
✅ ChromaDB: OK

Error details: Ollama service not responding
```

**Tips:**

- Run after `codefind init` to verify setup
- Non-zero exit code if any component fails

---

### open

Open a search result in your editor.

```bash
codefind open <id>
```

**What it does:**

- Opens the file at the specific line
- Uses editor from config, `$EDITOR`, or vim

**Examples:**

```bash
# After running a query, open result #1
codefind open 1

# Open by partial ID
codefind open a1b2
```

**Supported editors:**

- VS Code (`code --goto file:line`)
- Sublime Text (`subl file:line`)
- JetBrains IDEs (`idea --line N file`)
- vim/nvim/nano/emacs (`+line file`)

---

### clear

Remove a project from the index.

```bash
codefind clear [path]
```

**What it does:**

- Deletes local manifest from `~/.codefind/manifests/`
- Deletes vector data from server (ChromaDB)

**Examples:**

```bash
# Clear current directory
codefind clear

# Clear specific project
codefind clear /path/to/project
```

**Warning:** This permanently deletes indexed data. Re-run `codefind index` to re-index.

---

### cleanup

Clean up deleted chunks from the server.

```bash
codefind cleanup [options]
```

**Options:**
| Option | Description |
|--------|-------------|
| `--list` | List deleted chunks without removing |
| `--force` | Skip confirmation prompt |

**Examples:**

```bash
# Preview what would be cleaned
codefind cleanup --list

# Clean with confirmation
codefind cleanup

# Clean without confirmation
codefind cleanup --force
```

**When to use:**

- After many file modifications
- When `codefind stats` shows high storage overhead

---

## Authentication Commands

### auth login

Authenticate with the server.

```bash
codefind auth login
```

**What it does:**

- Prompts for auth key
- Stores securely in macOS Keychain

**Example:**

```
$ codefind auth login
Enter auth key: ********
✅ Authenticated successfully
```

---

### auth logout

Remove stored credentials.

```bash
codefind auth logout
```

**Example output:**

```
$ codefind auth logout
✓ Auth key removed from system keychain
```

---

### auth status

Check authentication status.

```bash
codefind auth status
```

**Example output:**

```
✅ Authenticated
   Stored in: macOS Keychain
```

---

## LSP Commands

### lsp status

Show LSP availability for all languages.

```bash
codefind lsp status
```

**Example output:**

```
LSP Server Status:
──────────────────────────────────────────
✅ gopls (go)
   Version: 0.21.0
   Path: /usr/local/bin/gopls

✅ Eclipse JDT LS (java)
   Version: installed
   Path: /usr/local/bin/jdtls

✅ OCaml LSP (ocaml)
   Version: 1.25.0
   Path: ~/.opam/default/bin/ocamllsp

✅ Pyright (python)
   Version: unknown
   Path: ~/.nvm/versions/node/v22.x.x/bin/pyright-langserver

✅ rust-analyzer (rust)
   Version: 1.86.0
   Path: ~/.cargo/bin/rust-analyzer

✅ SourceKit-LSP (swift)
   Version: installed
   Path: /usr/bin/sourcekit-lsp

✅ TypeScript Language Server (typescript/javascript)
   Version: 5.1.3
   Path: ~/.nvm/versions/node/v22.x.x/bin/typescript-language-server

──────────────────────────────────────────
Found: 7/7 LSP servers
```

---

### lsp test

Test a specific LSP with end-to-end workflow.

```bash
codefind lsp test <language>
```

**Supported languages:** go, python, typescript, java, swift

**Example:**

```bash
$ codefind lsp test go

Testing go LSP...
──────────────────────────────────────────
✅ Process started successfully (0.12s)
✅ Initialize completed (0.45s)
✅ Document symbols retrieved (0.08s)
   Found 5 symbols in test file

LSP is working correctly!
```

---

## Utility Commands

### list-files

Preview files that would be indexed (dry run).

```bash
codefind list-files
```

**Example output:**

```
Found 42 files to index:

Go (15 files):
  cmd/main.go
  internal/api/handler.go
  ...

Python (12 files):
  scripts/migrate.py
  ...

TypeScript (15 files):
  src/components/App.tsx
  ...
```

---

### chunk-file

Show chunks for a specific file.

```bash
codefind chunk-file <file>
```

**Example:**

```bash
$ codefind chunk-file internal/api/handler.go

Chunks for internal/api/handler.go:

[1] Lines 1-45 (function: main)
    package main...

[2] Lines 47-89 (function: handleRequest)
    func handleRequest(w http.ResponseWriter...
```

---

### help

Show help and usage information.

```bash
codefind help
codefind --help
codefind -h
```
