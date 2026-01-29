# codefind-patterns

Analyze code patterns across indexed projects to understand architectural decisions and recurring implementations.

## Description

This skill helps AI agents identify, analyze, and understand code patterns across one or multiple projects. It's particularly useful for understanding how common problems are solved, finding architectural patterns, and ensuring consistency across a codebase.

## When to Use

Use this skill when you need to:
- Find all error handling patterns in a codebase
- Locate all authentication/authorization implementations
- Identify API endpoint definitions across services
- Discover configuration management patterns
- Analyze how a particular library or framework is used
- Find recurring design patterns (factory, singleton, observer, etc.)
- Understand logging, monitoring, or observability patterns
- Identify security patterns (input validation, encryption, etc.)

## Prerequisites

- Multiple projects indexed for cross-project pattern analysis
- Or a large codebase with consistent patterns
- Run `codefind list` to verify indexed projects

## Usage

### Pattern Discovery Workflow

1. **Identify the pattern you're looking for**
   - Define what constitutes the pattern
   - Choose appropriate search terms

2. **Search for pattern instances**
   ```bash
   codefind query "pattern description" --all
   ```

3. **Analyze the results**
   - Group similar implementations
   - Note variations and differences
   - Identify best practices and anti-patterns

4. **Document findings**
   - Summarize common approaches
   - Note edge cases and variations
   - Recommend standardization opportunities

## Common Pattern Searches

### Error Handling Patterns

**Find error handling implementations:**
```bash
# Find error wrappers and handlers
codefind query "error handling wrapper custom errors" --lang=go

# Find try-catch patterns
codefind query "exception handling try catch finally" --lang=python

# Find error logging patterns
codefind query "error logging structured logging" --all
```

**Analysis Questions:**
- Are errors wrapped with context?
- Is there centralized error handling?
- How are errors logged and monitored?
- Are custom error types used?

### Authentication Patterns

**Find auth implementations:**
```bash
# Find JWT authentication
codefind query "JWT authentication token validation" --all

# Find session management
codefind query "session management user sessions" --lang=python

# Find OAuth implementations
codefind query "OAuth2 authentication flow" --all
```

**Analysis Questions:**
- What auth mechanisms are used? (JWT, sessions, OAuth)
- How are tokens validated and refreshed?
- Where is auth middleware implemented?
- How are permissions/roles handled?

### API Endpoint Patterns

**Find API definitions:**
```bash
# Find REST endpoints
codefind query "REST API endpoints routes handlers" --path=api

# Find GraphQL resolvers
codefind query "GraphQL resolvers mutations queries" --all

# Find request validation
codefind query "request validation input validation" --path=api
```

**Analysis Questions:**
- How are routes defined and organized?
- What validation is applied?
- How are responses formatted?
- Is there consistent error handling?

### Configuration Patterns

**Find config management:**
```bash
# Find environment config
codefind query "environment variables configuration" --all

# Find config files
codefind query "configuration settings management" --all

# Find feature flags
codefind query "feature flags feature toggles" --all
```

**Analysis Questions:**
- How is configuration loaded and validated?
- Are there different configs for different environments?
- How are secrets managed?
- Is there a config hierarchy?

## Examples

### Example 1: Analyze Error Handling Patterns

**Step 1: Search for error handling**
```bash
codefind query "error handling custom errors" --lang=go
```

**Results:**
```
1. [API-Gateway] internal/errors/handler.go:15-45
   Custom error types and error wrapping

2. [Code-Search] internal/indexer/errors.go:10-35
   Error definitions with context

3. [Analytics] pkg/errors/errors.go:20-50
   Error handling with retry logic
```

**Step 2: Read implementations**
- Read each file to understand the pattern
- Note: All use wrapped errors with context
- All define custom error types

**Step 3: Synthesize findings**
```
Pattern: Custom Error Wrapping
- All services define custom error types
- Errors include context (file, function, operation)
- Common: RetryableError, ValidationError, NotFoundError
- All use fmt.Errorf with %w for wrapping
Recommendation: Create shared error package
```

