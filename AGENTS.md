# AGENTS.md

## Recent Session Summary (Today)

### pkg to internal Migration Complete ✅

**Problem Solved**: Successfully migrated all packages from `pkg/` to `internal/platform/` to align with Go project best practices and improve code organization.

**Migration Accomplished**:
- **9 packages migrated**: config, cors, errors, event, eventbus, logging, outbox, websocket, testutils
- **New structure**: All shared infrastructure now organized under `internal/platform/`
- **Updated imports**: All import statements updated to use new paths
- **Zero breaking changes**: All functionality preserved with improved organization

**New Project Structure**:
```
internal/
├── entity/customer/          # Domain entities
├── event/customer/           # Event handling  
├── platform/                # Shared infrastructure
│   ├── config/              # Configuration management
│   ├── cors/                # CORS handling
│   ├── errors/              # Error handling utilities
│   ├── event/               # Event interfaces
│   ├── eventbus/            # Event bus implementation
│   ├── logging/             # Logging utilities
│   ├── outbox/              # Outbox pattern
│   └── websocket/           # WebSocket server
├── service/customer/         # Business logic
└── testutils/               # Testing utilities

cmd/                         # Application entrypoints
deployments/                 # Kubernetes configs
docs/                        # Documentation
resources/                   # Setup resources
```

**Benefits Achieved**:
- **Better Organization**: Clear separation between domain logic and shared infrastructure
- **Go Best Practices**: Follows standard Go project layout with `internal/` for private code
- **Maintainability**: Easier to locate and manage shared components
- **Import Clarity**: More descriptive import paths (`internal/platform/config` vs `pkg/config`)

**Files Updated**: All import statements across the codebase updated to reflect new structure
**Status**: ✅ Complete and tested

## Recent Session Summary (Today)

### Environment Variable Expansion & Config Testing - Complete ✅

**Problem Solved**: Two critical issues were resolved:
1. **Environment variables in .env files weren't being expanded** (e.g., `$PSQL_CUSTOMER_ROLE` remained as literal text)
2. **Config package had zero test coverage** despite being critical infrastructure

**Solution Implemented**:

#### **1. Environment Variable Expansion Fix**
- **Modified `getEnv()` and `getEnvArray()` functions** in `internal/platform/config/config.go`:
  - Added `os.ExpandEnv()` calls to expand shell variables in loaded values
  - Example: `"postgres://$PSQL_CUSTOMER_ROLE:$PSQL_CUSTOMER_PASSWORD@localhost:30432/$PSQL_CUSTOMER_DB?sslmode=disable"`
    → `"postgres://customersuser:customerssecret@localhost:30432/customersdb?sslmode=disable"`

#### **2. Absolute Path Resolution**
- **Enhanced `ResolveEnvFile()`** in `internal/platform/config/env.go`:
  - Added `findProjectRoot()` function to locate project root by finding `go.mod`
  - Modified to return absolute paths instead of relative paths
  - Ensures `.env.local` is found regardless of test execution directory

#### **3. Comprehensive Config Testing**
- **Created `internal/platform/config/config_test.go`** (565 lines): 25 test functions covering all config functionality
- **Created `internal/platform/config/env_test.go`** (75 lines): Environment file resolution tests
- **Updated `internal/testutils/testutils.go`**: Enhanced debugging and logging

**Results Achieved**:
- ✅ **Environment variables properly expanded** in database URLs and all config values
- ✅ **Tests work from any directory** (project root detection)
- ✅ **25/25 config tests passing** with 77.8% coverage
- ✅ **All customer service tests passing** with proper database connections

## Build Commands
- `make services-build` - Build all services (customer, eventreader)
- `make services-test` - Run tests for all services  
- `make services-lint` - Run golangci-lint on all services
- `go test ./cmd/customer/...` - Run tests for single service
- `go test -run TestName ./path/to/package` - Run single test
- `go run ./cmd/customer` - Run customer service locally

## Code Style Guidelines

### Imports
- Group imports: stdlib, third-party, project packages
- Use absolute imports with module prefix "go-shopping-poc/"

### Formatting & Types
- Use `gofmt` for formatting
- Use explicit types for function parameters and returns
- Prefer uuid.UUID for IDs, convert to string for JSON
- Use struct tags: `json:"field_name" db:"field_name"`
- For nullable UUID fields in database, use `*uuid.UUID` in Go structs

### Naming Conventions
- PascalCase for exported types, functions, constants
- camelCase for local variables and private fields
- Use descriptive names: CustomerHandler, CreateCustomer
- Interface names: Reader, Writer, Service suffix

### Error Handling
- Always handle errors, never use _
- Return errors from functions, don't panic
- Use structured error responses from `internal/platform/errors` package
- Validate input parameters early

### Project Structure
- cmd/ for main applications
- internal/ for private application code
- Follow domain-driven design: entity, service, handler layers
- internal/platform/ for shared infrastructure components

### Database Schema Alignment
- Entity structs must match PostgreSQL schema exactly
- Use `*uuid.UUID` for nullable UUID fields (database: `uuid NULL`)
- Use `uuid.UUID` for required UUID fields (database: `uuid not null`)
- Document foreign key cascade behaviors in repository interfaces
- Test both nullable and non-nullable field scenarios

### Event Architecture
- Use typed event handlers with generics for type safety
- Events implement `event.Event` interface from `internal/platform/event`
- Create `EventFactory[T]` for type-safe event reconstruction
- Use `eventbus.SubscribeTyped()` from `internal/platform/eventbus`
- No global event registry - use direct factory pattern instead

### Testing Guidelines
- Use shared utilities from `internal/testutils` for test setup
- Follow test naming conventions: `TestFunctionName_Condition_ExpectedResult`
- Use `SetupTestDB()` for database tests with proper cleanup
- Write comprehensive tests for all public methods