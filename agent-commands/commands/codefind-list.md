# /codefind list

Slash command for listing all indexed code repositories.

## Command Syntax

```bash
/codefind list                  # List all indexed projects
/codefind list --stats          # Include detailed statistics (if implemented)
/codefind list --json           # JSON output (if implemented)
```

## When to Use

Use this slash command when:
- Verifying a project has been indexed
- Finding project names for query filters
- Checking when projects were last updated
- Understanding what code is available to search
- Planning multi-project queries

## Prerequisites

- At least one project indexed with `/codefind index`
- Access to ~/.codefind/manifests/ directory

## Expected Output

### Standard List Format

```
Indexed Projects:

1. Code-Search
   Path: /Users/you/projects/Code-Search
   Indexed: 2026-01-28 10:30:00
   Commit: abc123def456
   Files: 14
   Chunks: 130

2. API-Gateway
   Path: /Users/you/projects/API-Gateway
   Indexed: 2026-01-27 15:45:00
   Commit: def789ghi012
   Files: 45
   Chunks: 678

3. Analytics-Service
   Path: /Users/you/projects/Analytics-Service
   Indexed: 2026-01-25 09:20:00
   Commit: ghi012jkl345
   Files: 32
   Chunks: 456

Total: 3 projects, 91 files, 1,264 chunks
```

**Key information to extract:**
- **Project Name**: Use with --project flag in queries
- **Path**: Absolute path to repository
- **Indexed**: Last indexing timestamp
- **Commit**: Git commit hash (git repos only)
- **Files**: Number of indexed files
- **Chunks**: Total chunks available for search

### Empty List

```
No indexed projects found.

To index a project, run:
  codefind index
```

**Indicates:**
- No projects have been indexed yet
- Need to run `/codefind index` first

### Single Project

```
Indexed Projects:

1. Code-Search
   Path: /Users/you/projects/Code-Search
   Indexed: 2026-01-28 10:30:00
   Commit: abc123def456
   Files: 14
   Chunks: 130

Total: 1 project, 14 files, 130 chunks
```

## Output Fields

### Project Name

**Format:** Directory basename
**Examples:**
- `/Users/you/projects/Code-Search` → `Code-Search`
- `/home/user/my-api-gateway` → `my-api-gateway`

**Usage:**
```bash
# Use in queries
/codefind query "auth" --project="Code-Search"
```

**Important:**
- Names are case-sensitive
- Derived from directory name
- Used in all --project filters

### Path

**Format:** Absolute path to repository root
**Example:** `/Users/you/projects/Code-Search`

**Usage:**
- Navigate to project: `cd /path/to/project`
- Verify correct repository
- Identify duplicate indexes

### Indexed Timestamp

**Format:** `YYYY-MM-DD HH:MM:SS`
**Example:** `2026-01-28 10:30:00`

**Indicates:**
- When project was last indexed
- How fresh the index is
- Whether re-indexing is needed

**Staleness check:**
- Compare to current time
- If > 1 day old, consider re-indexing
- If recent code changes, definitely re-index

### Commit Hash (Git Repos)

**Format:** Short SHA (12 characters)
**Example:** `abc123def456`

**Only for git repositories:**
- Shows which commit was indexed
- Non-git repos won't show this field

**Usage:**
```bash
# Check current commit
git rev-parse HEAD

# Compare to indexed commit
# If different, re-index to include latest changes
```

### File Count

**Example:** `Files: 14`

**Indicates:**
- Number of code files indexed
- Excludes filtered files (.gitignore, build artifacts)

**Typical ranges:**
- Small: 10-50 files
- Medium: 100-500 files
- Large: 1000+ files

### Chunk Count

**Example:** `Chunks: 130`

**Indicates:**
- Total searchable code chunks
- More chunks = more granular search

**Typical ratios:**
- LSP chunking: ~5-15 chunks per file
- Window chunking: ~10-30 chunks per file

### Summary Line

**Format:** `Total: X projects, Y files, Z chunks`
**Example:** `Total: 3 projects, 91 files, 1,264 chunks`

**Provides:**
- Quick overview of indexed codebase
- Total searchable content
- Scope of available projects

## Agent Usage Notes

### Verify Project Exists

**Check if specific project is indexed:**

```bash
# Method 1: Run list and parse
/codefind list | grep "ProjectName"

# Method 2: Run list and check output
/codefind list
# Look for project in output
```

**If not found:**
```bash
cd /path/to/project
/codefind index
```

### Extract Project Names

**For use in queries:**

1. Run `/codefind list`
2. Extract project names from output
3. Use exact names in `--project` flags

**Example workflow:**
```bash
# List projects
/codefind list
# Output shows: Code-Search, API-Gateway, Analytics-Service

# Use in query
/codefind query "auth" --project="API-Gateway"
```

**Important:**
- Names are case-sensitive
- Use exact name as shown in list
- Wrap in quotes if name has spaces

