# Pagination Guide

Detailed guide for navigating large result sets using pagination.

## Basic Pagination

```bash
# First page (default)
codefind query "database"

# Second page
codefind query "database" --page=2

# Third page
codefind query "database" --page=3
```

**Default behavior:**
- Page size: 20 results per page
- Page number: 1 (first page)
- Results are consistently ordered by similarity score

## Custom Page Size

```bash
# Show 10 results per page
codefind query "functions" --page-size=10

# Show 50 results per page (maximum)
codefind query "functions" --page-size=50

# Navigate to second page with 30 results per page
codefind query "functions" --page=2 --page-size=30
```

**Page size limits:**
- Minimum: 1 result
- Maximum: 50 results
- Default: 20 results

**When to adjust page size:**
- Small page size: Quick review of top results
- Large page size: Comprehensive review of all matches

## Pagination Examples

### Scenario: Browse all authentication implementations

```bash
# Start with page 1
codefind query "authentication middleware" --all

# Results show: "Showing results 1-20 of 47"
# View next page
codefind query "authentication middleware" --all --page=2

# Results show: "Showing results 21-40 of 47"
# View final page
codefind query "authentication middleware" --all --page=3

# Results show: "Showing results 41-47 of 47"
```

### Scenario: Review top matches with small pages

```bash
# See top 5 results
codefind query "JWT validation" --page-size=5

# If these are good, see next 5
codefind query "JWT validation" --page-size=5 --page=2
```

### Scenario: Comprehensive review with large pages

```bash
# Get first 50 results (maximum)
codefind query "error handling" --page-size=50

# Get next batch
codefind query "error handling" --page-size=50 --page=2
```

## Understanding Pagination Output

**Result header:**
```
Results for: "authentication middleware"
Showing results 1-20 of 47
```

**Interpretation:**
- Query: "authentication middleware"
- Current range: Results 1 through 20
- Total matches: 47 results
- Pages available: 3 pages (47 ÷ 20 = 2.35, rounded up)

## Pagination Strategy

### Quick Review (Small Pages)

**Use when:**
- You expect top results to be sufficient
- Want to minimize noise
- Need fast review of best matches

**Strategy:**
```bash
# Start with 5-10 results
codefind query "specific feature" --page-size=5

# Review scores
# If good matches found, stop
# If need more, increment page
codefind query "specific feature" --page-size=5 --page=2
```

### Comprehensive Review (Large Pages)

**Use when:**
- Exploring unfamiliar codebase
- Need to see all implementations
- Comparing multiple approaches

**Strategy:**
```bash
# Get maximum results per page
codefind query "error handling" --page-size=50

# Review all
# Continue to next page if needed
codefind query "error handling" --page-size=50 --page=2
```

### Targeted Review (Medium Pages)

**Use when:**
- Standard search workflow
- Balance between speed and coverage

**Strategy:**
```bash
# Default 20 results
codefind query "database connection"

# Or customize to 30
codefind query "database connection" --page-size=30
```

## Pagination with Filters

Combine pagination with filters for targeted browsing:

```bash
# Python code, first page
codefind query "async" --lang=python --page-size=20

# Python code, second page
codefind query "async" --lang=python --page-size=20 --page=2

# Specific project, third page
codefind query "handlers" --project="API" --page=3
```

## Pagination Best Practices

1. **Start with default page size (20)**
   - Good balance for most searches
   - Adjust based on results

2. **Use smaller pages for specific queries**
   - When you expect few good matches
   - When top results are usually sufficient

3. **Use larger pages for broad queries**
   - When exploring patterns
   - When comparing many implementations

4. **Keep page size consistent**
   - Don't change page size between pages
   - Helps maintain consistent mental model

5. **Note total result count**
   - Understand scope before deep diving
   - Plan how many pages to review

## Pagination Shortcuts

```bash
# Quick: Just top results
codefind query "function" --page-size=5

# Standard: Default behavior
codefind query "function"

# Thorough: Maximum results
codefind query "function" --page-size=50

# Navigate: Specific page
codefind query "function" --page=3

# Custom: Specific size and page
codefind query "function" --page=2 --page-size=30
```

## Common Pagination Patterns

### Pattern 1: Top Result Scan

```bash
# Quick scan of top 10
codefind query "validation logic" --page-size=10

# If found, open result
codefind open 1

# If not found, refine query
```

### Pattern 2: Comprehensive Browse

```bash
# Page through all results
codefind query "error handling" --all

# Results: "Showing 1-20 of 85"
for page in {1..5}; do
  codefind query "error handling" --all --page=$page
done
```

### Pattern 3: Filter and Page

```bash
# Start broad
codefind query "authentication"

# Results: Too many, add filter
codefind query "authentication" --lang=go

# Still many, use pagination
codefind query "authentication" --lang=go --page-size=15
```

## Pagination Limitations

**Maximum page size:** 50 results
- Server limitation
- Prevents overwhelming output
- Use multiple pages for more results

**Consistent ordering:**
- Same query always returns same order
- Based on similarity scores
- Results won't change between pages

**No infinite scrolling:**
- Must explicitly request next page
- Can't auto-load more results
- Intentional design for clarity
