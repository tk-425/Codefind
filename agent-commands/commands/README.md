# Codefind Slash Commands

This directory contains workflow documentation for codefind slash commands that AI agents can use.

## Available Slash Commands

### 1. /codefind index
**File:** `codefind-index.md`
**Purpose:** Index code repositories for semantic search

**Basic usage:**
```bash
/codefind index                    # Index current directory
/codefind index /path/to/repo      # Index specific path
/codefind index --window-only      # Force window chunking
```

**When to use:**
- First-time repository indexing
- Updating index after code changes
- Re-indexing with different settings

**Key outputs:**
- Total chunks indexed
- Files processed count
- LSP availability status
- Manifest file location
- Commit hash (git repos)

---

### 2. /codefind query
**File:** `codefind-query.md`
**Purpose:** Search indexed code semantically

**Basic usage:**
```bash
/codefind query "search text"
/codefind query "text" --project="ProjectName"
/codefind query "text" --all
/codefind query "text" --lang=python --path=src
```

**When to use:**
- Finding specific implementations
- Searching for patterns or features
- Locating API endpoints or functions
- Cross-project code discovery

**Key outputs:**
- Ranked search results
- File paths and line ranges
- Similarity scores
- Symbol information
- Pagination details

---

### 3. /codefind list
**File:** `codefind-list.md`
**Purpose:** List all indexed projects

**Basic usage:**
```bash
/codefind list                     # Show all indexed projects
```

**When to use:**
- Verifying project indexing status
- Finding project names for filters
- Checking index freshness
- Planning multi-project queries

**Key outputs:**
- Project names
- Repository paths
- Last indexed timestamps
- Commit hashes
- File and chunk counts

---

## Slash Command vs. Skills

### Slash Commands (This Directory)
**Location:** `agent-commands/commands/`
**Format:** Workflow markdown files
**Focus:** Quick reference for command syntax and expected outputs
**Target:** Fast lookups during agent execution

### Skills
**Location:** `agent-commands/skills/`
**Format:** Comprehensive SKILL.md files
**Focus:** Detailed workflows, best practices, troubleshooting
**Target:** In-depth guidance for complex tasks

**Relationship:**
- Slash commands: "How to run the command"
- Skills: "How to accomplish the task"

## Usage by AI Agents

### When to Use Slash Command Docs

Read these files when you need:
- Command syntax and flags
- Expected output format
- Quick error handling
- Output parsing patterns
- Basic usage examples

### Workflow

1. **Read command doc** to understand syntax
2. **Execute command** via Bash tool
3. **Parse output** using documented patterns
4. **Handle errors** based on error handling section
5. **Reference skills** for advanced workflows

## Document Structure

Each slash command file includes:

1. **Command Syntax** - All available options and flags
2. **When to Use** - Appropriate use cases
3. **Prerequisites** - Requirements before running
4. **Expected Output** - Sample outputs with annotations
5. **Filters and Options** - Detailed flag explanations
6. **Error Handling** - Common errors and resolutions
7. **Agent Usage Notes** - Specific guidance for agents
8. **Output Parsing** - Regex patterns and extraction methods
9. **Best Practices** - Guidelines for effective usage
10. **Related Commands** - Workflow integration
11. **Quick Reference** - Common command patterns

## Integration with Codefind CLI

All slash commands map directly to codefind CLI commands:

```bash
/codefind index    → codefind index
/codefind query    → codefind query
/codefind list     → codefind list
```

Agents execute these via the Bash tool:
```bash
codefind index
codefind query "search text" --project="Name"
codefind list
```

## Output Parsing Guidance

All command docs include:

**Structured formats:**
- Line-by-line output patterns
- Field descriptions
- Parsing hints

**Regex patterns:**
- Extraction patterns for key fields
- Capture group definitions
- Examples

**Success/error indicators:**
- How to detect success
- Error message patterns
- Exit codes

## Common Workflows

### Initial Setup
```bash
# 1. Index a repository
/codefind index

# 2. Verify indexing
/codefind list

# 3. Test search
/codefind query "test"
```

### Routine Search
```bash
# 1. Check available projects
/codefind list

# 2. Search with filters
/codefind query "pattern" --project="Name" --lang=python

# 3. Review results and open relevant code
codefind open 1
```

### Multi-Project Analysis
```bash
# 1. List all projects
/codefind list

# 2. Search across all
/codefind query "error handling" --all

# 3. Filter results by project and language
/codefind query "auth" --projects="API,Auth" --lang=go
```

### Update Workflow
```bash
# 1. Check index status
/codefind list
# Note: Check timestamps

# 2. Update stale project
cd /path/to/project
/codefind index

# 3. Verify update
/codefind list
```

## Phase 3B.1 Completion Status

✅ **codefind-index.md** (306 lines)
   - Command syntax and options
   - Expected outputs (success, incremental, errors)
   - Authentication and LSP handling
   - Agent usage notes and parsing guidance

✅ **codefind-query.md** (471 lines)
   - Complete query syntax with all filters
   - Result format and interpretation
   - Pagination and multi-project queries
   - Output parsing patterns and examples

✅ **codefind-list.md** (464 lines)
   - List output format
   - Field descriptions
   - Index freshness checking
   - Project management workflows

**Total:** 3 slash commands, 1,241 lines of documentation

All Phase 3B.1 requirements complete! ✅

## Related Documentation

- **Skills:** `../skills/` - Comprehensive workflow guides
- **Implementation Plan:** `../../.docs/Phases/Phase-3-B.md`
- **CLI Reference:** `../../.docs/CLI-COMMANDS-AND-WORKFLOW.md`
