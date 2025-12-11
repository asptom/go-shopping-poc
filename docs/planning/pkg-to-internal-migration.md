# pkg/ to internal/platform/ Migration Plan

## Executive Summary

This document outlines a comprehensive migration plan to move packages from `pkg/` to `internal/platform/` to align with Go project layout standards. The migration will be executed in incremental phases to ensure system stability and minimize breaking changes.

## Current State Analysis

### Current pkg/ Structure
```
pkg/
├── config/          # Configuration loading and management
├── cors/            # CORS middleware
├── errors/          # Centralized error handling
├── event/           # Event interfaces and types
├── eventbus/        # Event bus implementation
├── logging/         # Logging utilities
├── outbox/          # Outbox pattern implementation
├── testutils/       # Testing utilities
└── websocket/       # WebSocket server
```

### Target Structure
```
internal/
└── platform/
    ├── config/
    ├── cors/
    ├── errors/
    ├── event/
    ├── eventbus/
    ├── logging/
    ├── outbox/
    └── websocket/

pkg/ (post-migration)
└── testutils/       # May remain or move to internal/testutils/
```

## Package Analysis

### 1. config/
**Purpose**: Configuration loading and management with environment variable expansion
**Dependencies**: 
- Internal: `go-shopping-poc/pkg/logging`
- External: `github.com/joho/godotenv`
**Migration Suitability**: ✅ HIGH - Pure infrastructure, no external consumers
**Risk Level**: LOW

### 2. cors/
**Purpose**: CORS middleware for HTTP services
**Dependencies**: 
- Internal: `go-shopping-poc/pkg/config`
- External: None (standard library only)
**Migration Suitability**: ✅ HIGH - Infrastructure middleware
**Risk Level**: LOW

### 3. errors/
**Purpose**: Centralized error handling utilities
**Dependencies**: 
- Internal: None
- External: None (standard library only)
**Migration Suitability**: ✅ HIGH - Pure infrastructure utility
**Risk Level**: LOW

### 4. event/
**Purpose**: Event interfaces and type definitions
**Dependencies**: 
- Internal: None
- External: None (standard library only)
**Migration Suitability**: ✅ HIGH - Core event infrastructure
**Risk Level**: LOW

### 5. eventbus/
**Purpose**: Kafka-based event bus implementation
**Dependencies**: 
- Internal: `go-shopping-poc/pkg/event`, `go-shopping-poc/pkg/logging`
- External: `github.com/segmentio/kafka-go`
**Migration Suitability**: ✅ HIGH - Infrastructure component
**Risk Level**: MEDIUM (due to Kafka complexity)

### 6. logging/
**Purpose**: Application logging utilities
**Dependencies**: 
- Internal: None
- External: None (standard library only)
**Migration Suitability**: ✅ HIGH - Core infrastructure
**Risk Level**: LOW

### 7. outbox/
**Purpose**: Outbox pattern implementation for reliable event publishing
**Dependencies**: 
- Internal: `go-shopping-poc/pkg/eventbus`, `go-shopping-poc/pkg/logging`
- External: None (standard library only)
**Migration Suitability**: ✅ HIGH - Infrastructure pattern
**Risk Level**: MEDIUM (data consistency critical)

### 8. testutils/
**Purpose**: Shared testing utilities and helpers
**Dependencies**: 
- Internal: `go-shopping-poc/pkg/config`
- External: `github.com/google/uuid`, `github.com/jmoiron/sqlx`
**Migration Suitability**: ⚠️ DEBATE - Could stay in pkg/ for external test usage
**Risk Level**: LOW

### 9. websocket/
**Purpose**: WebSocket client and server implementation
**Dependencies**: 
- Internal: `go-shopping-poc/pkg/config`, `go-shopping-poc/pkg/logging`
- External: `github.com/gorilla/websocket`
**Migration Suitability**: ✅ HIGH - Infrastructure component
**Risk Level**: MEDIUM (real-time communications)

## Dependency Mapping

### Services Using pkg/ Packages

#### cmd/customer/main.go
- `go-shopping-poc/pkg/config`
- `go-shopping-poc/pkg/cors`
- `go-shopping-poc/pkg/eventbus`
- `go-shopping-poc/pkg/logging`
- `go-shopping-poc/pkg/outbox`

#### cmd/eventreader/main.go
- `go-shopping-poc/pkg/config`
- `go-shopping-poc/pkg/eventbus`
- `go-shopping-poc/pkg/logging`

#### cmd/websocket/main.go
- `go-shopping-poc/pkg/config`
- `go-shopping-poc/pkg/logging`
- `go-shopping-poc/pkg/websocket`

