# Customer Service Bootstrap Tests

## What We Test

### Health Endpoint (`main_test.go`)
- ✅ Returns HTTP 200 OK status
- ✅ Returns correct JSON response body
- ✅ Sets correct Content-Type header
- ✅ Works with different HTTP methods

**Why**: Critical for Kubernetes liveness/readiness probes and production monitoring.

## What We DON'T Test

- ❌ main() function (wiring/glue code, exits on failure)
- ❌ HTTP server startup (stdlib, integration concern)
- ❌ Graceful shutdown (platform-specific, OS behavior)
- ❌ Database/Event bus creation (external dependencies)
- ❌ Route registration (implementation detail)
- ❌ Signal handling (OS-level behavior)

## Philosophy

Tests follow principle: **Test business behavior, not implementation details.**

## Running Tests

```bash
# Run all customer bootstrap tests
go test -v ./cmd/customer/...

# Run specific test
go test -v ./cmd/customer/... -run TestHealthHandler
```
