---
name: codefind-index
description: Index code repositories for semantic search using LSP-based symbol chunking. Use when you need to index a new repository, update an existing index after code changes, re-index with different settings, or troubleshoot indexing issues.
---

# codefind-index

Index code repositories for semantic search using LSP-based symbol chunking and embeddings.

## Description

This skill guides AI agents through the process of indexing code repositories with codefind. It covers both initial indexing and incremental updates, handles LSP-based chunking, and manages authentication with the codefind server.

## When to Use

Use this skill when you need to:
- Index a new repository for the first time
- Update an existing index after code changes
- Re-index a repository with different settings
- Troubleshoot indexing issues
- Verify indexed content

## Prerequisites

Before indexing:
1. **Authentication configured**: Run `codefind auth status` to verify
   - If not configured: `codefind auth login` and enter your auth key
2. **Server connectivity**: Verify with `codefind server status` (if available)
3. **Repository ready**: Ensure you're in a valid code repository
4. **LSPs available** (optional): Run `codefind lsp status` to check

## Usage

### Basic Indexing Workflow

**Step 1: Verify prerequisites**
```bash
# Check auth status
codefind auth status

# Check available LSPs (optional)
codefind lsp status
```

**Step 2: Run initial index**
```bash
# Index current directory
codefind index

# Index specific directory
codefind index /path/to/repo
```

**Step 3: Verify indexing**
```bash
# Check if project appears in list
codefind list

# Test with a query
codefind query "function or feature name"
```

### Incremental Indexing

After making code changes, re-run index to update:

```bash
# Re-index (automatically detects changes)
codefind index

# Index specific directory after changes
codefind index /path/to/repo
```

**How incremental indexing works:**
- For git repos: Detects changes using `git diff` against last indexed commit
- For non-git repos: Uses file modification times (mtime)
- Only changed/added/deleted files are processed
- Deleted code is marked as "deleted" (tombstone mode) rather than removed

## Command Options

```bash
codefind index [repo-path]
```

**Parameters:**
- `repo-path`: Optional path to repository (defaults to current directory)

**Flags** (if implemented):
- `--force`: Force full re-index (ignore incremental detection)
- `--window-only`: Use window-based chunking (skip LSP)
- `--dry-run`: Show what would be indexed without actually indexing

## Expected Output

### Successful Indexing

```
Indexing repository: Code-Search
Repository path: /Users/you/projects/Code-Search
Server URL: http://100.x.y.z:8000

Discovering files...
Found 14 files to index:
  - Go: 8 files
  - Python: 4 files
  - Markdown: 2 files

Checking LSP availability...
  ✓ Go (gopls) available
  ✓ Python (pyright) available

Chunking files...
  [1/14] cmd/codefind/main.go (LSP) - 12 chunks
  [2/14] internal/config/config.go (LSP) - 8 chunks
  [3/14] internal/indexer/indexer.go (LSP) - 15 chunks
  ...

Tokenizing chunks...
  Batch 1/17: 8 chunks ✓ (5%)
  Batch 2/17: 8 chunks ✓ (12%)
  ...
  Batch 17/17: 2 chunks ✓ (100%)

Sending to server...
  Batch 1/17: 8 chunks indexed ✓
  Batch 2/17: 8 chunks indexed ✓
  ...
  Batch 17/17: 2 chunks indexed ✓

✓ Successfully indexed 130 chunks from 14 files
✓ Manifest updated: ~/.codefind/manifests/code-search-abc123.json
✓ Last indexed commit: abc123def456
```

### Incremental Indexing Output

```
Indexing repository: Code-Search (incremental)
Repository path: /Users/you/projects/Code-Search

Detecting changes since last index...
  Last indexed: 2026-01-27 14:30:00
  Last commit: abc123def456
  Current commit: def789ghi012

Changes detected:
  Modified: 2 files
  Added: 1 file
  Deleted: 0 files

Processing changes...
  [1/3] internal/config/config.go (modified) - 6 chunks
  [2/3] internal/keychain/keychain.go (added) - 10 chunks
  [3/3] cmd/codefind/main.go (modified) - 12 chunks

Marking old chunks as deleted: 18 chunks
Indexing new chunks: 28 chunks

✓ Successfully indexed 28 new chunks
✓ Marked 18 old chunks as deleted
✓ Manifest updated
✓ Last indexed commit: def789ghi012
```

## Indexing Strategies

### LSP-Based Chunking (Primary)

**When LSP is available:**
- Code is chunked by symbol boundaries (functions, classes, methods)
- Each symbol becomes a chunk (unless it exceeds size limit)
- Symbol metadata is preserved (name, kind, line range)

**Benefits:**
- Precise chunks aligned with code structure
- Rich metadata for better search results
- No arbitrary splits in the middle of functions

**Supported languages:**
- TypeScript/JavaScript (typescript-language-server)
- Python (pyright)
- Go (gopls)
- Java (jdtls)
- Swift (sourcekit-lsp)

### Window-Based Chunking (Fallback)

**When LSP is unavailable:**
- Code is split into fixed-size windows (~450 tokens)
- 50-token overlap between windows for context
- No symbol metadata available

**Used for:**
- Languages without LSP support
- LSP failures or timeouts
- Markdown, text, and config files

## Authentication

Indexing requires authentication with the codefind server.

### First-time Setup

```bash
# Store auth key in system keychain
codefind auth login
# Enter your auth key when prompted
```

### Check Auth Status