#### internal/service/customer/
- `go-shopping-poc/pkg/errors` (handler.go)
- `go-shopping-poc/pkg/logging` (repository.go)
- `go-shopping-poc/pkg/outbox` (repository.go, tests)
- `go-shopping-poc/pkg/testutils` (tests)

#### internal/event/customer/
- `go-shopping-poc/pkg/logging` (customereventhandler.go)

### Internal pkg/ Dependencies
```
config → logging
cors → config
eventbus → event, logging
outbox → eventbus, logging
websocket → config, logging
testutils → config
```

## Migration Strategy

### Overall Approach
1. **Bottom-Up Migration**: Start with leaf packages (no internal dependencies)
2. **Incremental Updates**: Update imports one service at a time
3. **Compatibility Layer**: Temporary alias packages during transition
4. **Comprehensive Testing**: Verify functionality after each phase

### Key Principles
- **No Breaking Changes**: System must remain functional throughout migration
- **Incremental Deployment**: Each phase must be independently deployable
- **Rollback Capability**: Ability to revert changes if issues arise
- **Test Coverage**: All packages must maintain test coverage

## Phased Implementation Plan

### Phase 1: Foundation Packages (LOW RISK)
**Packages**: `logging`, `errors`, `event`

**Rationale**: These are leaf packages with no internal dependencies and minimal risk.

**Steps**:
1. Create `internal/platform/logging/` directory
2. Copy `pkg/logging/*` to `internal/platform/logging/`
3. Update package imports in dependent packages
4. Update `pkg/config/config.go` import
5. Create compatibility alias: `pkg/logging/logging.go` → `internal/platform/logging`
6. Run tests to verify functionality
7. Repeat for `errors` and `event`

**Expected Duration**: 1-2 hours

**Rollback Plan**: Remove `internal/platform/` directories, restore original `pkg/` packages.

### Phase 2: Configuration Layer (LOW-MEDIUM RISK)
**Packages**: `config`

**Rationale**: Depends only on `logging` (already migrated), widely used but low complexity.

**Steps**:
1. Create `internal/platform/config/` directory
2. Copy `pkg/config/*` to `internal/platform/config/`
3. Update import statement to use new logging package
4. Update all service imports:
   - `cmd/customer/main.go`
   - `cmd/eventreader/main.go`
   - `cmd/websocket/main.go`
   - `pkg/testutils/testutils.go`
   - `pkg/cors/cors.go`
   - `pkg/websocket/websocket.go`
5. Create compatibility alias in `pkg/config/`
6. Run comprehensive tests
7. Verify all services start correctly

**Expected Duration**: 2-3 hours

**Rollback Plan**: Restore original `pkg/config/`, update all imports back.

### Phase 3: Middleware Packages (MEDIUM RISK)
**Packages**: `cors`, `websocket`

**Rationale**: Depend on `config` (already migrated), moderate complexity.

**Steps**:
1. Migrate `cors` package:
   - Copy to `internal/platform/cors/`
   - Update config import
   - Update `cmd/customer/main.go` import
   - Create compatibility alias
   - Test CORS functionality

2. Migrate `websocket` package:
   - Copy to `internal/platform/websocket/`
   - Update config and logging imports
   - Update `cmd/websocket/main.go` import
   - Create compatibility alias
   - Test WebSocket connectivity

**Expected Duration**: 2-3 hours

**Rollback Plan**: Restore original packages, update service imports.

### Phase 4: Event Infrastructure (HIGH RISK)
**Packages**: `eventbus`, `outbox`

**Rationale**: Complex event handling, critical for system functionality, multiple dependencies.

**Steps**:
1. Migrate `eventbus`:
   - Copy to `internal/platform/eventbus/`
   - Update event and logging imports
   - Update all service imports
   - Create compatibility alias
   - Test event publishing/consuming

2. Migrate `outbox`:
   - Copy to `internal/platform/outbox/`
   - Update eventbus and logging imports
   - Update repository imports
   - Create compatibility alias
   - Test outbox pattern functionality

**Expected Duration**: 4-5 hours

**Rollback Plan**: Restore original packages, verify event system integrity.

### Phase 5: Test Utilities (SPECIAL CASE)
**Package**: `testutils`

**Decision Point**: Evaluate whether to keep in `pkg/` or move to `internal/testutils/`

**Options**:
- **Option A**: Keep in `pkg/` for potential external test usage
- **Option B**: Move to `internal/testutils/` for consistency

**Recommendation**: Move to `internal/testutils/` as it's project-specific

**Steps**:
1. Create `internal/testutils/` directory
2. Copy `pkg/testutils/*` to `internal/testutils/`
3. Update config import
4. Update all test file imports
5. Remove original `pkg/testutils/`

**Expected Duration**: 1-2 hours