### Check Index Freshness

**Identify stale indexes:**

1. Compare "Indexed" timestamp to current time
2. Compare "Commit" hash to current git commit
3. Re-index if outdated

**Example:**
```bash
# List shows:
#   Indexed: 2026-01-20 09:00:00
#   Commit: abc123

# Current time: 2026-01-28
# Days since index: 8 days → consider re-indexing

# Current commit: def789
# Commit mismatch → definitely re-index
```

### Plan Multi-Project Queries

**Use list to plan query scope:**

```bash
# See what's available
/codefind list

# Output shows 3 projects:
# - Code-Search
# - API-Gateway
# - Auth-Service

# Plan query strategy:
# Option 1: Search all
/codefind query "pattern" --all

# Option 2: Search specific
/codefind query "pattern" --projects="API-Gateway,Auth-Service"

# Option 3: Search one
/codefind query "pattern" --project="Code-Search"
```

### Detect Issues

**No projects listed:**
- Need to index first
- Run `/codefind index` in repository

**Duplicate projects:**
- Same project listed twice with different paths
- Remove duplicate: `cd /path/to/duplicate && codefind clear`

**Very old timestamps:**
- Stale index, need update
- Run `/codefind index` to update

**Unexpected project count:**
- Missing projects: need to index them
- Extra projects: old indexes, consider clearing

## Output Parsing

### Standard Format

**Each project follows this pattern:**
```
<number>. <project_name>
   Path: <absolute_path>
   Indexed: <timestamp>
   Commit: <git_hash>      # (optional, git repos only)
   Files: <count>
   Chunks: <count>
```

**Regex for parsing:**
```regex
^\d+\. (.+)$                          # Project name
^   Path: (.+)$                       # Path
^   Indexed: (.+)$                    # Timestamp
^   Commit: ([a-f0-9]+)$             # Commit hash
^   Files: (\d+)$                     # File count
^   Chunks: (\d+)$                    # Chunk count
```

### Summary Line Parsing

**Pattern:**
```
Total: X projects, Y files, Z chunks
```

**Regex:**
```regex
^Total: (\d+) projects?, (\d+) files?, (\d+) chunks?$
```

**Capture groups:**
1. Project count
2. File count
3. Chunk count

### Success Indicators

- At least one project listed, OR
- "No indexed projects found" message
- Exit code: 0

### Error Indicators

- Error messages in output
- Exit code: non-zero
- Malformed output

## Best Practices for Agents

1. **Check before querying:**
   - Run `/codefind list` to see available projects
   - Verify target project exists
   - Get exact project name (case-sensitive)

2. **Monitor index freshness:**
   - Check "Indexed" timestamps
   - Compare commit hashes (git repos)
   - Re-index if stale

3. **Use for project discovery:**
   - See what code is searchable
   - Understand project scope
   - Plan multi-project searches

4. **Parse output systematically:**
   - Extract project names
   - Note paths for navigation
   - Record timestamps for tracking

5. **Handle edge cases:**
   - Empty list: guide user to index
   - Duplicates: help remove
   - Stale: suggest re-indexing

## Related Commands

**Before listing:**
- `/codefind index` - Index projects

**After listing:**
- `/codefind query "text" --project="Name"` - Query specific project
- `/codefind query "text" --all` - Query all listed projects
- `cd /path/to/project && /codefind index` - Update stale project

**For management:**
- `codefind clear` - Remove project index
- `codefind stats` - Detailed statistics (if available)

## Quick Reference

```bash
# List all projects
/codefind list

# Find specific project
/codefind list | grep "ProjectName"

# After listing, use project name in queries
/codefind query "search" --project="ProjectName"

# Check if project needs updating
/codefind list  # Check timestamp and commit
cd /path/to/project
/codefind index  # Update if needed

# Verify indexing succeeded
/codefind list | grep "NewProject"
```

## Example Workflows

### Workflow 1: Verify Before Query

```bash
# 1. Check what's indexed
/codefind list

# 2. Confirm project exists
# Output shows: Code-Search, API-Gateway

# 3. Query specific project
/codefind query "auth" --project="API-Gateway"
```

### Workflow 2: Update Stale Index

```bash
# 1. Check index status
/codefind list
# Shows: Indexed: 2026-01-20 (8 days ago)

# 2. Navigate to project
cd /Users/you/projects/Code-Search

# 3. Update index
/codefind index

# 4. Verify update
/codefind list
# Shows: Indexed: 2026-01-28 (current)
```

### Workflow 3: Multi-Project Setup

```bash
# 1. List current projects
/codefind list
# Shows: Code-Search

# 2. Index additional project
cd /path/to/api-gateway
/codefind index

# 3. Verify both indexed
/codefind list
# Shows: Code-Search, API-Gateway

# 4. Search across both
/codefind query "pattern" --all
```
