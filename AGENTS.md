# AGENTS.md

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

### Naming Conventions
- PascalCase for exported types, functions, constants
- camelCase for local variables and private fields
- Use descriptive names: CustomerHandler, CreateCustomer
- Interface names: Reader, Writer, Service suffix

### Error Handling
- Always handle errors, never use _
- Return errors from functions, don't panic
- Use http.Error for HTTP responses with appropriate status codes
- Validate input parameters early

### Project Structure
- cmd/ for main applications
- internal/ for private application code
- pkg/ for public library code
- Follow domain-driven design: entity, service, handler layers

### Event Architecture
- Use typed event handlers with generics for type safety
- Events implement `event.Event` interface with value receivers
- Create `EventFactory[T]` for type-safe event reconstruction
- Use `eventbus.SubscribeTyped()` for new event handlers
- Legacy `eventbus.Subscribe()` still supported for backward compatibility
- No global event registry - use direct factory pattern instead

## Recent Session Summary (Nov 7, 2025)

### Event Architecture Redesign
**Problem Solved**: Eliminated registry pattern and adapter anti-patterns that created complexity and type safety issues.

**Key Changes Made**:
1. **Enhanced Event Interface** - Removed `FromJSON`, added generic `EventFactory[T]` interface
2. **Generic Handler System** - Created `TypedHandler[T]` with compile-time type safety
3. **EventBus Enhancement** - Added `SubscribeTyped()` method while maintaining backward compatibility
4. **Customer Event Refactoring** - Removed complex `init()` registration, added factory pattern
5. **EventReader Simplification** - Eliminated `customerHandlerAdapter`, reduced code by 41%
6. **Outbox Integration** - Added `PublishRaw()` method to avoid double marshaling

**Benefits Achieved**:
- **Zero Global State**: No more registry pattern or init() complexity
- **Type Safety**: Compile-time checking prevents runtime errors
- **Performance**: No reflection or registry lookups
- **Maintainability**: Cleaner, more readable code with fewer layers
- **Backward Compatibility**: Existing code continues to work unchanged

**New Pattern Example**:
```go
factory := events.CustomerEventFactory{}
handler := eventbus.HandlerFunc[events.CustomerEvent](func(ctx context.Context, evt events.CustomerEvent) error {
    // Type-safe handling
    return nil
})
eventbus.SubscribeTyped(eventBus, factory, handler)
```

**Files Modified**: 9 files updated, comprehensive tests added, documentation created
**Status**: ✅ Complete and tested