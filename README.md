# Go Shopping POC

A proof-of-concept microservices-based shopping platform built with Go, demonstrating Clean Architecture principles, event-driven design, and Kubernetes deployment.

## Project Overview

This project implements a modern e-commerce platform with:
- **Event-driven architecture** using Kafka for inter-service communication
- **Clean Architecture** with strict separation of concerns
- **Kubernetes deployment** with Rancher Desktop
- **PostgreSQL** for data persistence
- **Type-safe event handling** with Go generics

## Architecture

### Clean Architecture Layers

```
internal/
├── contracts/events/              # Pure DTOs - Event data structures
├── platform/                      # Shared infrastructure (HOW)
│   ├── service/                   # Service lifecycle management
│   ├── event/bus/                 # Message transport abstraction
│   ├── event/handler/             # Generic event utilities
│   └── [config, cors, etc.]       # Other shared infrastructure
└── service/[domain]/              # Business logic + domain entities (WHAT)
    ├── entity.go                  # Domain models and validation
    ├── service.go                 # Business logic
    ├── repository.go              # Data access
    └── handler.go                 # HTTP handlers
```

### Event System

Contracts-first approach with type-safe event handling:
1. **Event Contracts**: Pure data structures in `internal/contracts/events/`
2. **Transport Layer**: Kafka behind `internal/platform/event/bus/` interface
3. **Generic Utilities**: Reusable patterns in `internal/platform/event/handler/`
4. **Service Integration**: Unified lifecycle via `internal/platform/service/`

## Services

- **Customer Service**: Customer management with PostgreSQL persistence
- **EventReader Service**: Event processing and handler orchestration
- **WebSocket Service**: Real-time communication (future implementation)

## Infrastructure

- **PostgreSQL**: Customer and order data storage
- **Kafka**: Event streaming and message broker
- **Keycloak**: OIDC authentication and authorization
- **MinIO**: S3-compatible object storage
- **Kubernetes**: Container orchestration with Rancher Desktop

## Quick Start

### Prerequisites
- Go 1.21+
- Rancher Desktop (Kubernetes)
- kubectl configured
- Docker

### Local Development Setup

1. **Clone and setup**
   ```bash
   git clone <repository-url>
   cd go-shopping-poc
   ```

2. **Deploy infrastructure**
   ```bash
   make deploy-all  # PostgreSQL, Kafka, Keycloak, MinIO
   ```

3. **Build and deploy services**
   ```bash
   make services-build
   make customer-deploy
   make eventreader-deploy
   ```

4. **Verify deployment**
   ```bash
   kubectl get pods
   kubectl logs -f deployment/customer
   ```

## Event System Usage

### Publishing Events
```go
event := events.NewCustomerCreated(customerID, customerData)
publisher := outbox.NewPublisher(eventBus)
err := publisher.PublishEvent(ctx, event)
```

### Handling Events
```go
utils := handler.NewEventUtils()
err := utils.HandleEventWithValidation(ctx, event, func(ctx context.Context, event events.Event) error {
    // Business logic here
    return nil
})
```

## Testing

```bash
# Unit tests
make services-test

# Integration tests (requires Kafka)
go test -tags=integration ./tests/integration/...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Development

### Adding New Event Types
1. Define event contract in `internal/contracts/events/[domain].go`
2. Implement Event interface methods
3. Create EventFactory[T] for reconstruction
4. Add validation to EventUtils if needed
5. Add comprehensive tests

### Service Development Pattern
```go
type MyService struct {
    *service.EventServiceBase
}

func NewMyService(eventBus bus.Bus) *MyService {
    return &MyService{
        EventServiceBase: service.NewEventServiceBase("my-service", eventBus),
    }
}
```

## Build Commands

- `make services-build` - Build all services
- `make services-test` - Run tests
- `make deploy-all` - Deploy to Kubernetes
- `go run ./cmd/customer` - Run locally

## Contributing

Follows Clean Architecture principles. See `AGENTS.md` for detailed guidelines. 
