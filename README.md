# Codefind

Semantic code search tool using vector embeddings and LSP-based chunking.

## What is Codefind?

Codefind is a client-server code search system that enables semantic search across your codebase using natural language queries. Instead of matching keywords, it understands the meaning of your code and finds relevant implementations, patterns, and examples.

**Example:**

```bash
codefind query "error handling with retry logic"
# Finds all error handling implementations, even if they don't contain those exact words
```

## Key Features

- **Semantic Search**: Natural language queries that understand intent, not just keywords
- **LSP-Based Chunking**: Uses Language Server Protocol to chunk code at symbol boundaries (functions, classes, methods)
- **Multi-Project Support**: Search across multiple repositories simultaneously
- **Incremental Indexing**: Fast updates when code changes
- **Language Support**: Go, Python, TypeScript, JavaScript, Java, Swift, Rust, OCaml
- **Claude Code Integration**: 7 agent skills and 3 slash commands for AI-assisted workflows
- **Advanced Filtering**: Filter by project, language, file path, and more
- **Pagination**: Navigate large result sets efficiently

## Architecture

**Client-Server Model:**

```
┌─────────────┐         ┌─────────────┐         ┌─────────────┐
│   codefind  │  HTTP   │   FastAPI   │         │   Ollama    │
│     CLI     │────────▶│   Server    │────────▶│  (Embed)    │
│   (Go)      │         │  (Python)   │         └─────────────┘
└─────────────┘         └─────────────┘                │
                               │                       │
                               │                       ▼
                               │                ┌─────────────┐
                               └───────────────▶│  ChromaDB   │
                                                │  (Vectors)  │
                                                └─────────────┘
```

**Components:**

- **Client (Go)**: CLI tool with LSP integration for local chunking
- **Server (Python)**: FastAPI server managing embeddings and search
- **Ollama**: Generates vector embeddings using `nomic-embed-text` model
- **ChromaDB**: Stores and searches vector embeddings

## Quick Start

### Prerequisites

- **Go 1.21+** for building the client
- **Server access** (see [docs/Platform-Setup.md](docs/Platform-Setup.md) for server installation)

### Installation

```bash
# Clone repository
git clone https://github.com/tk-425/Codefind.git
cd Codefind

# Build client
go build -o codefind ./cmd/codefind

# Optional: Add to PATH
sudo mv codefind /usr/local/bin/
```

### Initial Setup

1. **Initialize configuration**

   ```bash
   codefind init
   # Enter your server URL when prompted (e.g., http://192.168.1.100:8080)
   ```

2. **Authenticate**

   ```bash
   codefind auth login
   # Enter auth key provided by server admin
   ```

3. **Verify connection**
   ```bash
   codefind health
   # Expected: ✅ Server: OK, ✅ Ollama: OK, ✅ ChromaDB: OK
   ```

### Basic Usage

**Index a repository:**

```bash
cd /path/to/your/project
codefind index
```

**Search your code:**

```bash
# Basic semantic search
codefind query "JWT authentication"

# Filter by language
codefind query "database queries" --lang=python

# Search across all indexed projects
codefind query "error handling patterns" --all

# Limit results
codefind query "API endpoints" --limit=5
```

**List indexed projects:**

```bash
codefind list
```

**View project statistics:**

```bash
codefind stats
```

**Open result in editor:**

```bash
# After running a query, open result #1
codefind open 1
```

## Documentation

Detailed documentation is available in the [docs/](docs/) directory:

- [Quick Start Guide](docs/Quick-Start.md) - Get up and running in 5 minutes
- [Commands Reference](docs/Commands.md) - Complete CLI command documentation
- [Architecture Overview](docs/Architecture.md) - System design and data flow
- [Configuration](docs/Configuration.md) - Configuration options and settings
- [Platform Setup](docs/Platform-Setup.md) - Server installation and setup
- [Troubleshooting](docs/Troubleshooting.md) - Common issues and solutions
- [FAQ](docs/FAQ.md) - Frequently asked questions

## Agent Commands & Skills

Codefind includes integration with Claude Code through agent commands and skills.

### Slash Commands

Located in [agent-commands/commands/](agent-commands/commands/):

- **/codefind-index** - Index repositories with guided workflow
- **/codefind-query** - Execute semantic searches with filters
- **/codefind-list** - List and manage indexed projects

