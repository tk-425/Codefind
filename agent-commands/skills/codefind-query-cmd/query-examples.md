# Query Examples

Common query patterns and examples for different code search scenarios.

## Find Functions

### General Functions

```bash
codefind query "function that validates email addresses"
codefind query "function to connect to database"
codefind query "helper function for date formatting"
codefind query "utility function for string manipulation"
```

### Specific Function Types

```bash
# Validation functions
codefind query "input validation function"
codefind query "email validation with regex"
codefind query "password strength validator"

# Database functions
codefind query "database connection pool function"
codefind query "query builder function"
codefind query "transaction wrapper function"

# Utility functions
codefind query "JSON serialization function"
codefind query "date formatting utility"
codefind query "string sanitization function"
```

## Find Classes

### Domain Models

```bash
codefind query "user model class"
codefind query "product model with inventory"
codefind query "order class with validation"
```

### Service Classes

```bash
codefind query "HTTP client class"
codefind query "email service class"
codefind query "payment processing class"
```

### Infrastructure Classes

```bash
codefind query "configuration manager class"
codefind query "logger class implementation"
codefind query "cache manager class"
```

## Find Patterns

### Error Handling

```bash
codefind query "error handling with retry logic"
codefind query "custom error class definition"
codefind query "error recovery pattern"
codefind query "graceful error handling"
```

### Middleware

```bash
codefind query "middleware for request logging"
codefind query "authentication middleware"
codefind query "rate limiting middleware"
codefind query "CORS middleware setup"
```

### Decorators

```bash
codefind query "decorator for caching"
codefind query "authentication decorator"
codefind query "logging decorator pattern"
codefind query "rate limit decorator"
```

### Design Patterns

```bash
codefind query "singleton pattern implementation"
codefind query "factory pattern for objects"
codefind query "observer pattern example"
codefind query "dependency injection pattern"
```

## Find API Endpoints

### REST Endpoints

```bash
codefind query "POST endpoint for user creation"
codefind query "GET endpoint to list items"
codefind query "PUT endpoint for updating records"
codefind query "DELETE endpoint for removing data"
```

### Specific API Routes

```bash
codefind query "API route for authentication"
codefind query "endpoint for file upload"
codefind query "webhook endpoint handler"
codefind query "health check endpoint"
```

### GraphQL

```bash
codefind query "GraphQL resolver function"
codefind query "GraphQL mutation for user"
codefind query "GraphQL query type definition"
```

## Find Configuration

### Environment Configuration

```bash
codefind query "environment variable configuration"
codefind query "config file parser"
codefind query "settings loader from environment"
```

### Database Configuration

```bash
codefind query "database connection settings"
codefind query "database pool configuration"
codefind query "migration configuration"
```

### Feature Flags

```bash
codefind query "feature flags configuration"
codefind query "feature toggle implementation"
codefind query "A/B test configuration"
```

## Find Tests

### Unit Tests

```bash
codefind query "unit test for validation function"
codefind query "test case for edge conditions"
codefind query "mock setup for testing"
```

### Integration Tests

```bash
codefind query "integration test for API endpoint"
codefind query "database integration test"
codefind query "end-to-end test example"
```

### Test Utilities

```bash
codefind query "test fixture setup"
codefind query "test helper functions"
codefind query "test data factory"
```

## Find Security Implementations

### Authentication

```bash
codefind query "JWT token generation"
codefind query "password hashing with bcrypt"
codefind query "OAuth2 authentication flow"
codefind query "session management code"
```

### Authorization

```bash
codefind query "role-based access control"
codefind query "permission checking logic"
codefind query "authorization middleware"
```

### Input Validation

```bash
codefind query "SQL injection prevention"
codefind query "XSS sanitization function"
codefind query "input validation schema"
```

## Find Data Handling

### Database Operations

```bash
codefind query "database query with joins"
codefind query "bulk insert operation"
codefind query "transaction handling code"
codefind query "database migration script"
```

### Caching

```bash
codefind query "Redis caching implementation"
codefind query "in-memory cache setup"
codefind query "cache invalidation logic"
```

### Data Transformation

```bash
codefind query "JSON to object mapper"
codefind query "data serialization function"
codefind query "CSV parser implementation"
```

## Find Async/Concurrent Code

### Async Operations

```bash
codefind query "async await function example"
codefind query "promise chain implementation"
codefind query "parallel async operations"
```

### Concurrency

```bash
codefind query "goroutine worker pool"
codefind query "thread pool implementation"
codefind query "concurrent queue processing"
```

### Background Jobs

```bash
codefind query "background job scheduler"
codefind query "task queue worker"
codefind query "cron job implementation"
```

## Find Logging and Monitoring

### Logging

```bash
codefind query "structured logging setup"
codefind query "error logging with context"
codefind query "log level configuration"
```

### Monitoring

```bash
codefind query "metrics collection code"
codefind query "performance monitoring"
codefind query "health check implementation"
```

### Tracing

```bash
codefind query "distributed tracing setup"
codefind query "request ID propagation"
codefind query "span creation for tracing"
```

## Language-Specific Examples

### Python

```bash
codefind query "Python class with decorators" --lang=python
codefind query "asyncio event loop setup" --lang=python
codefind query "context manager implementation" --lang=python
```

### Go

```bash
codefind query "Go interface implementation" --lang=go
codefind query "error handling with errors package" --lang=go
codefind query "goroutine with channel communication" --lang=go
```

### TypeScript

```bash
codefind query "TypeScript interface definition" --lang=typescript
codefind query "React component with hooks" --lang=typescript
codefind query "generic type implementation" --lang=typescript
```

### JavaScript

```bash
codefind query "Express middleware function" --lang=javascript
codefind query "promise-based HTTP client" --lang=javascript
codefind query "React component lifecycle" --lang=javascript
```

## Framework-Specific Examples

### Django

```bash
codefind query "Django model with relationships"
codefind query "Django view function with form"
codefind query "Django middleware class"
```

### Flask

```bash
codefind query "Flask route with authentication"
codefind query "Flask blueprint configuration"
codefind query "Flask error handler"
```

### Express

```bash
codefind query "Express router setup"
codefind query "Express error handling middleware"
codefind query "Express request validation"
```

### React

```bash
codefind query "React custom hook"
codefind query "React context provider"
codefind query "React form with validation"
```

## Multi-Project Examples

### Cross-Project Patterns

```bash
# Find all authentication implementations
codefind query "user authentication logic" --all

# Compare error handling across services
codefind query "error handling pattern" --projects="API,Gateway,Auth"

# Find all database connection code
codefind query "database connection setup" --all
```

### Consistency Checks

```bash
# Find all logging implementations
codefind query "logging configuration" --all

# Find all API client implementations
codefind query "HTTP client setup" --all

# Find all validation patterns
codefind query "input validation" --all
```

## Query Refinement Examples

### From Broad to Specific

```bash
# Too broad
codefind query "validation"

# Better - add context
codefind query "email validation"

# Best - very specific
codefind query "email validation with regex and domain check"
```

### Adding Domain Terms

```bash
# Generic
codefind query "check password"

# Domain-specific
codefind query "bcrypt password hashing validation"

# Very specific
codefind query "bcrypt password verification with salt rounds"
```

### Adding Implementation Details

```bash
# High-level
codefind query "authentication"

# Implementation detail
codefind query "JWT authentication with refresh tokens"

# Very detailed
codefind query "JWT authentication with refresh tokens and Redis session storage"
```
