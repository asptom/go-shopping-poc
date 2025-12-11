# EventRegister Elimination Plan

## Executive Summary

This document outlines the complete elimination of `eventregister.go` and legacy handler support from the event system. The current system already uses typed handlers exclusively, with legacy support remaining only for backward compatibility. All active services (`eventreader`, `customer`) use the modern `SubscribeTyped` pattern.

## Current State Analysis

### Legacy System Components

#### 1. eventregister.go (`internal/platform/event/eventregister.go`)
- **Purpose**: Global registry for event unmarshal functions
- **Key Functions**: `Register()`, `UnmarshalEvent()`, `MustUnmarshalEvent()`
- **Current Usage**: Only used in `eventbus.go:157` for legacy handler support
- **Status**: **NO ACTIVE USAGE** in application code

#### 2. Legacy Handler Support in eventbus.go
- **Legacy Subscribe Method**: `Subscribe(eventType string, handler EventHandler)` (lines 78-83)
- **Legacy Handler Interface**: `EventHandler` with `Handle(ctx context.Context, event event.Event) error`
- **Legacy Processing Logic**: Lines 148-173 in `StartConsuming()`
- **Current Usage**: **NO ACTIVE USAGE** in services

#### 3. RawEvent Fallback Type
- **Purpose**: Implements `event.Event` for unknown event types
- **Current Usage**: Only as fallback in `UnmarshalEvent()`
- **Status**: Can be removed with eventregister.go

### Modern System Components (Fully Adopted)

#### 1. Typed Handler System
- **SubscribeTyped**: Generic, type-safe subscription method
- **HandlerFunc[T]**: Type-safe handler function
- **TypedHandler[T]**: Generic handler wrapper
- **EventFactory[T]**: Type-safe event reconstruction

#### 2. Active Service Usage
- **eventreader**: Uses `SubscribeTyped` with `CustomerEventFactory`
- **customer**: Uses eventbus only for publishing (outbox pattern)
- **websocket**: No event system usage

#### 3. Event Implementations
- **CustomerEvent**: Complete typed implementation with factory
- **CustomerEventFactory**: Implements `EventFactory[CustomerEvent]`

## Dependency Mapping

### Files Using Legacy System
1. **`internal/platform/eventbus/eventbus.go`**
   - Imports: `event "go-shopping-poc/internal/platform/event"`
   - Uses: `event.UnmarshalEvent()` (line 157)
   - Contains: Legacy `Subscribe()` method and processing logic

2. **`internal/platform/eventbus/handler_test.go`**
   - Contains: `TestBackwardCompatibility()` testing legacy handlers
   - Uses: Mock legacy handler for testing

### Files Using eventregister.go
1. **`internal/platform/eventbus/eventbus.go`** (indirect via `event.UnmarshalEvent`)
2. **`internal/platform/event/eventregister.go`** (self-contained)

### Files NOT Using Legacy System
1. **`cmd/eventreader/main.go`** - Uses `SubscribeTyped` exclusively
2. **`cmd/customer/main.go`** - Uses eventbus only for publishing
3. **`cmd/websocket/main.go`** - No event system usage
4. **`internal/platform/outbox/publisher.go`** - Uses `PublishRaw` only

## Migration Strategy

### Phase 1: Remove Legacy Handler Support from EventBus

#### 1.1 Remove Legacy Handler Infrastructure
**File**: `internal/platform/eventbus/eventbus.go`

**Remove**:
- `EventHandler` interface (lines 17-20)
- `handlers map[string][]EventHandler` field (line 28)
- `Subscribe()` method (lines 78-83)
- Legacy handler processing logic (lines 148-173)
- `event.UnmarshalEvent()` import and usage (line 157)

**Modify**:
- `NewEventBus()` constructor: Remove `handlers` initialization
- `StartConsuming()`: Keep only typed handler processing logic

#### 1.2 Update EventBus Structure
**Before**:
```go
type EventBus struct {
    writer   *kafka.Writer
    readers  map[string]*kafka.Reader
    handlers map[string][]EventHandler  // Remove
    typedHandlers map[string][]func(ctx context.Context, data []byte) error
    mu            sync.RWMutex
}
```

**After**:
```go
type EventBus struct {
    writer   *kafka.Writer
    readers  map[string]*kafka.Reader
    typedHandlers map[string][]func(ctx context.Context, data []byte) error
    mu            sync.RWMutex
}
```

### Phase 2: Eliminate eventregister.go

#### 2.1 Remove eventregister.go File
**File**: `internal/platform/event/eventregister.go`

**Action**: Complete file deletion

#### 2.2 Clean Up Event Package
**File**: `internal/platform/event/event.go`

**Status**: No changes needed - contains only modern interfaces

### Phase 3: Update Tests

#### 3.1 Remove Legacy Handler Tests
**File**: `internal/platform/eventbus/handler_test.go`

**Remove**:
- `TestBackwardCompatibility()` function (lines 123-141)
- `mockLegacyHandler` type (lines 143-148)
- Legacy handler import references

