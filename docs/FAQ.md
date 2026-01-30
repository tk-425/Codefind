# Frequently Asked Questions (FAQ)

## General Questions

### What is codefind?

Codefind is a semantic code search tool that indexes your codebase and allows natural language queries. Unlike grep/ripgrep that match text literally, codefind finds code based on meaning.

### How is it different from grep?

| Feature       | grep/rg           | codefind                  |
| ------------- | ----------------- | ------------------------- |
| Search type   | Text matching     | Semantic similarity       |
| Query         | Exact/regex       | Natural language          |
| Understanding | None              | Code context              |
| Example       | `grep "validate"` | `"user input validation"` |

### What languages are supported?

**LSP chunking (best quality):**

- Go (gopls)
- Python (Pyright)
- TypeScript/JavaScript (TypeScript Language Server)
- Java (Eclipse JDT LS)
- Swift (SourceKit-LSP)
- Rust (rust-analyzer)
- OCaml (OCaml LSP)

**Window chunking (fallback):**

- All text-based languages

---

## Setup Questions

### Do I need to run my own server?

Yes. Codefind requires a server running:

- Ollama (embeddings)
- ChromaDB (vector storage)
- FastAPI (API layer)

### Can I use it on multiple machines?

Yes! The server can be shared. Use Tailscale for secure remote access.

### How much disk space does indexing use?

Roughly 5-10KB per code file indexed. A 100-file repo uses ~500KB-1MB.

---

## Indexing Questions

### How long does indexing take?

| Repo Size              | Time          |
| ---------------------- | ------------- |
| Small (10-50 files)    | 30-60 seconds |
| Medium (100-500 files) | 3-8 minutes   |
| Large (1000+ files)    | 15-30 minutes |

### Does indexing read my entire file?

No. Files are chunked (split) into smaller pieces. Each chunk is embedded separately.

### What files are skipped?

- Files in .gitignore
- Binary files
- Build artifacts (node_modules, vendor, etc.)

### Why use LSP chunking?

LSP understands code structure. It creates chunks at logical boundaries (functions, classes) rather than arbitrary line counts. This improves search quality.

---

## Query Questions

### Why are my results not relevant?

Try:

1. More specific queries
2. Include context: "error handling in authentication"
3. Re-index if code changed
4. Check project is indexed: `codefind list`

### Can I search across multiple projects?

Yes:

```bash
codefind query "pattern" --all          # All projects
codefind query "pattern" --projects="A,B"  # Specific projects
```

### What do similarity scores mean?

| Score     | Meaning         |
| --------- | --------------- |
| 0.90+     | Excellent match |
| 0.75-0.89 | Good match      |
| 0.60-0.74 | Moderate        |
| < 0.60    | Weak match      |

---

## Troubleshooting Questions

### Why is the server unreachable?

Check:

1. Server is running
2. Correct IP address
3. Tailscale is connected (if using)
4. Port 8080 is open

### Why did indexing fail?

Common causes:

- Server timeout (transient, retry)
- LSP not available (falls back to window chunking)
- Authentication expired

### Why is LSP slow?

LSPs can be slow on:

- First run (loading)
- Large files
- Complex codebases

Use `--window-only` for faster (but less accurate) indexing.

---

## Security Questions

### Is my code sent to the cloud?

No. Code is only sent to YOUR server. Nothing goes to external services.

### Is the connection encrypted?

- Tailscale: Yes (WireGuard encryption)
- Local network: HTTP (consider HTTPS for production)

### Who can access my indexed code?

Only users with a valid auth key for your server.

---

## Performance Questions

### How can I speed up indexing?

1. Use `--window-only` to skip LSP
2. Index smaller subdirectories
3. Ensure server has adequate resources

### How much memory does the server need?

- Ollama: 4GB+ RAM (8GB recommended)
- ChromaDB: 1GB+
- API Server: 512MB

### Can I index very large repos?

Yes, but:

- May take 30+ minutes
- Consider indexing subdirectories
- Ensure server has resources

---

## Feature Questions

### Can I delete an indexed project?

Not yet. Planned for future release.

### Can I update only changed files?

Yes! Re-running `codefind index` does incremental updates automatically.

### Can I export search results?

Query results are printed to stdout. Use shell redirection:

```bash
codefind query "pattern" > results.txt
```
