# codefind-migrate

Assist with code migration tasks by finding similar implementations and suggesting migration strategies.

## Description

This skill helps AI agents facilitate code migration tasks, including migrating code between projects, upgrading to new libraries/frameworks, refactoring patterns, and porting implementations to different languages. It leverages semantic search to find similar code and dependencies to guide migration decisions.

## When to Use

Use this skill when you need to:
- Migrate code from one project to another
- Upgrade a library or framework to a newer version
- Refactor code to use a new pattern or architecture
- Port implementations to a different language
- Extract shared code into a common library
- Replace deprecated APIs with new ones
- Consolidate duplicate implementations

## Prerequisites

- Source and target projects indexed with `codefind index`
- Understanding of the migration goal and constraints
- Knowledge of what code needs to be migrated

## Usage

### Migration Workflow

1. **Identify source code to migrate**
   - Find the implementation in the source project
   - Understand its dependencies and usage

2. **Search for similar implementations**
   - Search target project for similar patterns
   - Find examples of how it's done in the target context

3. **Analyze dependencies**
   - Find all code that depends on what you're migrating
   - Identify external dependencies that need to be ported

4. **Create migration plan**
   - Document what needs to change
   - Identify potential conflicts or issues
   - Plan migration steps

5. **Execute and verify**
   - Implement the migration
   - Test thoroughly
   - Update documentation

## Migration Scenarios

### Scenario 1: Migrate Feature Between Projects

**Goal:** Move authentication logic from Project A to Project B

**Step 1: Find source implementation**
```bash
codefind query "JWT authentication validation" --project="ProjectA"
```

**Step 2: Find similar code in target**
```bash
codefind query "authentication middleware" --project="ProjectB"
```

**Step 3: Find dependencies**
```bash
# Find all uses of the auth module in source
codefind query "import auth validate_token" --project="ProjectA"

# Check what auth libraries target uses
codefind query "authentication library dependencies" --project="ProjectB"
```

**Step 4: Plan migration**
- Compare authentication approaches
- Note differences in dependencies
- Identify code that needs adaptation
- Plan for testing

### Scenario 2: Upgrade Library Version

**Goal:** Upgrade from requests 2.x to 3.x

**Step 1: Find all usages of old API**
```bash
codefind query "requests library HTTP client" --all
```

**Step 2: Check for deprecated patterns**
```bash
# Find patterns that changed in v3
codefind query "requests session retry timeout" --all
```

**Step 3: Find migration examples**
```bash
# Search for any code already using v3 patterns
codefind query "requests async await httpx" --all
```

**Step 4: Create migration checklist**
- List all files using requests
- Note deprecated API usage
- Plan replacement code
- Update tests

### Scenario 3: Refactor to New Pattern

**Goal:** Refactor callback-based code to use async/await

**Step 1: Find callback code**
```bash
codefind query "callback pattern asynchronous" --project="MyProject"
```

**Step 2: Find async examples**
```bash
codefind query "async await asyncio" --project="MyProject" --lang=python
```

**Step 3: Identify dependencies**
```bash
# Find all code calling the callback-based functions
codefind query "call callback register listener" --project="MyProject"
```

**Step 4: Plan refactoring**
- List functions to convert
- Note breaking changes
- Update callers
- Add async tests

### Scenario 4: Port Implementation to Another Language

**Goal:** Port Python validation logic to Go

**Step 1: Find source implementation**
```bash
codefind query "input validation sanitization" --project="PythonService" --lang=python
```

**Step 2: Find similar Go code**
```bash
codefind query "input validation struct tags" --project="GoService" --lang=go
```

**Step 3: Compare patterns**
- Python uses decorators and type hints
- Go uses struct tags and validator libraries
- Note idiomatic differences

**Step 4: Create Go implementation**
- Adapt logic to Go idioms
- Use appropriate Go libraries
- Match validation behavior
- Write equivalent tests

## Examples

### Example 1: Extract Shared Code to Library

**Scenario:** Multiple projects have duplicate error handling code

**Step 1: Find duplicate implementations**
```bash
codefind query "custom error wrapping context" --all
```

**Results:**
```
1. [API-Gateway] internal/errors/errors.go:15-45
2. [Analytics] pkg/errors/errors.go:20-50
3. [Worker] internal/errors/handler.go:10-40
```

**Step 2: Compare implementations**
- Read all three implementations
- Note similarities: all wrap errors with context
- Note differences: different error types, some have retry logic

**Step 3: Design shared library**
```
Shared Error Library Design:
- Core: Error wrapping with context
- Types: RetryableError, ValidationError, NotFoundError
- Features: Stack traces, structured fields
- From API-Gateway: Retry logic
- From Analytics: Error categorization
- From Worker: Error metrics
```

