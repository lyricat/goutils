# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

goutils is a comprehensive Go utility library providing packages for AI integrations, data storage, web services, and social media integrations. The codebase follows a modular package structure with each package providing focused functionality.

## Common Development Commands

### Build and Test
```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./ai/...
go test ./httphelper/middleware/...

# Run tests with coverage
go test -cover ./...

# Build the main example
go build -o main main/main.go

# Generate database access code
go run gen/gen.go
```

### Code Quality
```bash
# Format code
go fmt ./...

# Run go vet for static analysis
go vet ./...

# Check for module issues
go mod tidy
go mod verify
```

## High-Level Architecture

### Package Organization and Dependencies

The repository uses a layered architecture where higher-level packages depend on lower-level utilities:

1. **AI Layer** (`/ai/`): Provides unified interfaces for multiple AI providers. The main `ai.go` file contains the factory pattern implementation that switches between providers based on configuration. When adding new AI providers, follow the existing pattern of implementing the provider-specific client and adding it to the switch statement in `GetClient()`.

2. **Storage Layer**: 
   - `/qdrant/`: Vector database operations using GRPC. The client maintains connection pooling and supports batch operations.
   - `/storage/`: S3 operations for file storage, integrated with the model layer for attachments.

3. **Web Services Layer** (`/httphelper/`):
   - Middleware components are designed to be composable using Chi router.
   - The Ban middleware maintains an in-memory blocklist and checks requests against malicious path patterns defined in `maliciousPaths` slice.
   - Rate limiting uses Redis for distributed rate limiting across multiple instances.

4. **Data Models** (`/model/`):
   - Uses GORM for database operations with generated repository pattern.
   - The `Attachment` model integrates with S3 storage for file handling.
   - JSON fields use custom `JSONMap` type from `/structs/` for type-safe JSON operations.

### Key Design Patterns

**Factory Pattern in AI Package**: The AI package uses configuration-driven instantiation. When working with AI providers, always use `GetClient()` with proper configuration rather than instantiating providers directly.

**Middleware Composition**: HTTP middleware follows the standard `http.Handler` pattern and can be chained. New middleware should follow the pattern in `/httphelper/middleware/ban.go`.

**Repository Pattern for Database Access**: Database operations go through generated repositories. After modifying models, regenerate the access layer using `go run gen/gen.go`.

### Configuration Management

The codebase uses configuration structs for each major component:
- AI providers: `AIConfig` with provider-specific settings
- Qdrant: `QdrantConfig` with connection details
- Social media: Provider-specific config structs (e.g., `TwitterConfig`)

Environment variables are typically loaded into these structs at initialization.

### Testing Approach

Tests follow Go conventions with `*_test.go` files alongside implementation files. Key testing patterns:
- Unit tests for utility functions (see `/ai/util_test.go`)
- Integration tests for middleware with mock HTTP servers (see `/httphelper/middleware/ban_test.go`)
- Table-driven tests for multiple scenarios

### Security Considerations

When working with this codebase:
- The Ban middleware automatically blocks IPs attempting to access sensitive paths
- OAuth tokens and API keys should be loaded from environment variables
- The `/structs/` package provides safe JSON handling to prevent injection attacks
- Redis is used for session storage to avoid storing sensitive data in cookies

### Common Development Tasks

**Adding a New AI Provider**:
1. Create provider file in `/ai/` (e.g., `provider_name.go`)
2. Implement the provider client with required methods
3. Add provider to the switch statement in `ai.go:GetClient()`
4. Update `AIConfig` struct if new configuration fields are needed

**Adding New Middleware**:
1. Create new file in `/httphelper/middleware/`
2. Follow the pattern of existing middleware (e.g., `ban.go`)
3. Implement the `http.Handler` interface
4. Add appropriate tests

**Working with Vector Database**:
- Qdrant operations are in `/qdrant/`
- Use `NewQdrantClient()` for initialization
- Points are inserted with embeddings from the AI package

**Database Model Changes**:
1. Modify models in `/model/`
2. Run `go run gen/gen.go` to regenerate access code
3. Run migrations if database schema changes