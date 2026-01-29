# Output Formats

Different output formats for `codefind list` and their use cases.

## Standard Format (Human-Readable)

The default output format, optimized for human readability.

### Single Project

```
Indexed Projects:

1. ProjectName
   Path: /full/path/to/project
   Indexed: 2026-01-28 10:30:00
   Commit: abc123def456
   Files: 14
   Chunks: 130

Total: 1 project, 14 files, 130 chunks
```

**Structure:**
- Header: "Indexed Projects:"
- Numbered list (1, 2, 3...)
- Indented fields (3 spaces)
- Summary line at end

### Multiple Projects

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

**Features:**
- Easy to scan visually
- Clear separation between projects
- Blank lines between entries
- Summary provides totals

### Empty List

```
No indexed projects found.

To index a project, run:
  codefind index
```

**Indicates:**
- No projects indexed yet
- Helpful message with next steps

## JSON Format

Structured output for programmatic parsing and automation.

**Command:**
```bash
codefind list --json
```

### Single Project JSON

```json
{
  "projects": [
    {
      "name": "Code-Search",
      "path": "/Users/you/projects/Code-Search",
      "indexed_at": "2026-01-28T10:30:00Z",
      "commit": "abc123def456",
      "file_count": 14,
      "chunk_count": 130
    }
  ],
  "total_projects": 1,
  "total_files": 14,
  "total_chunks": 130
}
```

**Field mapping:**
- `name`: Project name (string)
- `path`: Absolute path (string)
- `indexed_at`: ISO 8601 timestamp (string)
- `commit`: Git commit hash (string, null for non-git)
- `file_count`: Number of files (integer)
- `chunk_count`: Number of chunks (integer)

**Summary fields:**
- `total_projects`: Total number of indexed projects
- `total_files`: Sum of all file counts
- `total_chunks`: Sum of all chunk counts

### Multiple Projects JSON

```json
{
  "projects": [
    {
      "name": "Code-Search",
      "path": "/Users/you/projects/Code-Search",
      "indexed_at": "2026-01-28T10:30:00Z",
      "commit": "abc123def456",
      "file_count": 14,
      "chunk_count": 130
    },
    {
      "name": "API-Gateway",
      "path": "/Users/you/projects/API-Gateway",
      "indexed_at": "2026-01-27T15:45:00Z",
      "commit": "def789ghi012",
      "file_count": 45,
      "chunk_count": 678
    },
    {
      "name": "Analytics-Service",
      "path": "/Users/you/projects/Analytics-Service",
      "indexed_at": "2026-01-25T09:20:00Z",
      "commit": "ghi012jkl345",
      "file_count": 32,
      "chunk_count": 456
    }
  ],
  "total_projects": 3,
  "total_files": 91,
  "total_chunks": 1264
}
```

### Empty List JSON

```json
{
  "projects": [],
  "total_projects": 0,
  "total_files": 0,
  "total_chunks": 0
}
```

## Parsing Standard Format

### Extract Project Names

**Using grep:**
```bash
codefind list | grep -E "^[0-9]+\." | sed 's/^[0-9]*\. //'
```

**Output:**
```
Code-Search
API-Gateway
Analytics-Service
```

**Using awk:**
```bash
codefind list | awk '/^[0-9]+\./ {print $2}'
```

### Extract Paths

```bash
codefind list | grep "Path:" | awk '{print $2}'
```

**Output:**
```
/Users/you/projects/Code-Search
/Users/you/projects/API-Gateway
/Users/you/projects/Analytics-Service
```

### Extract Timestamps

```bash
codefind list | grep "Indexed:" | awk '{print $2, $3}'
```

**Output:**
```
2026-01-28 10:30:00
2026-01-27 15:45:00
2026-01-25 09:20:00
```

### Extract Commit Hashes

```bash
codefind list | grep "Commit:" | awk '{print $2}'
```

**Output:**
```
abc123def456
def789ghi012
ghi012jkl345
```

## Parsing JSON Format

### Using jq

**Extract project names:**
```bash
codefind list --json | jq -r '.projects[].name'
```

**Output:**
```
Code-Search
API-Gateway
Analytics-Service
```

**Extract paths:**
```bash
codefind list --json | jq -r '.projects[].path'
```

**Get project count:**
```bash
codefind list --json | jq '.total_projects'
```

**Filter by file count > 20:**
```bash
codefind list --json | jq '.projects[] | select(.file_count > 20)'
```

