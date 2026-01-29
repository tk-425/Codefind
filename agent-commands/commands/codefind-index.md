# /codefind index

Slash command for indexing code repositories with semantic search.

## Command Syntax

```bash
/codefind index                    # Index current directory
/codefind index /path/to/repo      # Index specific repository
/codefind index --window-only      # Force window-based chunking (skip LSP)
```

## When to Use

Use this slash command when:
- Indexing a new repository for the first time
- Updating an existing index after code changes
- Re-indexing after configuration changes
- Setting up semantic search for a codebase

## Prerequisites

Before running:
1. **Authentication configured**: `codefind auth status` shows ✓
2. **In a code repository**: Run from repository root or specify path
3. **Server accessible**: codefind server is running and reachable

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
  ...

Tokenizing chunks...
  Batch 1/17: 8 chunks ✓ (5%)
  ...
  Batch 17/17: 2 chunks ✓ (100%)

Sending to server...
  Batch 1/17: 8 chunks indexed ✓
  ...
  Batch 17/17: 2 chunks indexed ✓

✓ Successfully indexed 130 chunks from 14 files
✓ Manifest updated: ~/.codefind/manifests/code-search-abc123.json
✓ Last indexed commit: abc123def456
```

**Key information to extract:**
- Total chunks indexed
- Number of files processed
- Languages detected
- Manifest file location
- Last indexed commit (git repos)

### Incremental Update

```
Indexing repository: Code-Search (incremental)

Detecting changes since last index...
  Last indexed: 2026-01-27 14:30:00
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
```

**Key information:**
- Change detection worked (incremental mode)
- Number of modified/added/deleted files
- Old chunks marked as deleted (tombstone mode)
- New chunks indexed

## Error Handling

### Authentication Error

```
Error: Authentication failed (401 Unauthorized)
Please run: codefind auth login
```

**Resolution:**
1. Run `codefind auth login`
2. Enter valid auth key
3. Retry indexing

### Server Unreachable

```
Error: Failed to connect to server at http://100.x.y.z:8000
Please check server is running and network is accessible
```

**Resolution:**
1. Verify server is running
2. Check network connectivity
3. Verify server URL: `codefind config get server_url`
4. Update if needed: `codefind config set server_url http://new-url:8000`

### No Files Found

```
Warning: No files found to index
Repository may be empty or all files are filtered by .gitignore
```

**Resolution:**
1. Check you're in the right directory: `pwd`
2. List discoverable files: `codefind list-files`
3. Review .gitignore rules if needed

### LSP Not Available

```
Checking LSP availability...
  ✗ Python (pyright) not found
Falling back to window-based chunking for Python files
```

**This is normal:**
- Indexing continues with window-based chunking
- No action needed unless you want LSP support
- To add LSP: Install pyright (`npm install -g pyright`)

## Agent Usage Notes

### After Indexing

1. **Verify indexing succeeded:**
   ```bash
   codefind list | grep "ProjectName"
   ```

2. **Test with a query:**
   ```bash
   codefind query "test search"
   ```

3. **Check what was indexed:**
   - Look for "Successfully indexed X chunks" in output
   - Note the manifest file location
   - Record the commit hash (for tracking)

### Incremental Workflow

For routine updates after code changes:

```bash
# Make code changes
# ...

# Re-run index (automatically detects changes)
/codefind index

# Verify update
codefind list  # Check timestamp is current
```

### Flags and Options

**Current directory:**
```bash
/codefind index
```

**Specific path:**
```bash
/codefind index /path/to/repository
```

**Force window chunking (skip LSP):**
```bash
/codefind index --window-only
```
- Use when LSP is slow or failing
- Uses fixed-size windows instead of symbols
- No symbol metadata in results

## Output Parsing

For automated agents, key fields to extract:

**Success indicators:**
- `✓ Successfully indexed N chunks`
- `✓ Manifest updated`
- Exit code: 0

**Metadata:**
- Total chunks: Extract number from "Successfully indexed X chunks"
- File count: Extract from "Found X files to index"
- Languages: Extract from language breakdown
- Commit hash: Extract from "Last indexed commit: XXXXX"

**Failure indicators:**
- `Error:` prefix in output
- `Failed to connect`
- `Authentication failed`
- Exit code: non-zero

## Related Commands

**Before indexing:**
- `codefind auth status` - Verify authentication
- `codefind list-files` - Preview what will be indexed
- `codefind lsp status` - Check LSP availability

**After indexing:**
- `codefind list` - Verify project appears
- `codefind query "test"` - Test search functionality
- `codefind stats` - View indexing statistics

## Performance Expectations

**Small repo (10-50 files):**
- Time: 30-60 seconds
- Chunks: 100-500

**Medium repo (100-500 files):**
- Time: 3-8 minutes
- Chunks: 1000-5000

**Large repo (1000+ files):**
- Time: 15-30 minutes
- Chunks: 10,000+

**Incremental updates:**
- Usually <10% of full index time
- Only processes changed files

## Best Practices for Agents

1. **Always verify prerequisites:**
   - Check auth status before indexing
   - Verify you're in the right directory

2. **Handle errors gracefully:**
   - Parse error messages
   - Provide actionable feedback to users
   - Don't retry authentication errors without user input

3. **Track indexing state:**
   - Note the manifest file location
   - Record commit hash for git repos
   - Save indexing timestamp

4. **Use incremental updates:**
   - Run `/codefind index` again after changes
   - Don't clear and re-index unless necessary

5. **Verify success:**
   - Check for success indicators in output
   - Verify project appears in `codefind list`
   - Test with a simple query

## Quick Reference

```bash
# Basic usage
/codefind index

# Specific path
/codefind index /path/to/repo

# Force window chunking
/codefind index --window-only

# Verify success
codefind list | grep "ProjectName"

# Test search
codefind query "test"
```