**Step 4: Create migration plan**
```
Migration Steps:
1. Create new shared library: pkg/errors
2. Implement core functionality + best features from each
3. Migrate API-Gateway (low risk, good tests)
4. Migrate Analytics (medium risk)
5. Migrate Worker (high risk, update carefully)
6. Remove old implementations
```

### Example 2: Migrate Database Layer

**Scenario:** Migrate from raw SQL to ORM

**Step 1: Find SQL query code**
```bash
codefind query "SQL query database execute" --project="MyApp"
```

**Step 2: Find existing ORM usage**
```bash
codefind query "ORM model query filter" --project="MyApp"
```

**Step 3: Analyze query patterns**
```
SQL Queries Found:
- 45 SELECT queries (mostly simple)
- 12 INSERT queries
- 8 UPDATE queries
- 5 complex JOINs
Existing ORM usage:
- SQLAlchemy already used in auth module
- Good test coverage
- Migration path exists
```

**Step 4: Migration strategy**
```
Phase 1: Simple queries (SELECT single table)
Phase 2: INSERT/UPDATE queries
Phase 3: Complex JOINs (may need raw SQL in ORM)
Phase 4: Optimize and test performance
```

### Example 3: Modernize API Endpoints

**Scenario:** Update REST API to use newer framework features

**Step 1: Find old endpoint patterns**
```bash
codefind query "route handler decorator Flask" --project="API"
```

**Step 2: Find new pattern examples**
```bash
codefind query "Blueprint MethodView Flask modern" --project="API"
```

**Step 3: Compare patterns**
```
Old Pattern:
@app.route('/users', methods=['GET', 'POST'])
def users():
    if request.method == 'GET':
        # handle GET
    elif request.method == 'POST':
        # handle POST

New Pattern:
class UsersView(MethodView):
    def get(self):
        # handle GET
    def post(self):
        # handle POST
```

**Step 4: Migration plan**
```
1. Create new MethodView classes
2. Migrate one endpoint as proof of concept
3. Test thoroughly
4. Migrate remaining endpoints in batches
5. Remove old route handlers
6. Update documentation
```

## Dependency Analysis

Before migrating, analyze dependencies:

### Find Direct Dependencies
```bash
# Find imports/includes
codefind query "import library_name" --project="Source"

# Find function calls
codefind query "library_name.function_call" --project="Source"
```

### Find Transitive Dependencies
```bash
# What does the code you're migrating depend on?
codefind query "imported modules dependencies" --project="Source"
```

### Check Target Compatibility
```bash
# Does target have compatible libraries?
codefind query "dependency library alternative" --project="Target"
```

## Migration Best Practices

1. **Start small:**
   - Migrate low-risk code first
   - Prove the migration pattern works
   - Learn from early migrations

2. **Find good examples:**
   - Search for code already using the target pattern
   - Learn from existing implementations
   - Follow established patterns

3. **Check dependencies:**
   - Find all code depending on what you're migrating
   - Update callers along with the code
   - Don't break existing functionality

4. **Test thoroughly:**
   - Find existing tests for the code
   - Port tests to new implementation
   - Add new tests for migration-specific concerns

5. **Document changes:**
   - Note what changed and why
   - Document new patterns and usage
   - Update architecture docs

6. **Plan rollback:**
   - Keep old code until new is proven
   - Have a rollback strategy
   - Monitor after migration

## Common Migration Patterns

### Library Upgrade
1. Find all usages of old API
2. Check deprecation warnings
3. Find migration guide examples
4. Update code incrementally
5. Test each change

### Extract to Shared Library
1. Find duplicate implementations
2. Compare and find common core
3. Design unified API
4. Migrate projects one by one
5. Remove duplicates

### Language Port
1. Find source implementation
2. Find idiomatic target examples
3. Translate logic (not literal)
4. Use appropriate libraries
5. Match behavior, not code

### Pattern Refactor
1. Find old pattern usage
2. Find or create new pattern
3. Create adapter if needed
4. Migrate incrementally
5. Remove old pattern

## Troubleshooting

**Can't find similar code:**
- Try broader search terms
- Search in other projects
- Look for related patterns

**Dependencies conflict:**
- Check version compatibility
- Find alternative libraries
- Consider gradual migration

**Breaking changes required:**
- Plan phased rollout
- Use feature flags
- Maintain backward compatibility temporarily

## Related Skills

- **codefind-search**: For finding code to migrate
- **codefind-patterns**: For understanding target patterns
- **codefind-summarize**: For documenting migration changes
