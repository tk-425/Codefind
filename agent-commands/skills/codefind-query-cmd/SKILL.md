---
name: codefind-query-cmd
description: Execute semantic code search queries with precise filters. Use when you need to search code with specific project, language, or path filters, handle pagination, or understand query command syntax.
---

# codefind-query-cmd

Execute semantic code search queries using the codefind query command.

## Description

This skill provides detailed guidance on using the `codefind query` command for semantic code search. It covers query syntax, filtering options, result interpretation, and pagination. This complements the **codefind-search** skill by focusing specifically on command-line usage and options.

## When to Use

Use this skill when you need to:
- Execute specific queries with precise filters
- Understand query command syntax and flags
- Filter results by project, language, or path
- Use pagination to navigate many results
- Search across multiple projects
- Troubleshoot query issues

## Prerequisites

- At least one project indexed with `codefind index`
- Understanding of what you're searching for
- Basic familiarity with command-line tools

## Command Syntax

```bash
codefind query <query-text> [flags]
```

**Required:**
- `<query-text>`: Natural language description of what you're looking for

**Optional flags:**
- `--project=<name>`: Search specific project
- `--projects=<name1,name2>`: Search multiple projects
- `--all`: Search all indexed projects
- `--lang=<language>`: Filter by programming language
- `--path=<prefix>`: Filter by file path prefix
- `--top-k=<n>`: Number of results to retrieve (default: 10)
- `--page=<n>`: Page number for pagination (default: 1)
- `--page-size=<n>`: Results per page (default: 20)

## Basic Usage

### Simple Query

```bash
codefind query "JWT authentication"
```

**Expected output:**
```
Results for: "JWT authentication"

1. [Code-Search] src/auth/jwt.py:45-67 (score: 0.89)
   Function: validate_jwt_token
   Validates JWT tokens and extracts user claims

2. [Code-Search] tests/auth_test.py:30-55 (score: 0.75)
   Function: test_jwt_validation
   Unit tests for JWT validation logic
```

**What this returns:**
- Results from the current project (if in a repo directory)
- Top 20 results by default
- Ranked by semantic similarity (0.0 to 1.0)
- Each result shows: project, file, line range, score, symbol info

### Query with Natural Language

```bash
codefind query "how to handle database connection errors"
```

**Good query characteristics:**
- Descriptive and specific
- Uses natural language
- Describes intent, not just keywords

**Examples:**
- ✓ "validate user input for SQL injection"
- ✓ "error handling with retry logic"
- ✓ "authentication middleware for API endpoints"
- ✗ "validate sql" (too generic)
- ✗ "error" (too vague)

## Filtering Options

### Filter by Project

**Search specific project:**
```bash
codefind query "database queries" --project="Code-Search"
```

**Search multiple projects:**
```bash
codefind query "authentication" --projects="API-Gateway,Auth-Service"
```

**Search all projects:**
```bash
codefind query "error handling patterns" --all
```

**When to use:**
- `--project`: You know which project has the code
- `--projects`: Compare implementations across specific projects
- `--all`: Find all instances across your codebase

### Filter by Language

```bash
# Python only
codefind query "async functions" --lang=python

# Go only
codefind query "error handling" --lang=go

# TypeScript only
codefind query "React components" --lang=typescript
```

**Supported languages:**
- `python`, `go`, `typescript`, `javascript`, `java`, `swift`
- Language detection is based on file extension

**Use case:**
- Find language-specific implementations
- Avoid results from other languages
- Learn patterns in a specific language

### Filter by Path

```bash
# Only search in src/api directory
codefind query "endpoints" --path=src/api

# Only search in internal directory
codefind query "handlers" --path=internal

# Search in tests
codefind query "test examples" --path=tests
```

**Use case:**
- Limit search to specific module or package
- Find code in particular directory structure
- Separate production code from tests

### Combine Filters

```bash
# Python code in API directory
codefind query "validation" --lang=python --path=api

# Authentication across all projects, Go only
codefind query "auth middleware" --all --lang=go

# Error handling in specific project and directory
codefind query "error handling" --project="API" --path=internal
```

## Pagination

For detailed pagination guidance including page size customization, navigation strategies, and browsing patterns, see [pagination-guide.md](pagination-guide.md).

**Quick reference:**
```bash
# Default pagination (20 per page)
codefind query "database"

# Custom page size
codefind query "database" --page-size=30

# Navigate pages
codefind query "database" --page=2

# Combine
codefind query "database" --page=2 --page-size=30
```

**Page size limits:** 1-50 results (default: 20)

## Result Interpretation

### Result Format

```
<id>. [<project>] <file>:<start_line>-<end_line> (score: <similarity>)
   <symbol_kind>: <symbol_name>
   <preview_text>
```

**Example:**
```
1. [API-Gateway] internal/handlers/auth.go:120-145 (score: 0.92)
   Function: ValidateToken
   Validates JWT token and extracts user claims
```

**Fields explained:**
- **1**: Result ID (use with `codefind open 1`)
- **[API-Gateway]**: Project name
- **internal/handlers/auth.go**: File path relative to project root
- **120-145**: Line range (start-end)
- **(score: 0.92)**: Similarity score (0.0 to 1.0, higher is better)
- **Function**: Symbol kind (Function, Class, Method, etc.)
- **ValidateToken**: Symbol name
- **Preview**: First line or description of the code

