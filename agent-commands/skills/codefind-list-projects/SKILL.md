---
name: codefind-list-projects
description: List and manage indexed code repositories. Use when you need to view all indexed projects, verify a project has been indexed, check index freshness, or find project names for query filtering.
---

# codefind-list-projects

List and manage indexed code repositories.

## Description

This skill helps AI agents view all indexed projects, check indexing status, and understand what code is available for semantic search. It covers the `codefind list` command and related project management operations.

## When to Use

Use this skill when you need to:
- View all indexed projects
- Verify a project has been indexed
- Check when a project was last indexed
- Find project names for query filtering
- Understand what code is available to search
- Troubleshoot missing projects

## Prerequisites

- At least one project indexed with `codefind index`
- Understanding of where your code repositories are located

## Command Syntax

```bash
codefind list
```

**No arguments required** - lists all indexed projects

**Optional flags** (if implemented):
- `--stats`: Show detailed statistics (chunk counts, storage)
- `--json`: Output in JSON format for parsing
- `--verbose`: Show additional metadata

## Basic Usage

### List All Projects

```bash
codefind list
```

**Expected output:**
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

### Understanding the Output

**For each project:**
- **Project Name**: Used with `--project` flag in queries
- **Path**: Absolute path to the repository
- **Indexed**: When the project was last indexed
- **Commit** (git repos): Git commit hash at last index
- **Files**: Number of indexed files
- **Chunks**: Total number of code chunks

**Summary:**
- Total projects indexed
- Total files across all projects
- Total chunks available for search

## Use Cases

### Verify Project Is Indexed

**Check if a specific project appears:**

```bash
codefind list | grep "ProjectName"
```

**If not listed:**
- Project needs to be indexed first
- Run `codefind index` in the project directory

### Find Project Name for Queries

**Use project names from list in queries:**

```bash
# First, list to find exact project name
codefind list

# Then use project name in query
codefind query "authentication" --project="API-Gateway"
```

**Important:**
- Project names are case-sensitive
- Use exact name as shown in `codefind list`
- Names are derived from directory names

### Check Indexing Status

**Verify recent changes are indexed:**

```bash
codefind list
```

**Look for:**
- **Indexed timestamp**: Should be recent if you made changes
- **Commit hash**: Should match current git commit

**If outdated:**
```bash
cd /path/to/project
codefind index  # Run incremental update
```

### Compare Projects

**View all projects to understand your codebase:**

```bash
codefind list
```

**Useful for:**
- Seeing what code is searchable
- Comparing project sizes (files, chunks)
- Identifying projects that need updating
- Planning multi-project queries

## Project Information Details

For detailed explanations of all fields (project name, path, timestamp, commit hash, file count, chunk count, and summary line), see [field-reference.md](field-reference.md).

**Quick field summary:**
- **Project Name**: Derived from directory name, used in `--project` flag
- **Path**: Absolute path to repository
- **Indexed**: Last indexing timestamp (YYYY-MM-DD HH:MM:SS)
- **Commit**: Git commit hash (12 chars, git repos only)
- **Files**: Number of indexed code files
- **Chunks**: Total searchable chunks (LSP: ~5-15/file, Window: ~10-30/file)

## Advanced Usage

### Identify Stale Indexes

**Find projects that haven't been updated recently:**

```bash
codefind list | grep "Indexed: 2026-01"  # Old dates
```

**Or manually check:**
- Compare "Indexed" timestamp to when you last modified code
- Compare "Commit" hash to current `git rev-parse HEAD`

**Update stale indexes:**
```bash
cd /path/to/stale/project
codefind index
```

### Verify Index After Changes

**Workflow:**

1. Make code changes
2. Run incremental index:
   ```bash
   codefind index
   ```
3. Verify update:
   ```bash
   codefind list
   ```
4. Check timestamp and commit hash are current

### Plan Multi-Project Queries

**Review available projects:**

```bash
codefind list
```

**Plan query strategy:**
- Which projects to search?
- Use `--project`, `--projects`, or `--all`?
- Are all relevant projects indexed?

**Example:**
```
Indexed: Code-Search, API-Gateway, Auth-Service

Query plan:
- Search auth code: --projects="API-Gateway,Auth-Service"
- Search everything: --all
- Search specific: --project="Code-Search"
```