**Find stale projects (indexed before specific date):**
```bash
codefind list --json | jq '.projects[] | select(.indexed_at < "2026-01-20")'
```

### Using Python

```python
import json
import subprocess

# Run command
result = subprocess.run(['codefind', 'list', '--json'], capture_output=True, text=True)
data = json.loads(result.stdout)

# Extract project names
names = [p['name'] for p in data['projects']]
print(names)

# Filter large projects (>100 files)
large = [p for p in data['projects'] if p['file_count'] > 100]

# Get total stats
print(f"Total: {data['total_projects']} projects, {data['total_chunks']} chunks")
```

### Using JavaScript/Node

```javascript
const { execSync } = require('child_process');

// Run command
const output = execSync('codefind list --json', { encoding: 'utf8' });
const data = JSON.parse(output);

// Extract project names
const names = data.projects.map(p => p.name);
console.log(names);

// Find projects indexed today
const today = new Date().toISOString().split('T')[0];
const recent = data.projects.filter(p => p.indexed_at.startsWith(today));
```

## Output for Scripting

### Check if Project Exists

**Bash:**
```bash
if codefind list | grep -q "ProjectName"; then
  echo "Project is indexed"
else
  echo "Project not indexed"
fi
```

**With JSON:**
```bash
if codefind list --json | jq -e '.projects[] | select(.name == "ProjectName")' > /dev/null; then
  echo "Project is indexed"
else
  echo "Project not indexed"
fi
```

### Count Projects

**Standard format:**
```bash
codefind list | grep -c "^[0-9]*\."
```

**JSON format:**
```bash
codefind list --json | jq '.total_projects'
```

### Get Total Chunks

**Standard format:**
```bash
codefind list | tail -1 | grep -oE '[0-9,]+ chunks' | tr -d ',' | awk '{print $1}'
```

**JSON format:**
```bash
codefind list --json | jq '.total_chunks'
```

## Use Cases by Format

### Standard Format Best For

- **Interactive CLI usage**: Quick visual review
- **Human readability**: Easy to scan and understand
- **Quick checks**: "Is project X indexed?"
- **Manual workflows**: When copying project names for queries

**Example workflow:**
```bash
# List projects
codefind list

# Pick project name from output
# Use in query
codefind query "auth" --project="ProjectName"
```

### JSON Format Best For

- **Automation scripts**: Reliable programmatic parsing
- **CI/CD pipelines**: Checking index status in builds
- **Monitoring**: Tracking index freshness over time
- **Complex filtering**: Finding projects by criteria

**Example workflow:**
```bash
# Find stale projects in automation
codefind list --json | jq -r '.projects[] | select(.indexed_at < "2026-01-20") | .name'

# Auto-reindex stale projects
for project in $(codefind list --json | jq -r '...'); do
  cd "$project_path" && codefind index
done
```

## Format Comparison

| Feature | Standard | JSON |
|---------|----------|------|
| Human-readable | ✅ Excellent | ❌ Poor |
| Machine-parseable | ⚠️ Possible | ✅ Excellent |
| Stable format | ⚠️ May change | ✅ Stable schema |
| Compact | ✅ Yes | ❌ More verbose |
| Supports filtering | ⚠️ Via grep/awk | ✅ Via jq/scripts |
| Best for | Interactive use | Automation |

## Output Examples for Different Scenarios

### Fresh Repository

```
1. Active-Project
   Path: /Users/you/active/project
   Indexed: 2026-01-28 14:30:00  ← Today
   Commit: abc123def456           ← Current commit
   Files: 50
   Chunks: 650
```

**Status:** ✅ Fresh and current

### Stale Repository

```
2. Old-Project
   Path: /Users/you/old/project
   Indexed: 2025-11-10 09:15:00  ← 79 days ago
   Commit: xyz789old000           ← Old commit
   Files: 120
   Chunks: 1500
```

**Status:** ⚠️ Needs re-indexing

### Non-Git Repository

```
3. Legacy-App
   Path: /Users/you/legacy/app
   Indexed: 2026-01-27 10:00:00
   (no Commit field)              ← Not a git repo
   Files: 200
   Chunks: 2800
```

**Status:** ℹ️ Non-git repository

### Large Repository

```
4. Monorepo
   Path: /Users/you/company/monorepo
   Indexed: 2026-01-28 08:00:00
   Commit: def456ghi789
   Files: 5,420                   ← Large project
   Chunks: 68,500                 ← Many chunks
```

**Status:** ℹ️ Large codebase