### Similarity Scores

**Score interpretation:**
- **0.90 - 1.00**: Excellent match, very relevant
- **0.75 - 0.89**: Good match, likely relevant
- **0.60 - 0.74**: Moderate match, possibly relevant
- **0.50 - 0.59**: Weak match, review carefully
- **< 0.50**: Poor match, likely not relevant

**Tips:**
- Focus on results with scores > 0.75
- Lower scores may still be useful for exploratory searches
- Different queries may have different score distributions

## Advanced Query Techniques

### Specific vs. Broad Queries

**Specific (recommended for finding exact implementations):**
```bash
codefind query "JWT token validation with expiration check"
```

**Broad (useful for exploration):**
```bash
codefind query "authentication"
```

**When to be specific:**
- You know exactly what you're looking for
- Need precise implementations
- Want to reduce noise in results

**When to be broad:**
- Exploring unfamiliar codebase
- Finding all instances of a pattern
- Learning how something is implemented

### Query Refinement

**If results aren't relevant:**

1. **Add more context:**
   ```bash
   # Too broad
   codefind query "validation"

   # Better
   codefind query "input validation for API requests"
   ```

2. **Use domain-specific terms:**
   ```bash
   # Generic
   codefind query "check password"

   # Domain-specific
   codefind query "bcrypt password hashing validation"
   ```

3. **Add filters:**
   ```bash
   # Add language filter
   codefind query "async functions" --lang=python

   # Add path filter
   codefind query "handlers" --path=api
   ```

### Multi-Project Queries

**Find all implementations across projects:**

```bash
# All error handling across all projects
codefind query "error handling patterns" --all

# Compare authentication across specific projects
codefind query "user authentication" --projects="API,Auth-Service,Gateway"
```

**Analyze results:**
- Group by project to see different approaches
- Compare similarity scores across projects
- Identify best implementation to standardize on

## Output Redirection

**Save results to file:**
```bash
codefind query "API endpoints" > results.txt
```

**Pipe to other commands:**
```bash
# Count results
codefind query "functions" | grep -c "Function:"

# Filter results
codefind query "auth" | grep "JWT"
```

## Common Queries

For comprehensive query examples including functions, classes, patterns, API endpoints, configuration, tests, security, data handling, async code, logging, and framework-specific examples, see [query-examples.md](query-examples.md).

**Quick examples:**
```bash
# Find functions
codefind query "function that validates email addresses"

# Find classes
codefind query "user model class"

# Find patterns
codefind query "error handling with retry logic"

# Find API endpoints
codefind query "POST endpoint for user creation"

# Find configuration
codefind query "database connection settings"
```

## Troubleshooting

### Issue: No results found

**Possible causes:**
- Project not indexed
- Query too specific
- Wrong project selected

**Solutions:**
```bash
# Verify project is indexed
codefind list

# Try broader query
codefind query "broader search term"

# Search all projects
codefind query "term" --all

# Re-index if needed
codefind index
```

### Issue: Too many irrelevant results

**Solutions:**
```bash
# Add filters
codefind query "term" --lang=python --path=src

# Make query more specific
codefind query "very specific description of what you need"

# Reduce page size to review top results
codefind query "term" --page-size=5
```

### Issue: Results from wrong project

**Solutions:**
```bash
# Specify project explicitly
codefind query "term" --project="CorrectProject"

# List projects to find correct name
codefind list
```

### Issue: Can't find recent changes

**Solution:**
```bash
# Re-index to pick up recent changes
cd /path/to/repo
codefind index

# Then query again
codefind query "recent feature"
```

## Best Practices

1. **Start simple, then refine:**
   - First: Broad query without filters
   - Then: Add filters based on initial results

2. **Use natural language:**
   - Describe what you're looking for
   - Include context and intent
   - Use complete phrases, not just keywords

3. **Leverage filters:**
   - Use `--lang` when language is known
   - Use `--path` to limit scope
   - Use `--project` when working in multi-project setup

4. **Review scores:**
   - Focus on high-score results first
   - Don't ignore moderate scores completely
   - Low scores often indicate query needs refinement

5. **Iterate on queries:**
   - If results aren't good, rephrase
   - Try different terminology
   - Add or remove specificity

## Integration with Other Commands

**After querying:**
```bash
# View specific result in editor
codefind open <id>

# Read the full file
cat <file-path>

# Find all usages of a function
grep -r "function_name" .
```

**Before querying:**
```bash
# Verify projects are indexed
codefind list

# Check what's available to search
codefind list

# Update index if needed
codefind index
```

## Related Skills

- **codefind-search**: High-level search workflow and strategies
- **codefind-index**: Index repositories before querying
- **codefind-list-projects**: View available projects
- **codefind-patterns**: Analyze patterns across results
- **codefind-summarize**: Summarize found code

## Quick Reference

```bash
# Basic query
codefind query "search text"

# Filter by project
codefind query "text" --project="ProjectName"

# Filter by language
codefind query "text" --lang=python

# Filter by path
codefind query "text" --path=src/api

# Search all projects
codefind query "text" --all

# Pagination
codefind query "text" --page=2 --page-size=30

# Combine filters
codefind query "text" --project="API" --lang=go --path=internal
```
