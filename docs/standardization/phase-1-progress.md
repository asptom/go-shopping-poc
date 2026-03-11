# Phase 1 Progress - Platform Contract Hardening

## Metadata

- Date: 2026-03-11
- Author/LLM: OpenCode (GPT-5.3 Codex)
- Branch/Commit at start: `gpt_refactor` / `4d66678`
- Scope: Phase 1 only (`internal/platform/*` contracts and docs/tests)

## Checklist

| Item | Status | Evidence |
|---|---|---|
| Generalize platform SSE internals (STD-001) | done | `internal/platform/sse/handler.go`, `internal/platform/sse/hub.go`, `internal/platform/sse/client.go`, `internal/platform/sse/provider.go` now use stream-agnostic naming/config. |
| Introduce domain-agnostic HTTP transport helpers (STD-002) | done | New package `internal/platform/httpx` with strict decode, JSON write, no-content, and required path/query param helpers. |
| Introduce transport error helper package (STD-003) | done | New package `internal/platform/httperr` with envelope helpers (`Send`, `SendWithCode`, `InvalidRequest`, `Validation`, `NotFound`, `Internal`). |
| Add package responsibility docs (no domain logic rule) | done | `internal/platform/httpx/doc.go`, `internal/platform/httperr/doc.go`. |
| Add/adjust platform unit tests | done | `internal/platform/httpx/json_test.go`, `internal/platform/httperr/response_test.go`, `internal/platform/sse/handler_test.go`, `internal/platform/sse/provider_test.go`. |
| Keep `internal/platform/* -> internal/service/*` forbidden import boundary | done | `rg -n '"go-shopping-poc/internal/service/' internal/platform --glob '*.go'` returned 0 matches. |
| Keep cart endpoint behavior compatible while removing platform cart assumptions | done | Cart wiring now sets SSE handler options in `cmd/cart/main.go` to preserve `Missing cart ID`, `cartId`, and `cart_id` semantics. |

## API Surface Summary

### `internal/platform/httpx`

- `DecodeJSON(r *http.Request, target any) error`
  - strict JSON decode (`DisallowUnknownFields`) + single-object enforcement.
- `WriteJSON(w http.ResponseWriter, status int, value any) error`
  - marshals JSON and writes content-type + status.
- `WriteNoContent(w http.ResponseWriter)`
  - writes HTTP 204.
- `RequirePathParam(r *http.Request, key string) (string, error)`
- `RequireQueryParam(r *http.Request, key string) (string, error)`
  - returns typed `ParamError`; supports `errors.Is` against `ErrMissingPathParam` and `ErrMissingQueryParam`.

### `internal/platform/httperr`

- `Send(w, statusCode, errorType, message)`
- `SendWithCode(w, statusCode, errorType, message, code)`
- `InvalidRequest(w, message)`
- `Validation(w, message)`
- `NotFound(w, message)`
- `Internal(w, message)`

### `internal/platform/sse` hardening

- Introduced generic handler configuration API:
  - `HandlerOption`
  - `WithPathParam(...)`
  - `WithMissingIDMessage(...)`
  - `WithConnectedIDField(...)`
  - `WithLogIDKey(...)`
- Provider now supports `WithHandlerOptions(...)` for service-specific compatibility at composition time.
- Internal naming generalized from cart-specific to stream-oriented (`streamID`, `stream_id`), including hub/client internals and logs.

## Architecture Boundary Checks

- Checked forbidden import direction:
  - Command: `rg -n '"go-shopping-poc/internal/service/' internal/platform --glob '*.go'`
  - Result: no matches.
- Cross-domain behavior not changed in this phase; domain services were not migrated to `httpx`/`httperr` yet (Phase 2+).

## Compatibility Notes for Phase 2

- `httpx` and `httperr` are now available for cart pilot migration in Phase 2.
- SSE platform is domain-agnostic internally, but cart HTTP behavior is preserved through wiring in `cmd/cart/main.go` using handler options.
- Existing cart eventhandler code that calls `sse.Hub.Publish(cartID, ...)` remains compatible (parameter name changed internally only).

## Deferred / Known Gaps

- Domain handlers still use local transport mechanics (`decodeJSON`, `writeJSON`, inline `json.NewDecoder`/`json.NewEncoder`) until Phase 2/3 migrations.
- Logging standard keys (`operation`, `request_id`, `duration_ms`) are not globally applied in this phase by design.
- Some cart terminology remains in `internal/platform/sse/provider_test.go` only as compatibility test data.

## Machine-Verifiable Done Report

| Check ID | Command | Expected | Actual | Result |
|---|---|---|---|---|
| P1-DONE-001 | `go test ./internal/platform/...` | exit code 0 | pass | PASS |
| P1-DONE-002 | `go test ./...` | exit code 0 | pass | PASS |
| P1-DONE-003 | `go list ./...` | package list emitted | package list emitted | PASS |
| P1-DONE-004 | `rg -n '"go-shopping-poc/internal/service/' internal/platform --glob '*.go'` | 0 matches | 0 matches | PASS |
| P1-DONE-005 | `go list ./... | rg 'internal/platform/httpx|internal/platform/httperr'` | both packages listed | both listed | PASS |