### Skills

Located in [agent-commands/skills/](agent-commands/skills/):

**Command Skills:**

- **codefind-index** - Indexing operations and troubleshooting
- **codefind-query-cmd** - Query syntax, filters, and pagination
- **codefind-list-projects** - Project management and verification

**Workflow Skills:**

- **codefind-search** - High-level semantic search workflows
- **codefind-patterns** - Analyze code patterns across projects
- **codefind-migrate** - Code migration and refactoring assistance
- **codefind-summarize** - Generate documentation from code

See [agent-commands/skills/README.md](agent-commands/skills/README.md) for complete documentation.

## Examples

### Find Authentication Code

```bash
codefind query "JWT token validation with expiration check"
```

### Find Error Handling Patterns

```bash
codefind query "error handling with retry logic" --all
```

### Find API Endpoints in Go

```bash
codefind query "HTTP POST endpoint for user creation" --lang=go
```

### Search Specific Directory

```bash
codefind query "database connection" --path=internal/
```

### Multi-Project Pattern Analysis

```bash
# Search across multiple projects
codefind query "authentication middleware" --projects="api-gateway,auth-service"
```

## LSP Integration

Codefind uses Language Server Protocol for intelligent code chunking:

**Check LSP status:**

```bash
codefind lsp status
```

**Test LSP for a language:**

```bash
codefind lsp test go
codefind lsp test python
```

**Supported LSP servers:**

- **Go**: gopls
- **Python**: Pyright
- **TypeScript/JavaScript**: TypeScript Language Server
- **Java**: Eclipse JDT LS
- **Swift**: SourceKit-LSP
- **Rust**: rust-analyzer
- **OCaml**: OCaml LSP

If LSP is unavailable for a language, Codefind automatically falls back to window-based chunking.

## Development Status

**Status**: Beta/Alpha

This project is under active development. Features and APIs may change.

**Current capabilities:**

- ✅ Semantic search with natural language queries
- ✅ LSP-based symbol chunking
- ✅ Multi-project indexing and search
- ✅ Incremental indexing
- ✅ Advanced filtering (language, path, project)
- ✅ Pagination support
- ✅ Claude Code agent integration

**Known limitations:**

- Server setup requires manual configuration
- Some languages fall back to window chunking
- Large repositories may take time to index initially

## Project Structure

```
.
├── cmd/codefind/           # CLI entry point
├── internal/               # Internal Go packages
│   ├── chunker/           # LSP and window-based chunking
│   ├── indexer/           # Indexing orchestration
│   ├── query/             # Query execution
│   ├── client/            # API client
│   ├── config/            # Configuration management
│   └── lsp/               # LSP integration
├── agent-commands/         # Claude Code integration
│   ├── commands/          # Slash commands
│   └── skills/            # Agent skills
├── docs/                   # Documentation
└── codefind-server/        # Python server (separate deployment)
```

## Troubleshooting

**No results found?**

- Verify project is indexed: `codefind list`
- Check server health: `codefind health`
- Try broader query terms

**Index taking too long?**

- Use `--concurrency=1` for serial mode
- Check LSP server status: `codefind lsp status`
- Use `--window-only` to skip LSP

**Server connection issues?**

- Verify server URL in `~/.codefind/config.json`
- Check authentication: `codefind auth status`
- Test connectivity: `codefind health`

See [docs/Troubleshooting.md](docs/Troubleshooting.md) for detailed solutions.

## Storage Locations

**Client:**

- Configuration: `~/.codefind/config.json`
- Project manifests: `~/.codefind/manifests/`

**Server:**

- Configuration: `~/.codefind-server/.env`
- Vector data: ChromaDB (Docker container)

## Advanced Features

### Incremental Indexing

```bash
# After making code changes
codefind index  # Only re-indexes changed files
```

### Cleanup Deleted Chunks

```bash
# View deleted chunks
codefind cleanup --list

# Remove deleted chunks from server
codefind cleanup
```

### Concurrent Indexing

```bash
# Faster indexing with parallel requests (default: 2, max: 8)
codefind index --concurrency=4
```

### Pagination

```bash
# Navigate large result sets
codefind query "functions" --page=2 --page-size=30
```

---

**Built with**: Go, Python, FastAPI, Ollama, ChromaDB, Language Server Protocol
