# Outbox Pattern Implementation

This package provides a complete outbox pattern implementation for reliable event publishing in distributed systems. The outbox pattern ensures that events are stored in the database before being published to external systems, guaranteeing at-least-once delivery.

## Components

### Writer
The `Writer` handles storing events in the outbox table within database transactions. Events are written atomically with business data changes.

### Publisher
The `Publisher` reads events from the outbox table and publishes them to external systems (e.g., message brokers). It runs as a background process with configurable batch processing.

### Provider
The `OutboxProvider` encapsulates the setup and configuration of outbox components, providing a clean interface for dependency injection.

## Usage

### Basic Setup

```go
// Load dependencies
dbProvider := database.NewDatabaseProvider("postgres://...")
db := dbProvider.GetDatabase()

eventBusProvider := event.NewEventBusProvider(event.EventBusConfig{
    WriteTopic: "events",
    GroupID:    "my-service",
})
eventBus := eventBusProvider.GetEventBus()

// Create outbox provider
outboxProvider, err := outbox.NewOutboxProvider(db, eventBus)
if err != nil {
    log.Fatal(err)
}

// Get components
writer := outboxProvider.GetOutboxWriter()
publisher := outboxProvider.GetOutboxPublisher()

// Start the publisher
publisher.Start()
defer publisher.Stop()
```

### Writing Events

```go
// Within a database transaction
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()

// Perform business logic
_, err = tx.Exec(ctx, "INSERT INTO users (name) VALUES ($1)", "John")
if err != nil {
    return err
}

// Write event to outbox
event := events.NewUserCreated(userID, userData)
err = writer.WriteEvent(ctx, tx, event)
if err != nil {
    return err
}

// Commit transaction (includes event)
return tx.Commit()
```

### Publishing Events

The publisher automatically processes events in the background:

```go
// Start publishing (typically in service startup)
publisher.Start()

// The publisher will:
// 1. Read unpublished events from outbox table
// 2. Publish them to the event bus
// 3. Mark them as published
// 4. Clean up old events

// Stop publishing (typically in service shutdown)
publisher.Stop()
```

## Configuration

The outbox pattern uses platform configuration loaded from `config/platform-outbox.env`:

```env
OUTBOX_BATCH_SIZE=10
OUTBOX_DELETE_BATCH_SIZE=10
OUTBOX_PROCESS_INTERVAL=5s
OUTBOX_MAX_RETRIES=3
```

## Clean Architecture

This implementation follows clean architecture principles:

- **Contracts Layer**: Event definitions in `internal/contracts/events/`
- **Platform Layer**: Infrastructure components (writer, publisher, provider)
- **Service Layer**: Business logic using the provider interface

The provider pattern enables:
- Dependency injection
- Testability with mocks
- Loose coupling between components
- Consistent component lifecycle management

## Error Handling

The implementation includes comprehensive error handling:

- Configuration validation
- Database connection errors
- Event publishing failures with retry logic
- Graceful shutdown handling
- Structured logging for debugging

## Testing

The provider includes comprehensive tests covering:

- Valid and invalid configuration scenarios
- Component creation and retrieval
- Error handling for missing dependencies
- Graceful handling of unavailable configuration in test environments</content>
<parameter name="filePath">internal/platform/outbox/README.md