### Phase 6: Cleanup and Documentation
**Purpose**: Remove compatibility aliases and update documentation

**Steps**:
1. Remove all compatibility alias packages from `pkg/`
2. Update any remaining import references
3. Update AGENTS.md with new import paths
4. Update any documentation referencing pkg/ structure
5. Verify all tests pass
6. Run full integration tests

**Expected Duration**: 1-2 hours

## Risk Assessment

### High-Risk Areas
1. **Event System**: `eventbus` and `outbox` are critical for system functionality
2. **Service Startup**: All services depend on migrated packages
3. **Database Operations**: Outbox pattern affects data consistency

### Mitigation Strategies
1. **Comprehensive Testing**: Unit, integration, and end-to-end tests
2. **Incremental Deployment**: One phase at a time
3. **Compatibility Aliases**: Temporary fallback during transition
4. **Rollback Procedures**: Documented rollback for each phase
5. **Monitoring**: Enhanced logging during migration

### Failure Scenarios
1. **Service Won't Start**: Import path issues or missing dependencies
2. **Event System Failure**: Eventbus or outbox migration issues
3. **Test Failures**: Import path updates in test files
4. **Configuration Issues**: Config loading problems

## Testing Strategy

### Pre-Migration Testing
1. **Baseline Tests**: Run full test suite to establish baseline
2. **Integration Tests**: Verify all services work together
3. **Performance Tests**: Establish performance benchmarks

### Phase-Specific Testing
1. **Unit Tests**: Run tests for migrated packages
2. **Import Tests**: Verify all imports resolve correctly
3. **Service Tests**: Start each service individually
4. **Functionality Tests**: Verify core features work

### Post-Migration Testing
1. **Full Test Suite**: Run all tests with new structure
2. **Integration Tests**: Verify service-to-service communication
3. **Performance Regression**: Compare against baseline
4. **End-to-End Tests**: Verify complete user workflows

### Automated Testing Commands
```bash
# Run all tests
make services-test

# Test specific service
go test ./cmd/customer/...

# Test specific package
go test ./internal/platform/config/...

# Run with coverage
go test -cover ./...
```

## Rollback Plan

### Phase-Level Rollback
Each phase includes specific rollback procedures:
1. **Stop Services**: Gracefully shutdown all running services
2. **Restore Files**: Restore original `pkg/` packages from git
3. **Update Imports**: Revert import statements to original paths
4. **Verify Functionality**: Run tests to ensure system works
5. **Document Issues**: Record problems for future improvement

### Complete Rollback
If entire migration needs to be reverted:
1. **Git Reset**: Reset to pre-migration commit
2. **Clean Build**: `go clean -modcache` and rebuild
3. **Verify Services**: Start all services and verify functionality
4. **Review Issues**: Analyze what went wrong

## Implementation Timeline

### Total Estimated Duration: 12-18 hours

**Phase 1**: 2 hours (Foundation packages)
**Phase 2**: 3 hours (Configuration layer)
**Phase 3**: 3 hours (Middleware packages)
**Phase 4**: 5 hours (Event infrastructure)
**Phase 5**: 2 hours (Test utilities)
**Phase 6**: 2 hours (Cleanup and documentation)
**Buffer**: 1-3 hours for unexpected issues

### Recommended Schedule
- **Day 1**: Phases 1-2 (Low risk, high confidence)
- **Day 2**: Phases 3-4 (Medium-high risk, more time needed)
- **Day 3**: Phases 5-6 (Finalization and cleanup)

## Success Criteria

### Technical Success
1. ✅ All packages successfully migrated to `internal/platform/`
2. ✅ All services start and run correctly
3. ✅ All tests pass with new structure
4. ✅ No performance regression
5. ✅ Import paths follow Go standards

### Operational Success
1. ✅ Zero downtime during migration
2. ✅ No data loss or corruption
3. ✅ All functionality preserved
4. ✅ Documentation updated
5. ✅ Team trained on new structure

## Post-Migration Benefits

### Immediate Benefits
1. **Standards Compliance**: Aligns with Go project layout standards
2. **Clear Boundaries**: Internal vs external package distinction
3. **Better Organization**: Platform infrastructure grouped together

### Long-term Benefits
1. **Maintainability**: Clearer package structure
2. **Security**: Internal packages not accessible to external modules
3. **Scalability**: Better foundation for future platform additions
4. **Developer Experience**: Easier to understand project structure

## Conclusion

This migration plan provides a structured, incremental approach to moving packages from `pkg/` to `internal/platform/` while maintaining system stability and minimizing risk. The phased approach allows for careful testing and rollback capabilities at each stage.

The migration aligns the project with Go best practices while preserving all existing functionality. With proper execution and testing, this will improve the project's long-term maintainability and adherence to standards.