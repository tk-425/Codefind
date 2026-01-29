# codefind-search

Search codebase semantically using natural language queries.

## Description

This skill enables AI agents to perform semantic code search across indexed projects using codefind. It provides a structured approach to searching for code implementations, patterns, and specific functionality using natural language queries.

## When to Use

Use this skill when you need to:
- Find implementations of specific features or functions
- Locate code handling particular operations (e.g., "authentication", "error handling")
- Search for API endpoints or database queries
- Discover code patterns across one or multiple projects
- Find examples of how a library or framework is used

## Prerequisites

Before using this skill:
1. Verify codefind is installed and available in PATH
2. Ensure at least one project has been indexed with `codefind index`
3. Check indexed projects with `codefind list`

## Usage

### Basic Search Workflow

1. **Run the search query**
   ```bash
   codefind query "your search query here"
   ```

2. **Parse the results**
   - Results include: project name, file path, line numbers, similarity score
   - Each result has an ID number for easy reference
   - Results are ranked by semantic similarity (highest score first)

3. **Examine relevant code**
   - Use `codefind open <id>` to open a specific result in your editor
   - Or use the Read tool to examine the file at the specified line range

### Search Options

**Filter by project:**
```bash
codefind query "authentication" --project="Code-Search"
```

**Filter by language:**
```bash
codefind query "error handling" --lang=python
```

**Filter by file path:**
```bash
codefind query "api endpoints" --path=src/api
```

**Search across all projects:**
```bash
codefind query "JWT validation" --all
```

**Pagination:**
```bash
codefind query "database queries" --page=2 --page-size=20
```

## Examples

### Example 1: Find Authentication Implementation

**Query:**
```bash
codefind query "JWT token validation"
```

**Expected Output:**
```
Results for: "JWT token validation"

1. [Code-Search] src/auth/jwt.py:45-67 (score: 0.89)
   Function: validate_jwt_token
   Validates JWT tokens and extracts user claims

2. [API-Gateway] internal/middleware/auth.go:120-145 (score: 0.82)
   Function: JWTMiddleware
   Middleware for validating JWT tokens in requests

3. [Code-Search] tests/auth_test.py:30-55 (score: 0.75)
   Function: test_jwt_validation
   Unit tests for JWT validation logic
```

**Next Steps:**
- Use `codefind open 1` to examine the main implementation
- Read the file to understand the validation logic
- Check tests in result #3 for usage examples

### Example 2: Find Error Handling Patterns

**Query:**
```bash
codefind query "error handling patterns" --lang=go
```

**Expected Output:**
```
Results for: "error handling patterns" (filtered by language: go)

1. [API-Gateway] internal/handlers/user.go:78-95 (score: 0.91)
   Function: handleUserError
   Centralized error handling for user operations

2. [Code-Search] internal/indexer/indexer.go:210-230 (score: 0.85)
   Function: handleIndexError
   Error handling with retry logic
```

### Example 3: Search Across Multiple Projects

**Query:**
```bash
codefind query "database connection pooling" --all
```

**Expected Output:**
```
Results for: "database connection pooling" (searching all projects)

1. [API-Gateway] internal/db/pool.go:15-45 (score: 0.93)
   Class: ConnectionPool
   PostgreSQL connection pool implementation

2. [Analytics-Service] src/database/mysql_pool.py:20-50 (score: 0.87)
   Class: MySQLPool
   MySQL connection pool with health checks

3. [Code-Search] server/database/chromadb.py:30-55 (score: 0.78)
   Function: get_connection
   ChromaDB connection management
```

## Output Format

Each search result contains:
- **Result ID**: Sequential number for easy reference (1, 2, 3...)
- **Project Name**: In brackets [ProjectName]
- **File Path**: Relative path from project root
- **Line Range**: start_line-end_line
- **Similarity Score**: 0.0 to 1.0 (higher is better)
- **Symbol Info**: Function/class name and kind (if available)
- **Content Preview**: First few lines of the matched code

## Best Practices for Agents

1. **Start broad, then narrow:**
   - First search without filters to see what's available
   - Then refine with --project, --lang, or --path filters

2. **Use semantic queries:**
   - Good: "validate user authentication tokens"
   - Better than: "auth validate token"
   - The tool understands natural language context

3. **Check multiple results:**
   - Don't stop at the first result
   - Different projects may have different approaches
   - Lower-scored results might still be relevant

4. **Combine with file reading:**
   - After finding relevant code, read the full file for context
   - Check surrounding functions and imports
   - Look for related test files

5. **Iterate on queries:**
   - If results aren't relevant, rephrase the query
   - Try different terminology or more specific descriptions
   - Use domain-specific terms when appropriate

## Troubleshooting

**No results found:**
- Verify the project is indexed: `codefind list`
- Try broader or different search terms
- Check if the code you're looking for exists in indexed files

**Too many results:**
- Use filters: --project, --lang, --path
- Make query more specific
- Reduce --page-size or use pagination

**Low similarity scores:**
- Results below 0.5 may not be relevant
- Rephrase query to be more specific
- Consider the domain terminology used in the codebase

## Integration with Other Tools

After finding code with codefind-search:
- Use **Read** tool to examine the full file
- Use **Grep** to find all usages of a function/class
- Use **Glob** to find related files
- Use **codefind open** to jump directly to the code in an editor

## Related Skills

- **codefind-patterns**: For analyzing recurring code patterns
- **codefind-migrate**: For finding similar implementations to migrate
- **codefind-summarize**: For generating summaries of found code