## Project Management

### Add New Project

**Index a new project:**

```bash
cd /path/to/new/project
codefind index
```

**Verify it's listed:**
```bash
codefind list | grep "new-project"
```

### Remove Project

**Clear project index:**

```bash
cd /path/to/project
codefind clear
```

**Verify removal:**
```bash
codefind list
# Project should no longer appear
```

**Note:** This removes the index but doesn't delete the code

### Re-index Project

**Complete re-index:**

```bash
cd /path/to/project
codefind clear
codefind index
```

**When to re-index:**
- Corrupted index
- Changed model configuration
- Major refactoring
- Switching between LSP and window chunking

## Troubleshooting

### Issue: Project not listed

**Possible causes:**
- Project hasn't been indexed yet
- Index was cleared
- Manifest file deleted

**Solution:**
```bash
cd /path/to/project
codefind index
codefind list  # Verify it appears
```

### Issue: Outdated information

**Symptoms:**
- Old timestamp
- Old commit hash
- Missing recent files

**Solution:**
```bash
cd /path/to/project
codefind index  # Incremental update
codefind list   # Verify update
```

### Issue: Duplicate projects

**Symptoms:**
- Same project listed twice
- Different paths to same code

**Example:**
```
1. Code-Search
   Path: /Users/you/projects/Code-Search

2. Code-Search
   Path: /home/you/projects/Code-Search
```

**Causes:**
- Indexed from different mount points
- Symlinks to same repo
- Copied repository

**Solution:**
```bash
# Remove duplicate
cd /path/to/duplicate
codefind clear

# Keep only one canonical index
cd /path/to/canonical
codefind index
```

### Issue: Empty list

**If no projects are listed:**

**Verify:**
```bash
# Check for manifest files
ls ~/.codefind/manifests/

# If empty, no projects indexed yet
```

**Solution:**
```bash
# Index your first project
cd /path/to/project
codefind index
codefind list  # Should show project now
```

## Best Practices

1. **Regular checks:**
   - Run `codefind list` before multi-project queries
   - Verify projects are current
   - Update stale indexes

2. **Consistent naming:**
   - Use clear, descriptive directory names
   - Project name comes from directory name
   - Avoid special characters in directory names

3. **Index maintenance:**
   - Run `codefind index` after significant changes
   - Use incremental indexing for routine updates
   - Full re-index only when necessary

4. **Multi-project planning:**
   - Know which projects are indexed
   - Plan query scope based on available projects
   - Index related projects for better cross-project search

## Integration with Other Commands

### Before Querying

```bash
# Check what projects are available
codefind list

# Query specific project
codefind query "function" --project="ProjectName"

# Query multiple projects
codefind query "pattern" --projects="Proj1,Proj2"

# Query all projects
codefind query "code" --all
```

### After Indexing

```bash
# Index project
codefind index

# Verify indexing succeeded
codefind list

# Test with query
codefind query "test query"
```

### For Project Management

```bash
# List all projects
codefind list

# Remove old project
cd /path/to/old/project
codefind clear

# Verify removal
codefind list
```

## Output Formats

For detailed information about output formats, parsing strategies, and automation examples, see [output-formats.md](output-formats.md).

**Available formats:**
- **Standard** (default): Human-readable, visual scanning
- **JSON** (`--json`): Machine-parseable, automation-friendly

**Quick parsing examples:**
```bash
# Standard: Extract project names
codefind list | grep -E "^[0-9]+\." | sed 's/^[0-9]*\. //'

# JSON: Extract project names
codefind list --json | jq -r '.projects[].name'

# JSON: Find stale projects
codefind list --json | jq '.projects[] | select(.indexed_at < "2026-01-20")'
```

## Related Skills

- **codefind-index**: Index projects that will appear in list
- **codefind-search**: Search indexed projects
- **codefind-query-cmd**: Use project names in queries
- **codefind-patterns**: Analyze patterns across listed projects

## Quick Reference

```bash
# List all indexed projects
codefind list

# Find specific project
codefind list | grep "ProjectName"

# After listing, use project name in queries
codefind query "text" --project="ProjectName"

# Check if project needs updating
codefind list  # Check timestamp and commit

# Update project index
cd /path/to/project
codefind index

# Verify update
codefind list
```
