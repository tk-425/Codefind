# Codefind Agent Skills

This directory contains reusable skills for AI agents (Claude, Gemini, etc.) to use with the codefind semantic code search tool.

## Available Skills

### Command Skills (Phase 3B.1)

Core command-line operations for codefind.

#### 1. codefind-index
**Purpose:** Index code repositories for semantic search

**Use when:**
- Indexing a new repository for the first time
- Updating an existing index after code changes
- Troubleshooting indexing issues

**Key Features:**
- LSP-based symbol chunking (primary method)
- Window-based fallback chunking
- Incremental indexing for fast updates
- Authentication management
- Progress tracking and error handling

**Location:** `codefind-index/SKILL.md`

---

#### 2. codefind-query-cmd
**Purpose:** Execute semantic code search queries

**Use when:**
- Searching with precise filters and options
- Understanding query command syntax
- Using pagination to navigate results
- Filtering by project, language, or path

**Key Features:**
- Natural language query support
- Multi-project search capabilities
- Advanced filtering options
- Pagination for large result sets
- Result interpretation guidance

**Location:** `codefind-query-cmd/SKILL.md`

---

#### 3. codefind-list-projects
**Purpose:** List and manage indexed repositories

**Use when:**
- Viewing all indexed projects
- Verifying a project has been indexed
- Finding project names for query filtering
- Checking indexing status and timestamps

**Key Features:**
- Project overview with statistics
- Index status verification
- Project management operations
- Stale index identification

**Location:** `codefind-list-projects/SKILL.md`

---

### Workflow Skills (Phase 3B.2)

High-level workflows and specialized use cases.

#### 4. codefind-search
**Purpose:** Search codebase semantically using natural language queries

**Use when:**
- Finding implementations of specific features
- Locating code handling particular operations
- Discovering code patterns across projects
- Searching for API endpoints or database queries

**Key Features:**
- Semantic search workflow guidance
- Result interpretation strategies
- Integration with editor (codefind open)
- Best practices for effective searching

**Location:** `codefind-search/SKILL.md`

---

#### 5. codefind-patterns
**Purpose:** Analyze code patterns across indexed projects

**Use when:**
- Finding all error handling patterns
- Locating authentication/authorization implementations
- Identifying API endpoint definitions
- Discovering configuration management patterns
- Analyzing framework usage patterns

**Key Features:**
- Cross-project pattern analysis
- Pattern comparison framework
- Standardization recommendations
- Common pattern query templates

**Location:** `codefind-patterns/SKILL.md`

---

#### 6. codefind-migrate
**Purpose:** Assist with code migration tasks

**Use when:**
- Migrating code between projects
- Upgrading libraries or frameworks
- Refactoring to new patterns
- Porting implementations to different languages
- Extracting shared code into libraries

**Key Features:**
- Migration workflow guidance
- Dependency analysis
- Similar code discovery
- Migration strategy templates
- Rollback planning

**Location:** `codefind-migrate/SKILL.md`

---

#### 7. codefind-summarize
**Purpose:** Generate summaries and documentation from code

**Use when:**
- Understanding unfamiliar codebases
- Generating documentation for code sections
- Explaining component relationships
- Creating architecture documentation
- Documenting API endpoints

**Key Features:**
- Function and module summarization
- Component relationship documentation
- API documentation generation
- Summary templates
- Architecture documentation

**Location:** `codefind-summarize/SKILL.md`

---

## Installation

These skills are portable and can be copied to:

### User-level (all projects)
```bash
mkdir -p ~/.claude/skills/
cp -r codefind-* ~/.claude/skills/
```

### Project-level (single project)
```bash
mkdir -p .claude/skills/
cp -r /path/to/agent-commands/skills/codefind-* .claude/skills/
```

### Plugin-level (plugin-specific)
```bash
cp -r /path/to/agent-commands/skills/codefind-* <plugin-dir>/skills/
```

## Usage for AI Agents

Each skill provides:
- Clear description and purpose
- Prerequisites and requirements
- Step-by-step workflow
- Common scenarios and examples
- Best practices
- Troubleshooting guidance
- Related skills

Agents should read the SKILL.md file to understand how to use each skill effectively.

## Skill Categories

### Command Skills
Focus on specific codefind CLI commands:
- **codefind-index**: Indexing operations
- **codefind-query-cmd**: Query command syntax and options
- **codefind-list-projects**: Project listing and management

### Workflow Skills
Focus on high-level workflows and use cases:
- **codefind-search**: General search workflow
- **codefind-patterns**: Pattern analysis across projects
- **codefind-migrate**: Code migration assistance
- **codefind-summarize**: Documentation generation

## Integration with Codefind

All skills work with the core codefind CLI commands:
- `codefind init` - Initialize configuration
- `codefind index` - Index a repository
- `codefind query` - Semantic search
- `codefind list` - List indexed projects
- `codefind open` - Open file in editor
- `codefind auth` - Manage authentication
- `codefind lsp` - LSP diagnostics

## Skill Relationships

```
                    codefind-index
                          |
                          v
              codefind-list-projects
                          |
                          v
        +----------------+----------------+
        |                |                |
        v                v                v
codefind-search   codefind-query-cmd   (verify)
        |                |
        v                v
   +---------+----------+---------+
   |         |          |         |
   v         v          v         v
patterns  migrate  summarize   (analyze)
```

**Typical flow:**
1. **Index** code with `codefind-index`
2. **List** projects with `codefind-list-projects`
3. **Search** with `codefind-search` or `codefind-query-cmd`
4. **Analyze** with `codefind-patterns`, `codefind-migrate`, or `codefind-summarize`

## Contributing

When creating new skills:
1. Create a directory: `codefind-<skill-name>/`
2. Add SKILL.md with clear structure
3. Include prerequisites and workflow
4. Provide concrete examples
5. Document best practices
6. Update this README

## Phase 3B Completion Status

### Step 3B.1: Agent Slash Commands ✅
✅ codefind-index skill created (489 lines)
✅ codefind-query-cmd skill created (542 lines)
✅ codefind-list-projects skill created (548 lines)

### Step 3B.2: Custom Skills for Agents ✅
✅ codefind-search skill created (211 lines)
✅ codefind-patterns skill created (339 lines)
✅ codefind-migrate skill created (391 lines)
✅ codefind-summarize skill created (451 lines)

**Total Skills:** 7 skills
**Total Documentation:** 2,971 lines across all skills

All Phase 3B requirements complete! 🎉