**Keep**: All typed handler tests (they're comprehensive and passing)

#### 3.2 Verify Test Coverage
- Ensure all typed handler functionality remains tested
- Confirm `SubscribeTyped` tests cover all scenarios
- Validate `EventFactory` tests remain intact

### Phase 4: Documentation Updates

#### 4.1 Update AGENTS.md
**Remove**:
- Any references to legacy handler patterns
- Backward compatibility mentions

**Ensure**:
- `SubscribeTyped` usage documentation is clear
- Event factory pattern is well documented

#### 4.2 Update Code Comments
**Remove**:
- "Maintains backward compatibility" comments
- Legacy system references

**Add**:
- Clear documentation for typed-only system

## Implementation Plan

### Step 1: Backup and Verify
1. Create git commit with current working state
2. Run full test suite to ensure baseline functionality
3. Verify all services start correctly

### Step 2: Remove Legacy EventBus Support
1. Modify `internal/platform/eventbus/eventbus.go`
2. Remove legacy handler fields and methods
3. Update `StartConsuming()` to use only typed handlers
4. Run tests to verify typed handlers still work

### Step 3: Remove eventregister.go
1. Delete `internal/platform/event/eventregister.go`
2. Remove import from eventbus.go
3. Verify no compilation errors

### Step 4: Update Tests
1. Remove legacy handler tests from `handler_test.go`
2. Run full test suite
3. Ensure all typed handler tests pass

### Step 5: Final Verification
1. Test `eventreader` service functionality
2. Test `customer` service event publishing
3. Verify outbox publisher works correctly
4. Run integration tests if available

### Step 6: Documentation Cleanup
1. Update AGENTS.md
2. Update code comments
3. Remove any legacy system documentation

## Risk Assessment

### Low Risk Items
- **Removing eventregister.go**: No active usage found
- **Removing legacy handler tests**: Only testing backward compatibility
- **Cleaning up comments**: Documentation changes only

### Medium Risk Items
- **Modifying EventBus structure**: Core component changes
- **Updating StartConsuming logic**: Event processing flow changes

### Mitigation Strategies
1. **Incremental Changes**: Make changes step-by-step with testing at each stage
2. **Comprehensive Testing**: Run full test suite after each change
3. **Service Verification**: Test actual service functionality after changes
4. **Rollback Plan**: Keep git commits granular for easy rollback

## Success Criteria

### Functional Requirements
- ✅ All typed handlers continue to work
- ✅ `eventreader` service processes events correctly
- ✅ `customer` service publishes events correctly
- ✅ Outbox publisher functions properly
- ✅ All tests pass

### Code Quality Requirements
- ✅ No legacy code remains
- ✅ No unused imports
- ✅ Clean, readable code structure
- ✅ Updated documentation

### Architectural Requirements
- ✅ Simplified event system
- ✅ Type-safe event handling only
- ✅ No global state (eventregister)
- ✅ Clear separation of concerns

## Cleanup Checklist

### Files to Modify
- [ ] `internal/platform/eventbus/eventbus.go` - Remove legacy support
- [ ] `internal/platform/eventbus/handler_test.go` - Remove legacy tests
- [ ] `AGENTS.md` - Update documentation

### Files to Delete
- [ ] `internal/platform/event/eventregister.go` - Complete removal

### Verification Steps
- [ ] All tests pass (`make services-test`)
- [ ] `eventreader` service starts and processes events
- [ ] `customer` service starts and publishes events
- [ ] No compilation errors
- [ ] No unused import warnings
- [ ] Code is properly formatted

## Post-Elimination Architecture

### Simplified Event System
```
EventBus
├── NewEventBus() - Constructor
├── SubscribeTyped[T]() - Type-safe subscription
├── Publish() - Event publishing
├── PublishRaw() - Raw JSON publishing (for outbox)
└── StartConsuming() - Event consumption (typed only)

TypedHandler[T]
├── NewTypedHandler() - Constructor
└── Handle() - Type-safe event processing

Event Interfaces
├── Event - Base event interface
└── EventFactory[T] - Type-safe event reconstruction
```

### Benefits Achieved
1. **Simplified Architecture**: No legacy code paths
2. **Type Safety**: Compile-time type checking for all event handling
3. **No Global State**: Eliminated eventregister global registry
4. **Cleaner Code**: Reduced complexity and improved maintainability
5. **Better Testing**: Focused tests on actual usage patterns

## Timeline Estimate

- **Phase 1**: 2-3 hours (EventBus modifications)
- **Phase 2**: 30 minutes (eventregister.go removal)
- **Phase 3**: 1-2 hours (Test updates and verification)
- **Phase 4**: 1 hour (Documentation updates)
- **Total**: 4.5-6.5 hours

## Conclusion

The elimination of `eventregister.go` and legacy handler support is a **low-risk, high-benefit** cleanup. The current system already operates exclusively with typed handlers, making this a removal of unused code rather than a migration. The result will be a cleaner, more maintainable event system with no functional changes to existing services.