### Example 2: Find Database Connection Patterns

**Step 1: Search for DB connections**
```bash
codefind query "database connection initialization" --all
```

**Results:**
```
1. [API-Gateway] internal/db/postgres.go:25-60
   PostgreSQL connection pool setup

2. [Analytics] src/db/mysql.py:15-45
   MySQL connection with retry

3. [Code-Search] server/database/chromadb.py:10-40
   ChromaDB HTTP client initialization
```

**Step 2: Analyze patterns**
- Pattern: Connection pooling in all services
- Pattern: Health checks on startup
- Pattern: Graceful shutdown handlers
- Variation: Different retry strategies

**Step 3: Document best practices**
```
Database Connection Pattern:
✓ Use connection pooling
✓ Implement health checks
✓ Handle reconnection
✓ Graceful shutdown
✗ Inconsistent retry strategies (needs standardization)
```

### Example 3: API Security Patterns

**Step 1: Find auth middleware**
```bash
codefind query "authentication middleware JWT validation" --all
```

**Step 2: Find input validation**
```bash
codefind query "input validation sanitization" --path=api
```

**Step 3: Find rate limiting**
```bash
codefind query "rate limiting throttling" --all
```

**Analysis:**
```
Security Pattern Analysis:
- Auth: JWT-based in all services
- Validation: Inconsistent (some use libraries, some custom)
- Rate Limiting: Only in API-Gateway (missing in others)
Recommendations:
1. Standardize input validation library
2. Add rate limiting to all public APIs
3. Centralize auth middleware
```

## Pattern Analysis Framework

When analyzing patterns, consider:

### 1. Consistency
- Are similar problems solved similarly?
- Are there unnecessary variations?
- Could patterns be standardized?

### 2. Completeness
- Are all necessary aspects covered?
- Are there missing patterns (e.g., no logging)?
- Are edge cases handled?

### 3. Quality
- Are patterns well-implemented?
- Are there anti-patterns or code smells?
- Is error handling robust?

### 4. Documentation
- Are patterns documented?
- Are there examples and tests?
- Is usage clear?

### 5. Evolution
- Are patterns outdated?
- Should they be modernized?
- Are there better alternatives?

## Cross-Project Pattern Analysis

For multi-service architectures:

1. **Search across all projects:**
   ```bash
   codefind query "pattern description" --all
   ```

2. **Group results by project:**
   - Note which projects have the pattern
   - Which projects are missing it?

3. **Compare implementations:**
   - What are the differences?
   - Which is the best implementation?
   - Should others adopt it?

4. **Identify standardization opportunities:**
   - Create shared libraries for common patterns
   - Document standard approaches
   - Refactor outliers

## Best Practices

1. **Search broadly first:**
   - Use --all to see patterns across projects
   - Don't limit to one language initially

2. **Look for variations:**
   - Similar patterns may use different terminology
   - Try multiple related queries

3. **Read multiple implementations:**
   - Don't stop at the first result
   - Compare at least 3-5 implementations

4. **Document your findings:**
   - Create pattern documentation
   - Include examples and anti-examples
   - Note when to use each variation

5. **Check for tests:**
   - Find test files for patterns
   - Tests show expected usage
   - Tests reveal edge cases

## Pattern Query Templates

**Error Handling:**
- "error handling custom errors retry logic"
- "exception handling try catch finally"
- "error wrapping context errors"

**Authentication:**
- "authentication middleware JWT bearer token"
- "session management cookie sessions"
- "OAuth2 authorization code flow"

**Logging:**
- "structured logging log levels context"
- "logging middleware request logging"
- "log aggregation correlation ID"

**Caching:**
- "cache implementation Redis memcache"
- "cache invalidation cache strategy"
- "memoization caching decorator"

**Validation:**
- "input validation schema validation"
- "data sanitization XSS prevention"
- "request validation middleware"

## Related Skills

- **codefind-search**: For finding specific implementations
- **codefind-migrate**: For applying patterns to new code
- **codefind-summarize**: For documenting discovered patterns
