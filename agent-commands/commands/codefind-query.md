# /codefind query

Slash command for semantic code search across indexed repositories.

## Command Syntax

```bash
/codefind query "search text"                      # Basic search
/codefind query "text" --project="ProjectName"     # Specific project
/codefind query "text" --projects="Proj1,Proj2"    # Multiple projects
/codefind query "text" --all                       # All projects
/codefind query "text" --lang=python               # Filter by language
/codefind query "text" --path=src/api              # Filter by path
/codefind query "text" --top-k=20                  # Number of results
/codefind query "text" --page=2 --page-size=30     # Pagination
```

## When to Use

Use this slash command when:
- Searching for specific code implementations
- Finding functions, classes, or patterns
- Locating API endpoints or configurations
- Discovering how features are implemented
- Learning from existing code examples

## Prerequisites

- At least one project indexed with `/codefind index`
- Understanding of what you're searching for
- Project name (if using --project filter)

## Expected Output

### Basic Query Results

```
Results for: "JWT authentication"

1. [Code-Search] src/auth/jwt.py:45-67 (score: 0.89)
   Function: validate_jwt_token
   Validates JWT tokens and extracts user claims

2. [API-Gateway] internal/middleware/auth.go:120-145 (score: 0.82)
   Function: JWTMiddleware
   Middleware for validating JWT tokens in requests

3. [Code-Search] tests/auth_test.py:30-55 (score: 0.75)
   Function: test_jwt_validation
   Unit tests for JWT validation logic

Showing results 1-3 of 3
```

**Key information to extract:**
- Result ID (1, 2, 3...) - use with `codefind open <id>`
- Project name in brackets [ProjectName]
- File path relative to project root
- Line range (start-end)
- Similarity score (0.0 to 1.0)
- Symbol type (Function, Class, Method, etc.)
- Symbol name
- Brief description/preview

### Filtered Query Results

```
Results for: "error handling" (filtered by language: go, path: internal/)

1. [API-Gateway] internal/errors/handler.go:15-45 (score: 0.91)
   Class: ErrorHandler
   Centralized error handling with logging

2. [Code-Search] internal/indexer/errors.go:10-35 (score: 0.85)
   Function: handleIndexError
   Error handling with retry logic

Showing results 1-2 of 2
Filters: language=go, path=internal/
```

**Additional information:**
- Active filters are shown
- Result count reflects filtered subset

### Paginated Results

```
Results for: "database queries"

1. [API-Gateway] internal/db/queries.go:20-45 (score: 0.93)
   ...

20. [Analytics] src/queries/reports.py:60-85 (score: 0.68)
   ...

Showing results 1-20 of 47
Use --page=2 to see more results
```

**Pagination info:**
- Current page range
- Total result count
- Hint for next page

### Multi-Project Results

```
Results for: "authentication" (searching all projects)

1. [API-Gateway] src/auth/jwt.go:15-45 (score: 0.94)
   Function: ValidateJWT
   JWT token validation

2. [Auth-Service] internal/oauth/handler.go:30-60 (score: 0.89)
   Function: HandleOAuth
   OAuth2 authentication flow

3. [Code-Search] server/routes/auth.py:45-70 (score: 0.82)
   Function: authenticate_request
   Request authentication middleware

Showing results 1-3 of 15
Projects searched: API-Gateway, Auth-Service, Code-Search
```

**Multi-project indicators:**
- Results from different projects
- Project names in brackets
- "Projects searched" footer

## Filters and Options

### Project Filters

**Single project:**
```bash
/codefind query "validation" --project="API-Gateway"
```

**Multiple projects:**
```bash
/codefind query "auth" --projects="API-Gateway,Auth-Service"
```

**All projects:**
```bash
/codefind query "error handling" --all
```

### Language Filter

```bash
/codefind query "async functions" --lang=python
/codefind query "interfaces" --lang=go
/codefind query "components" --lang=typescript
```

**Supported languages:**
- python, go, typescript, javascript, java, swift

### Path Filter

```bash
/codefind query "handlers" --path=api
/codefind query "models" --path=src/db
/codefind query "tests" --path=tests
```

**Path matching:**
- Prefix match on file path
- Relative to repository root
- Case-sensitive

### Pagination

```bash
# First page (default)
/codefind query "functions"

# Specific page
/codefind query "functions" --page=2

# Custom page size
/codefind query "functions" --page-size=10

# Combined
/codefind query "functions" --page=3 --page-size=50
```

**Limits:**
- Default page size: 20
- Max page size: 50
- Page numbers start at 1

### Combined Filters

```bash
/codefind query "validation" --project="API" --lang=go --path=internal
/codefind query "endpoints" --all --lang=python --page=2
```

## Result Interpretation

### Similarity Scores

**Score ranges:**
- **0.90-1.00**: Excellent match (highly relevant)
- **0.75-0.89**: Good match (likely relevant)
- **0.60-0.74**: Moderate match (possibly relevant)
- **0.50-0.59**: Weak match (review carefully)
- **< 0.50**: Poor match (likely not relevant)

**Agent guidance:**
- Focus on scores > 0.75 for most tasks
- Lower scores may be useful for exploration
- Score distribution varies by query

