# Field Reference

Detailed explanation of all fields in `codefind list` output.

## Project Name

**How it's determined:**
- Extracted from repository directory name
- Example: `/Users/you/projects/Code-Search` → `Code-Search`
- Example: `/home/user/my-api-gateway` → `my-api-gateway`

**Used in:**
- `--project` flag for queries
- Display in search results
- Project identification

**Important notes:**
- Names are case-sensitive
- Use exact name as shown in list
- Derived from final directory component

**Examples:**

| Full Path | Project Name |
|-----------|-------------|
| `/Users/you/projects/Code-Search` | `Code-Search` |
| `/home/user/api-gateway` | `api-gateway` |
| `/workspace/MyProject` | `MyProject` |
| `/repos/my-cool-app` | `my-cool-app` |

## Path

**Absolute path to repository:**
- Shows where the code is located on disk
- Useful for knowing which directory to `cd` into
- Helps identify duplicate indexes of same code

**Format:** Always absolute path (starts with `/` or drive letter)

**Use cases:**
- Navigate to project: `cd /path/to/project`
- Verify correct repository is indexed
- Identify duplicate indexes

**Examples:**

```
Path: /Users/you/projects/Code-Search
Path: /home/user/workspace/api-gateway
Path: C:\Users\you\projects\analytics
```

**Troubleshooting:**
- **Same project, different paths**: May indicate duplicate indexes
- **Path doesn't exist**: Index may be stale or directory was moved
- **Symlink paths**: May show actual path, not symlink

## Last Indexed Timestamp

**When the project was last indexed:**
- Shows exact date and time of last `codefind index` run
- Format: `YYYY-MM-DD HH:MM:SS`
- Uses local timezone

**Example:**
```
Indexed: 2026-01-28 10:30:00
```

**Indicates:**
- How fresh the index is
- Whether recent changes are searchable
- When to run incremental update

**Staleness check:**

| Age | Status | Action |
|-----|--------|--------|
| < 1 hour | Fresh | No action needed |
| 1-24 hours | Recent | Monitor for changes |
| 1-7 days | Aging | Consider re-indexing if active development |
| > 7 days | Stale | Recommend re-indexing |
| > 30 days | Very stale | Definitely re-index |

**How to check:**
```bash
# List to see timestamp
codefind list

# Compare to current time
date

# If outdated, re-index
cd /path/to/project
codefind index
```

## Commit Hash (Git Repos)

**Last indexed git commit:**
- Shows which commit was indexed
- Format: Short SHA (first 12 characters)
- Example: `abc123def456`
- Only appears for git repositories

**Example:**
```
Commit: abc123def456
```

**Use to:**
- Verify index is up-to-date with current branch
- Understand which version of code is indexed
- Detect if re-indexing is needed after commits

**Note:** Non-git repos won't show this field

**Checking if current:**

```bash
# See indexed commit
codefind list

# See current commit
git rev-parse HEAD

# Compare
# If different, re-index to include latest changes
```

**Examples:**

| Scenario | Indexed Commit | Current Commit | Action |
|----------|---------------|----------------|--------|
| Up-to-date | `abc123` | `abc123` | No action |
| Behind | `abc123` | `def456` | Re-index |
| Detached HEAD | `abc123` | (detached) | Verify intent |

**Common scenarios:**
- **Match**: Index is current with HEAD
- **Mismatch**: New commits since last index, re-index needed
- **Missing field**: Not a git repository

## File Count

**Number of indexed files:**
- Includes only code files
- Excludes files filtered by .gitignore
- Excludes build artifacts, dependencies, etc.

**Example:**
```
Files: 14
```

**Typical counts:**

| Project Size | File Count |
|-------------|-----------|
| Small | 10-50 files |
| Medium | 100-500 files |
| Large | 1,000-5,000 files |
| Very Large | 10,000+ files |

**What's included:**
- Source code files (.py, .go, .ts, .js, etc.)
- Configuration files (.yaml, .json, .toml)
- Documentation (.md, .rst)

**What's excluded:**
- Generated files (build artifacts)
- Dependencies (node_modules, vendor)
- Binary files (images, executables)
- Files matching .gitignore patterns

**Troubleshooting:**
- **Fewer files than expected**: Check .gitignore, some may be filtered
- **More files than expected**: May include generated or test files
- **Zero files**: Indexing failed or directory is empty

