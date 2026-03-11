# HTTP Handler Template

Use this skeleton for new HTTP handlers in domain packages.

## Goals

- Reuse `internal/platform/httpx` and `internal/platform/httperr`.
- Preserve domain-owned status/message mappings.
- Keep concise, structured logging with stable fields.

## Skeleton

```go
package mydomain

import (
    "errors"
    "log/slog"
    "net/http"

    "go-shopping-poc/internal/platform/httperr"
    "go-shopping-poc/internal/platform/httpx"
)

type Handler struct {
    service *Service
    logger  *slog.Logger
}

func NewHandler(service *Service, logger *slog.Logger) *Handler {
    if logger == nil {
        logger = slog.Default()
    }

    return &Handler{
        service: service,
        logger:  logger.With("component", "mydomain_handler"),
    }
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    requestID := r.Header.Get("X-Request-ID")
    log := h.logger.With("operation", "create_mydomain", "request_id", requestID)

    var req CreateRequest
    if err := httpx.DecodeJSON(r, &req); err != nil {
        log.Warn("Decode request failed", "error", err.Error())
        httperr.InvalidRequest(w, "Invalid JSON")
        return
    }

    obj, err := h.service.Create(r.Context(), req)
    if err != nil {
        switch {
        case errors.Is(err, ErrInvalidInput):
            httperr.Validation(w, "Invalid input")
        case errors.Is(err, ErrNotFound):
            httperr.NotFound(w, "Entity not found")
        default:
            log.Error("Create failed", "error", err.Error())
            httperr.Internal(w, "Create failed")
        }
        return
    }

    if err := httpx.WriteJSON(w, http.StatusCreated, obj); err != nil {
        log.Error("Encode response failed", "error", err.Error())
        httperr.Internal(w, "Failed to encode response")
        return
    }

    log.Info("Create completed")
}
```

## Checklist

- Uses `httpx` for decode/encode and param extraction.
- Uses `httperr` for transport error envelopes.
- Logger includes `component`, `operation`, and `request_id` when available.
- No duplicated local JSON helper stack unless compatibility requires a thin adapter.
