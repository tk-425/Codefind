---
name: codefind
description: Semantic code search for conceptual questions when the exact symbol name is unknown. Use for high-level "where is" questions, pattern discovery, and cross-project comparisons. Do NOT use if you know the exact function/variable name—use CodeGraph instead.
---

# CodeFind Query Command

## Overview

CodeFind is a semantic code search system that finds relevant code snippets by matching semantic meaning rather than keywords. The `codefind query` command enables intelligent cross-project code search with natural language descriptions, language and path filtering, and result pagination. Supports Go, Python, TypeScript, JavaScript, Java, and other languages indexed by the CodeFind server.

## When to Use CodeFind Query

Use CodeFind for **conceptual searches** when you don't know the exact symbol name:
- **"Where do we implement X?"** - "Where is authentication implemented?"
- **"Find pattern X"** - "Find error handling with retry logic"
- **"Compare implementations"** - "How do different services handle database connections?"
- **"Explore by concept"** - "Show me validation patterns in this service"
- **"Search broad topics"** - "What code deals with payment processing?"

**Do NOT use CodeFind if you know the exact function/variable/class name** — use CodeGraph instead.

## When NOT to Use CodeFind

**Use CodeGraph instead if:**
- You know the exact function, variable, or class name
- You need to trace call chains (who calls what)
- You need function signatures or parameters
- You need to find implementations of an interface
- You're doing impact analysis on a specific symbol

Example: Don't use CodeFind for "Who calls authenticate?" — use CodeGraph search instead.

## Prerequisites

Before querying, ensure:
1. **Global config is set up** with server URL:
   ```bash
   codefind config -g
   ```
2. **At least one project is indexed** by a project manager
3. **Server is running** and accessible via Tailscale or configured network

**Note:** Individual users do NOT need authentication to query—queries are read-only operations.

## Core Command

### Search Code Semantically

Perform semantic searches across indexed projects using natural language descriptions.

**Command:**
```bash
codefind query <query-text> [options]
```

**Options:**
| Option | Purpose | Example |
|--------|---------|---------|
| `--project=<name>` | Search single project | `--project="API-Service"` |
| `--projects=<list>` | Search multiple projects | `--projects="API,Auth,Gateway"` |
| `--all` | Search all indexed projects | `--all` |
| `--lang=<language>` | Filter by language | `--lang=python` |
| `--path=<prefix>` | Filter by directory | `--path=api/handlers` |
| `--page=<n>` | Page number | `--page=2` |
| `--page-size=<n>` | Results per page (1-50) | `--page-size=30` |

**Examples:**
```bash
# Basic query (current project)
codefind query "JWT token validation with expiration"

# Search all projects
codefind query "database connection retry logic" --all

# Filter by language and path
codefind query "request validation" --path=handlers --lang=go

# Multiple projects with pagination
codefind query "authentication" --projects="API,Auth" --page=2 --page-size=30
```

## Writing Effective Queries

**Effective queries** are specific and descriptive:
- ✓ `"JWT token validation with expiration checking"`
- ✓ `"database connection pool with retry and timeout"`
- ✓ `"error handling that wraps context and logs"`

**Ineffective queries** are too vague:
- ✗ `"auth"` — Too short
- ✗ `"database"` — Too generic
- ✗ `"error"` — Too vague

**Refinement strategy:** Start broad, add technical details based on results, add filters if still too broad, then target specific area with `--project` and `--path`.

## Understanding Results

### Result Format

```
<id>. [<project>] <file>:<lines> (score: <similarity>)
   <kind>: <name>
   <description>
```

**Example:**
```
1. [API-Service] src/auth/middleware.go:45-68 (score: 0.92)
   Function: ValidateJWT
   Validates JWT token signature and extracts user claims
```

### Similarity Scores

| Score | Quality | Action |
|-------|---------|--------|
| 0.90–1.00 | Excellent | Exactly what you need—prioritize |
| 0.75–0.89 | Good | Very relevant—highly useful |
| 0.60–0.74 | Moderate | Possibly relevant—worth checking |
| < 0.60 | Weak | Refine your query |

**Low scores indicate:** Query may be too vague or terminology mismatch. Use more specific technical terms.

## Filtering Strategy

**Start without filters** - Get broad results first
**Add filters based on results** - Refine if needed
**Combine filters for precision** - Project + language + path for maximum focus

### By Project Scope
```bash
codefind query "database queries"                           # Current project
codefind query "database queries" --project="ServiceName"  # Single project
codefind query "authentication" --projects="API,Auth"      # Multiple projects
codefind query "error handling" --all                       # All indexed projects
```

### By Language or Path
```bash
codefind query "async error handling" --lang=python
codefind query "handlers" --path=api
codefind query "validation" --lang=go --path=internal
```

## Common Workflow

**Find and understand implementation:**
```bash
codefind query "JWT token validation with expiration" --project="API-Service"
codefind open 1                    # Open top result in editor
```

**Compare implementations across services:**
```bash
codefind query "database connection pool retry" --all
# Results show different approaches from each service
```

## Troubleshooting

| Problem | Solution |
|---------|----------|
| No results | Check project name with `codefind list`, try simpler query terms, use `--all` to search broader |
| Low scores (< 0.60) | Use more specific technical terms, add more context to query |
| Too many results | Add language filter (`--lang`), path filter (`--path`), or make query more specific |
| Wrong project | Verify project name with `codefind list`, use `--project` or `--projects` to target |

## Best Practices

- **Use natural language with technical terms** - Not just keywords
- **Start broad, add filters incrementally** - Refine based on results
- **Trust high-scoring matches** - Prioritize results > 0.75
- **Combine filters for precision** - Project + language + path together

## Language Support

Go, Python, TypeScript, JavaScript, Java, Swift, Rust

## Related Skills

- **codefind-list**: View indexed projects
- **codegraph**: Symbol search and call graph analysis (complementary tool)

---

**Key takeaway:** Use natural language queries with specific technical terms, start without filters, add filters based on results, trust high-scoring matches (> 0.75), and refine queries when results are weak (< 0.60).