```bash
codefind auth status
```

**Expected output:**
```
✓ Auth key is configured
✓ Stored in system keychain (secure)
```

### Troubleshooting Auth

**If auth key is missing:**
```bash
codefind auth login
```

**If you need a new auth key:**
- Contact the server administrator
- They can generate a new auth key using server admin commands

## Troubleshooting

### Issue: No files found to index

**Possible causes:**
- Not in a code repository
- All files filtered by .gitignore
- No supported file types

**Solution:**
```bash
# Check what files would be discovered
codefind list-files

# Verify you're in the right directory
pwd

# Check .gitignore rules
cat .gitignore
```

### Issue: LSP not available

**Symptoms:**
```
Checking LSP availability...
  ✗ Python (pyright) not found
```

**Solution:**
```bash
# Install missing LSP
npm install -g pyright  # for Python

# Or use window-based chunking
codefind index --window-only
```

### Issue: Authentication failed

**Symptoms:**
```
Error: Authentication failed (401 Unauthorized)
```

**Solution:**
```bash
# Check auth status
codefind auth status

# Login with valid auth key
codefind auth login
```

### Issue: Server unreachable

**Symptoms:**
```
Error: Failed to connect to server
```

**Solution:**
```bash
# Check server URL configuration
codefind config get server_url

# Update server URL if needed
codefind config set server_url http://new-server-ip:8000

# Verify server is running
curl http://server-ip:8000/api/v1/health
```

### Issue: Indexing stalled or slow

**Possible causes:**
- Large files (>1000 lines)
- LSP timeout
- Network issues

**Solution:**
- Wait for timeout and automatic fallback
- Use `--window-only` to skip LSP
- Check network connectivity
- Index in smaller batches (by directory)

### Issue: Chunks exceed token limit

**Symptoms:**
```
Warning: Chunk exceeds 450 tokens, splitting...
```

**This is normal:**
- Large symbols (functions >450 tokens) are automatically split
- Split chunks have 50-token overlap
- Each split chunk is indexed separately

## Best Practices

### 1. Index incrementally
```bash
# After making changes, just re-run index
codefind index

# Don't clear and re-index unless necessary
# codefind clear  # Avoid this for routine updates
```

### 2. Verify LSPs before large indexing jobs
```bash
# Check LSP status
codefind lsp status

# Install missing LSPs if needed
# Better symbol-based chunks than window fallback
```

### 3. Index from repository root
```bash
# Good: Index entire repository
cd /path/to/repo
codefind index

# Avoid: Indexing subdirectories separately
# cd /path/to/repo/src
# codefind index  # Creates separate index
```

### 4. Monitor progress for large repos
- Indexing shows progress in real-time
- Batch processing prevents memory issues
- Network errors trigger automatic retries

### 5. Keep auth key secure
```bash
# Use keychain storage (recommended)
codefind auth login

# Avoid: Storing in config files or environment variables
```

## Integration with Other Skills

After indexing:
- Use **codefind-search** to search indexed code
- Use **codefind-list-projects** to verify indexing
- Use **codefind-query-cmd** to test queries

Before migrating or refactoring:
- Re-index to ensure latest code is searchable
- Use incremental indexing for fast updates

## Advanced Usage

### Force Full Re-index

```bash
# Clear existing index
codefind clear

# Re-index from scratch
codefind index
```

**When to use:**
- Model configuration changed
- Indexing corrupted or incomplete
- Major refactoring or restructuring

### Index Multiple Projects

```bash
# Index first project
cd /path/to/project-a
codefind index

# Index second project
cd /path/to/project-b
codefind index

# List all indexed projects
codefind list
```

### Check Index Status

```bash
# List all indexed projects
codefind list

# Query to verify content
codefind query "function name"

# Check manifest file
cat ~/.codefind/manifests/<repo-id>.json
```

## Understanding Index Manifest

After indexing, a manifest file is created:

**Location:** `~/.codefind/manifests/<repo-id>.json`

**Contains:**
- Repository path and ID
- Server URL
- Model configuration
- Last indexed timestamp
- Last indexed commit (git repos)
- File inventory (non-git repos)

**Example:**
```json
{
  "repo_id": "code-search-abc123",
  "project_name": "Code-Search",
  "repo_path": "/Users/you/projects/Code-Search",
  "model_id": "unclemusclez/jina-embeddings-v2-base-code",
  "chunk_size_tokens": 450,
  "chunk_overlap_tokens": 50,
  "last_indexed_at": "2026-01-28T10:30:00Z",
  "last_indexed_commit": "abc123def456"
}
```

## Performance Expectations

**Small repo** (10-50 files):
- Indexing time: 30-60 seconds
- Chunks: 100-500
- LSP overhead: Minimal

**Medium repo** (100-500 files):
- Indexing time: 3-8 minutes
- Chunks: 1000-5000
- LSP overhead: Moderate

**Large repo** (1000+ files):
- Indexing time: 15-30 minutes
- Chunks: 10,000+
- LSP overhead: Significant

**Incremental updates:**
- Usually <10% of full index time
- Only processes changed files
- Ideal for routine updates

## Related Skills

- **codefind-search**: Search indexed code
- **codefind-query-cmd**: Use query command
- **codefind-list-projects**: View indexed projects

## Related Commands

- `codefind list-files`: Preview files before indexing
- `codefind clear`: Remove index for a project
- `codefind auth login`: Configure authentication
- `codefind lsp status`: Check LSP availability