## Chunk Count

**Total number of code chunks:**
- Each chunk is a searchable unit
- LSP-based: Usually aligned with functions/classes/methods
- Window-based: Fixed-size overlapping windows
- More chunks = more granular search

**Example:**
```
Chunks: 130
```

**Typical ratios:**

| Chunking Method | Chunks per File |
|----------------|-----------------|
| LSP (symbol-based) | 5-15 chunks/file |
| Window (fixed-size) | 10-30 chunks/file |

**Depends on:**
- File size (larger files = more chunks)
- Code complexity (more functions = more chunks)
- Chunking method (LSP vs window)

**Examples:**

| Files | LSP Chunks | Window Chunks |
|-------|-----------|---------------|
| 10 files | 50-150 chunks | 100-300 chunks |
| 100 files | 500-1,500 chunks | 1,000-3,000 chunks |
| 1,000 files | 5,000-15,000 chunks | 10,000-30,000 chunks |

**LSP Chunking (Symbol-based):**
- Chunks align with language symbols (functions, classes, methods)
- Better semantic boundaries
- Variable chunk sizes based on code structure
- Typical: 5-15 chunks per file

**Window Chunking (Fixed-size):**
- Fixed-size sliding windows with overlap
- Consistent chunk sizes
- Better coverage, may split symbols
- Typical: 10-30 chunks per file

**Use chunk count to:**
- Estimate search granularity
- Compare indexing methods
- Understand index size
- Plan query expectations

## Summary Line

**Format:** `Total: X projects, Y files, Z chunks`

**Example:**
```
Total: 3 projects, 91 files, 1,264 chunks
```

**Provides:**
- Quick overview of entire indexed codebase
- Total searchable content across all projects
- Scope of what's available for semantic search

**Interpretation:**

| Total Projects | Meaning |
|---------------|---------|
| 1 | Single project indexed |
| 2-5 | Small multi-project setup |
| 6-20 | Medium multi-project setup |
| 20+ | Large multi-project setup |

**Use for:**
- Understanding total searchable codebase
- Planning query strategies
- Capacity planning
- Comparing index sizes over time

## Field Variations

### Git Repository

```
1. Code-Search
   Path: /Users/you/projects/Code-Search
   Indexed: 2026-01-28 10:30:00
   Commit: abc123def456        ← Git commit hash present
   Files: 14
   Chunks: 130
```

### Non-Git Repository

```
1. Legacy-Project
   Path: /Users/you/legacy/project
   Indexed: 2026-01-28 10:30:00
   (no Commit field)            ← No commit hash for non-git repos
   Files: 25
   Chunks: 180
```

### LSP vs Window Chunking

**LSP-based (fewer, semantic chunks):**
```
Files: 14
Chunks: 130    (ratio: 9.3 chunks/file)
```

**Window-based (more, overlapping chunks):**
```
Files: 14
Chunks: 280    (ratio: 20 chunks/file)
```

## Reading the Output

### Complete Example

```
Indexed Projects:

1. Code-Search
   Path: /Users/you/projects/Code-Search
   Indexed: 2026-01-28 10:30:00
   Commit: abc123def456
   Files: 14
   Chunks: 130
```

**Quick assessment:**
- **Name**: Code-Search (use in `--project=Code-Search`)
- **Location**: /Users/you/projects/Code-Search
- **Freshness**: Indexed today at 10:30 AM (very fresh)
- **Version**: Commit abc123def456 (verify with `git rev-parse HEAD`)
- **Size**: Small project (14 files)
- **Granularity**: 130 chunks (about 9 chunks per file, LSP-based)

**Status**: ✅ Fresh, ready to query

### Stale Example

```
2. Old-API
   Path: /Users/you/projects/old-api
   Indexed: 2025-12-15 14:20:00
   Commit: xyz789abc123
   Files: 45
   Chunks: 520
```

**Quick assessment:**
- **Name**: Old-API
- **Freshness**: Indexed 44 days ago (very stale)
- **Action needed**: ⚠️ Re-index recommended

### Multiple Projects

```
Indexed Projects:

1. Frontend (Fresh)
2. Backend (Fresh)
3. Shared (Aging)
4. Legacy (Stale)

Total: 4 projects
```

**Assessment:**
- 2 fresh, 1 aging, 1 stale
- Review aging/stale projects
- Consider selective re-indexing
