# Go Shopping POC

**Project Name**: go-shopping-poc

**Description**: This project is a proof-of-concept implementation of a microservices-based shopping platform built with Go. It demonstrates Clean Architecture principles, event-driven design, and modern Go development practices.

## Architecture

This project implements Clean Architecture principles with clear separation of concerns:

- **Contracts Layer** (`internal/contracts/`): Pure data structures and interfaces
- **Platform Layer** (`internal/platform/`): Shared infrastructure and reusable utilities
- **Service Layer** (`internal/service/`): Domain-specific business logic
- **Entity Layer** (`internal/entity/`): Domain models and database mappings

### Event System

The event system uses a contracts-first approach with type-safe event handling:

1. **Event Contracts**: Define event structures in `internal/contracts/events/`
2. **Transport Layer**: Kafka implementation abstracted behind interfaces
3. **Generic Handlers**: Reusable event processing utilities
4. **Service Integration**: Event-driven services using shared infrastructure

### Services

- **Customer Service**: Customer management with PostgreSQL persistence
- **EventReader Service**: Event processing and handler orchestration
- **WebSocket Service**: Real-time communication (future)

### Infrastructure

- **PostgreSQL**: Primary database for customer and order data
- **Kafka**: Event streaming and message broker
- **Keycloak**: OIDC authentication and authorization
- **MinIO**: S3-compatible object storage for product images
- **Kubernetes**: Container orchestration with Rancher Desktop

## Event System Usage

### Creating and Publishing Events

```go
import (
    "go-shopping-poc/internal/contracts/events"
    "go-shopping-poc/internal/platform/outbox"
)

// Create an event (example: customer, order, or any domain event)
event := events.NewCustomerCreated(customerID, customerData) // Replace with your event type

// Publish via outbox pattern
publisher := outbox.NewPublisher(eventBus)
err := publisher.PublishEvent(ctx, event)
```

### Handling Events with Generic Utilities

```go
import (
    "go-shopping-poc/internal/platform/event/handler"
    "go-shopping-poc/internal/contracts/events"
)

utils := handler.NewEventUtils()
err := utils.HandleEventWithValidation(ctx, event, func(ctx context.Context, event events.Event) error {
    // Your business logic here
    utils.LogEventProcessing(ctx, event.Type(), utils.GetEventID(event), utils.GetResourceID(event))
    return nil
})
```

### Service Handler Registration

```go
import (
    "go-shopping-poc/internal/service/eventreader"
    "go-shopping-poc/internal/service/yourdomain" // Replace with your domain package
)

// Register typed event handlers for any event type
err := eventreader.RegisterHandler(
    service,
    yourEventHandler.CreateFactory(),  // Factory for your event type
    yourEventHandler.CreateHandler(),  // Handler for your event type
)
```

## Local Development Setup

### Prerequisites

- **Go 1.21+**
- **Rancher Desktop** (Kubernetes cluster)
- **kubectl** configured for Rancher Desktop
- **Docker** (for local builds)

### Environment Setup

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd go-shopping-poc
   ```

2. **Start infrastructure services**
   ```bash
   # Deploy all services to Kubernetes
   make deploy-all

   # Or deploy individual components
   make kafka-deploy
   make postgres-deploy
   make keycloak-deploy
   make minio-deploy
   ```

3. **Configure environment variables**
   ```bash
   # Copy and modify environment files
   cp resources/kafka/topics.def.example resources/kafka/topics.def
   cp resources/postgresql/config/postgresql-config.yaml.example resources/postgresql/config/postgresql-config.yaml
   ```

4. **Build and run services**
   ```bash
   # Build all services
   make services-build

   # Deploy application services
   make customer-deploy
   make eventreader-deploy
   make websocket-deploy
   ```

### Service Verification

```bash
# Check pod status
kubectl get pods

# Check service health
kubectl get services

# View logs
kubectl logs -f deployment/customer
kubectl logs -f deployment/eventreader
```

## Testing

### Unit Tests

```bash
# Run all tests
make services-test

# Run specific service tests
go test ./internal/service/customer/...

# Run platform tests
go test ./internal/platform/...
```

### Integration Tests

```bash
# Run integration test suite (requires Kafka)
go test -tags=integration ./tests/integration/...

# Run specific integration test
go test -tags=integration -run TestEndToEndEventFlow ./tests/integration/
```

### Test Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Minimum coverage requirements: 80% for platform code, 70% for services
```

## Development Workflow

### Adding New Event Types

1. **Define event contract** in `internal/contracts/events/[domain].go`
2. **Implement Event interface** with Type(), Topic(), Payload(), ToJSON()
3. **Create EventFactory[T]** for type-safe JSON reconstruction
4. **Add validation logic** to `EventUtils.ValidateEvent()` if needed
5. **Update event type matcher** in `EventTypeMatcher` if needed
6. **Add comprehensive tests** for new event type

### Service Development

```go
// For event-driven services:
type MyService struct {
    *service.EventServiceBase
    // domain-specific fields
}

func NewMyService(eventBus bus.Bus) *MyService {
    return &MyService{
        EventServiceBase: service.NewEventServiceBase("my-service", eventBus),
    }
}
```

## Build Commands

- `make services-build` - Build all services
- `make services-test` - Run all tests
- `make services-lint` - Run golangci-lint
- `make deploy-all` - Deploy all services to Kubernetes
- `go run ./cmd/customer` - Run customer service locally

## Contributing

This project follows Clean Architecture principles. See `AGENTS.md` for detailed maintenance guidelines and contribution standards. 
