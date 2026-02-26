# Phase 6 Completion Plan - Product Domain Service

## Current Status

### Completed ✅
The following files have been updated to accept logger in constructors:
- `internal/service/product/service.go` - CatalogService has logger field, already uses slog
- `internal/service/product/service_admin.go` - AdminService has logger field in constructor, but still uses `log.Printf()` 
- `internal/service/product/repository.go` - Has logger field

### Remaining Tasks

#### 1. Event Handler (`internal/service/product/eventhandlers/on_cart_item_added.go`)
**Status:** NOT STARTED

**Changes needed:**
- Add `logger *slog.Logger` field to `OnCartItemAdded` struct
- Update `NewOnCartItemAdded` constructor to accept `logger *slog.Logger` parameter
- Replace all `log.Printf()` calls with `slogger.Debug/Info/Warn/Error` calls
- Replace all `log.Printf("[DEBUG]")`, `log.Printf("[INFO]")`, `log.Printf("[ERROR]")`, `log.Printf("[WARN]")` with appropriate slog calls

**Lines with log.Printf to migrate:**
- Line 32: `log.Printf("[ERROR] Product: Expected CartItemEvent, got %T", event)`
- Line 38: `log.Printf("[DEBUG] Product: Ignoring event type: %s", cartItemEvent.EventType)`
- Line 49-50: `log.Printf("[DEBUG] Product: Processing cart item added - CartID: %s, ProductID: %s, Quantity: %d", ...)`
- Line 55: `log.Printf("[ERROR] Product: Invalid product ID format: %s", payload.ProductID)`
- Line 62: `log.Printf("[DEBUG] Product: Product %s not found: %v", payload.ProductID, err)`
- Line 67: `log.Printf("[DEBUG] Product: Product %s is out of stock", payload.ProductID)`
- Line 71: `log.Printf("[DEBUG] Product: Product %s validated successfully", payload.ProductID)`
- Line 117: `log.Printf("[WARN] Product: Failed to trigger immediate outbox processing: %v", err)`
- Line 122-123: `log.Printf("[DEBUG] Product: Published validation result for product %s (available: %v)", ...)`

---

#### 2. HTTP Handler (`internal/service/product/handler.go`)
**Status:** NOT STARTED

**Changes needed:**
- Add `logger *slog.Logger` field to `CatalogHandler` struct
- Update `NewCatalogHandler` constructor to accept `logger *slog.Logger` parameter
- Replace `log.Printf()` calls with slog calls

**Lines with log.Printf to migrate:**
- Line 320: `log.Printf("[DEBUG] GetDirectImage: Handler invoked!")`
- Line 321: `log.Printf("[DEBUG] GetDirectImage: Method=%s, URL=%s, Path=%s", ...)`
- Line 327: `log.Printf("[DEBUG] GetDirectImage: URL params - productID='%s', imageName='%s'", ...)`
- Line 349: `log.Printf("[DEBUG] GetDirectImage: Constructed objectName: '%s'", objectName)`
- Line 352: `log.Printf("[DEBUG] GetDirectImage: Attempting to get object from bucket '%s', objectName: '%s'", ...)`
- Line 357: `log.Printf("[ERROR] GetDirectImage: Failed to get object from Minio: %v", err)`
- Line 379: `log.Printf("[ERROR] Failed to stream image: %v", err)`

---

#### 3. Admin HTTP Handler (`internal/service/product/handler_admin.go`)
**Status:** NOT STARTED

**Changes needed:**
- Check if it needs a logger (may be simpler, just needs migration)
- Replace any `log.Printf()` calls with slog

---

#### 4. Repository Files (5 files)
**Status:** NOT STARTED - These have `log.Printf()` calls that need migration

**Files to check:**
- `internal/service/product/repository_crud.go`
- `internal/service/product/repository_query.go`
- `internal/service/product/repository_image.go`
- `internal/service/product/repository_bulk.go`
- `internal/service/product/repository_util.go`

**Note:** The `productRepository` struct in `repository.go` already has a `logger` field, but the `NewProductRepository` constructor doesn't accept a logger parameter yet. Need to:
1. Update `NewProductRepository(db, outboxWriter, logger)` to accept logger
2. Update all callers to pass the logger

---

#### 5. Main Entry Point (`cmd/product/main.go`)
**Status:** NOT STARTED

**Changes needed:**
- Import `internal/platform/logging` package
- Import `internal/platform/providers` package
- Create a logger using the logging infrastructure
- Pass logger to `NewCatalogService`
- Update any other service/handler instantiations to pass logger

**Specific changes:**
- Line 86: `catalogService := product.NewCatalogService(???, catalogInfra, cfg)` - needs logger as first param
- Line 91: `cartItemHandler := eventhandlers.NewOnCartItemAdded(???, catalogService)` - needs logger
- Line 139: `catalogHandler := product.NewCatalogHandler(???, catalogService, minioStorage, cfg.MinIOBucket)` - needs logger

**Note:** Currently `NewCatalogService` signature in service.go shows it accepts logger as first param, but cmd/product/main.go line 86 shows it only passes 2 params. Need to verify actual signature and update main.go accordingly.

---

## Implementation Order

1. **Update repository.go** - Add logger parameter to `NewProductRepository`
2. **Update repository_*.go files** - Pass logger to NewProductRepository, migrate log.Printf
3. **Update eventhandlers/on_cart_item_added.go** - Add logger, migrate log.Printf
4. **Update handler.go** - Add logger, migrate log.Printf
5. **Update handler_admin.go** - Check and migrate if needed
6. **Update cmd/product/main.go** - Create logger provider and pass to all services

## Key Decisions Needed

1. **Constructor signatures**: Should `NewCatalogService`, `NewAdminService` take logger as first parameter (like other services) or should it follow a different pattern?

2. **Logger source**: Should the Product service create its own logger from the logging package, or receive it from the main entry point via provider?

3. **Minimal vs Complete**: 
   - **Minimal approach**: Just update constructors to accept logger, defer log.Printf conversions
   - **Complete approach**: Full migration of all log.Printf to slog

## Notes from Previous Work

- The CatalogService in `service.go` already has `logger *slog.Logger` field and uses slog
- The AdminService in `service_admin.go` has the logger field but doesn't use it (still has ~50+ log.Printf calls)
- The repository has a logger field but constructor doesn't accept it
- This is the largest domain service with ~99 log.Printf calls across multiple files
