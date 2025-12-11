# Migration Quick Reference

## Migration Order (Bottom-Up)
1. **logging** → errors → event (Phase 1)
2. **config** (Phase 2)
3. **cors** → websocket (Phase 3)
4. **eventbus** → outbox (Phase 4)
5. **testutils** → internal/testutils/ (Phase 5)
6. **Cleanup aliases** (Phase 6)

## Key Commands
```bash
# Test before migration
make services-test

# Test specific phase
go test ./internal/platform/config/...

# Build all services
make services-build

# Run linting
make services-lint
```

## Import Path Changes
```go
// Before
"go-shopping-poc/pkg/config"
"go-shopping-poc/pkg/logging"

// After  
"go-shopping-poc/internal/platform/config"
"go-shopping-poc/internal/platform/logging"
```

## Files to Update per Phase

### Phase 1 (logging, errors, event)
- `pkg/config/config.go` (logging import)

### Phase 2 (config)
- `cmd/customer/main.go`
- `cmd/eventreader/main.go` 
- `cmd/websocket/main.go`
- `pkg/testutils/testutils.go`
- `pkg/cors/cors.go`
- `pkg/websocket/websocket.go`

### Phase 3 (cors, websocket)
- `cmd/customer/main.go` (cors)
- `cmd/websocket/main.go` (websocket)

### Phase 4 (eventbus, outbox)
- `cmd/customer/main.go`
- `cmd/eventreader/main.go`
- `internal/service/customer/repository.go`
- `pkg/outbox/publisher.go`

### Phase 5 (testutils)
- All test files in `internal/service/customer/`

## Risk Levels
- 🔴 **HIGH**: eventbus, outbox (critical system components)
- 🟡 **MEDIUM**: config, websocket, cors (widely used)
- 🟢 **LOW**: logging, errors, event, testutils (simple utilities)

## Rollback Commands
```bash
# Reset to pre-migration state
git checkout -- pkg/ internal/
git checkout main

# Clean module cache
go clean -modcache
go mod tidy
```