### Result Fields

**Each result contains:**
```
<id>. [<project>] <file>:<start>-<end> (score: <similarity>)
   <symbol_kind>: <symbol_name>
   <preview>
```

**Example breakdown:**
```
1. [API-Gateway] internal/handlers/auth.go:120-145 (score: 0.92)
   Function: ValidateToken
   Validates JWT token and extracts user claims
```

- **1**: Result ID for `codefind open 1`
- **[API-Gateway]**: Project name
- **internal/handlers/auth.go**: File path
- **120-145**: Line range
- **(score: 0.92)**: Similarity score
- **Function**: Symbol kind
- **ValidateToken**: Symbol name
- **Validates...**: Description/preview

## Error Handling

### No Results Found

```
Results for: "nonexistent code"

No results found.

Suggestions:
- Try broader search terms
- Check if the code exists in indexed projects
- Verify projects are indexed: codefind list
```

**Agent actions:**
1. Verify projects are indexed
2. Suggest broader query
3. Check if re-indexing is needed

### Project Not Found

```
Error: Project "NonExistentProject" not found
Available projects:
- Code-Search
- API-Gateway
- Auth-Service

Use: codefind list
```

**Resolution:**
1. List available projects: `codefind list`
2. Use exact project name (case-sensitive)
3. Verify project is indexed

### Invalid Filters

```
Error: Invalid language "invalid-lang"
Supported languages: python, go, typescript, javascript, java, swift
```

**Resolution:**
- Use supported language names
- Check spelling
- Refer to language list

## Agent Usage Notes

### Query Construction

**Good queries:**
- Descriptive and specific
- Natural language descriptions
- Include intent and context

**Examples:**
```bash
✓ "validate user input for SQL injection"
✓ "error handling with retry logic"
✓ "authentication middleware for API endpoints"
✗ "validate sql" (too generic)
✗ "error" (too vague)
```

### Result Processing

**Extract key information:**
1. Result ID for opening files
2. File path for reading
3. Line range for context
4. Similarity score for ranking
5. Project name for multi-project searches

**Follow-up actions:**
```bash
# Open result in editor
codefind open 1

# Read the file
cat /path/to/file

# Find related code
grep -r "function_name" .
```

### Pagination Strategy

**For comprehensive review:**
1. Start with page 1
2. Review high-score results
3. Navigate to next pages as needed
4. Stop when scores drop below threshold

**Code:**
```bash
# Page 1
/codefind query "pattern" --page=1

# Review results, if more needed:
/codefind query "pattern" --page=2
```

### Multi-Project Queries

**When to use each:**

**--project:** Know which project has the code
```bash
/codefind query "auth" --project="API-Gateway"
```

**--projects:** Compare across specific projects
```bash
/codefind query "validation" --projects="API,Auth-Service"
```

**--all:** Find all instances
```bash
/codefind query "error handling" --all
```

## Output Parsing

### Structured Format

Results follow a consistent format for parsing:

**Pattern:**
```
<id>. [<project>] <file>:<start>-<end> (score: <similarity>)
   <symbol_kind>: <symbol_name>
   <description>
```

**Regex for parsing:**
```regex
^(\d+)\. \[([^\]]+)\] ([^:]+):(\d+)-(\d+) \(score: ([\d.]+)\)
```

**Capture groups:**
1. ID
2. Project
3. File path
4. Start line
5. End line
6. Score

### Success Indicators

- Results displayed (even if 0 results)
- "Showing results X-Y of Z" footer
- Exit code: 0

### Error Indicators

- "Error:" prefix in output
- "No results found" (not an error, just empty)
- Exit code: non-zero for actual errors

## Best Practices for Agents

1. **Use natural language:**
   - Describe what you're looking for
   - Include context and intent
   - Be specific but not overly constrained

2. **Start broad, refine:**
   - First query: no filters
   - Assess results
   - Add filters to narrow down

3. **Check scores:**
   - Focus on high scores (> 0.75)
   - Don't ignore moderate scores completely
   - Very low scores (<0.50) usually not relevant

4. **Handle pagination:**
   - Process page by page
   - Don't assume all results on page 1
   - Stop when scores drop significantly

5. **Verify project names:**
   - Use `codefind list` to get exact names
   - Project names are case-sensitive
   - Use quotes for names with spaces

6. **Combine with other tools:**
   - Open results with `codefind open <id>`
   - Read files with Read tool
   - Search for usages with Grep

## Related Commands

**Before querying:**
- `codefind list` - See available projects
- `codefind index` - Ensure code is indexed

**After querying:**
- `codefind open <id>` - Open result in editor
- Read/Grep/Glob - Examine results in detail

## Quick Reference

```bash
# Basic search
/codefind query "search text"

# Filter by project
/codefind query "text" --project="ProjectName"

# Filter by language
/codefind query "text" --lang=python

# Filter by path
/codefind query "text" --path=src/api

# Search all projects
/codefind query "text" --all

# Pagination
/codefind query "text" --page=2 --page-size=30

# Combined filters
/codefind query "text" --project="API" --lang=go --path=internal
```
