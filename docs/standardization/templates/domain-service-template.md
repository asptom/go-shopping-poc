# Domain Service Template

Use this skeleton for new domain services under `internal/service/<domain>`.

## Goals

- Keep business logic in `internal/service/<domain>`.
- Keep transport mechanics in platform packages.
- Keep logger component naming stable and `snake_case`.

## Skeleton

```go
package mydomain

import (
    "context"
    "log/slog"
)

type Service struct {
    repo   Repository
    logger *slog.Logger
}

func NewService(repo Repository, logger *slog.Logger) *Service {
    if logger == nil {
        logger = slog.Default()
    }

    return &Service{
        repo:   repo,
        logger: logger.With("component", "mydomain_service"),
    }
}

func (s *Service) DoBusinessOperation(ctx context.Context, id string) error {
    log := s.logger.With("operation", "do_business_operation", "mydomain_id", id)

    if id == "" {
        log.Warn("Business operation rejected", "error", "missing id")
        return ErrInvalidID
    }

    if err := s.repo.Run(ctx, id); err != nil {
        log.Error("Business operation failed", "error", err.Error())
        return err
    }

    log.Info("Business operation completed")
    return nil
}
```

## Checklist

- Service depends on repository interface, not concrete implementation.
- Component logger key is present and stable.
- `operation` key is added at boundary/failure points.
- No transport concerns (HTTP decode/encode/status mapping) in service.
