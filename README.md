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
- **Product Service**: Product catalog management with CSV ingestion and image processing
- **Product Loader**: Command-line tool for bulk product ingestion from CSV files
- **WebSocket Service**: Real-time communication (future implementation)

## Infrastructure

- **PostgreSQL**: Customer, product, and order data storage
- **Kafka**: Event streaming and message broker
- **Keycloak**: OIDC authentication and authorization
- **MinIO**: S3-compatible object storage for product images
- **Kubernetes**: Container orchestration with Rancher Desktop

## Deployment

### Quick Start: Kubernetes

#### Prerequisites
- Kubernetes cluster (Rancher Desktop, Minikube, or cloud)
- kubectl configured to connect to your cluster
- Docker
- GNU Make for deployment

#### Deploy to Development

```bash
# Navigate to project root directory
# Make will be used to build and deploy platforms and services
# All passwords are generated at deployment

gmake platform 
gmake services

or

gmake install

# These targets will:
# 1. Deploy platform services (PostgreSQL, Kafka, MinIO, Keycloak)
# 2. Deploy application services (customer, product, etc.)
# 3. Show deployment status
```

#### Verify Deployment

```bash
# Check all pods
kubectl get pods --all-namespaces

# Check ConfigMaps
kubectl get configmaps --all-namespaces

# Check Secrets (names only, values are hidden)
kubectl get secrets --all-namespaces

# View pod environment
kubectl describe pod -n pocstore customer-0 | grep -A 20 "Environment"
```

### Local Development / Environment Variables
Deploying to a local kubernetes environment using gmake is currently the only
method being used for development and testing.  The platforms and services consume
environment variables that are currently only configured in the Makefiles and within
platform or service configmaps. 

Testing will use programmatically generated environment variables.

#### Configuration Architecture

The project follows clean architecture with proper configuration separation:

**Platform ConfigMap**: Shared infrastructure (database pools, Kafka brokers, CORS, outbox)
- Used by: customer, product, eventreader, eventwriter services
- Location: `deploy/k8s/config/platform-configmap-for-services.yaml`

**Service ConfigMaps**: Service-specific settings (topics, ports, cache settings)
- Customer: topics, group, port
- Product: cache, MinIO bucket, CSV batch size
- EventReader: topics, group
- WebSocket: URL, timeout, path
- EventWriter: topics, group
- Location: `deply/k8s/service/<service>/<service>-configmap.yaml

### Adding New Services

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

## Product Service & Loader

### Product Service API

The Product Service provides RESTful endpoints for product catalog management:

```bash
# Get product by ID
GET /products/{id}

# Create product
POST /products

# Update product
PUT /products/{id}

# Delete product
DELETE /products/{id}

# Search products
GET /products/search?q={query}

# Get products by category
GET /products/category/{category}

# Get products by brand
GET /products/brand/{brand}

# Get in-stock products
GET /products/in-stock

# Ingest products from CSV
POST /products/ingest
```

### Product Loader CLI

The Product Loader is a command-line tool for bulk product ingestion from CSV files:

```bash
# Basic usage - load products from CSV file
go run ./cmd/product-loader -csv=/path/to/products.csv

# Load with custom batch ID for tracking
go run ./cmd/product-loader -csv=/path/to/products.csv -batch-id=my-batch-001

# Disable image caching (always download fresh images)
go run ./cmd/product-loader -csv=/path/to/products.csv -use-cache=false

# Reset image cache before processing
go run ./cmd/product-loader -csv=/path/to/products.csv -reset-cache=true
```

**Command-line Options:**
- `-csv`: Path to CSV file containing products (required)
- `-batch-id`: Optional batch ID for tracking (auto-generated if not provided)
- `-use-cache`: Use cached images when available (default: true)
- `-reset-cache`: Reset image cache before processing (default: false)

### Product Data Model

Products include comprehensive catalog information:
- Basic info: name, description, pricing, inventory
- Categorization: category, brand, model number
- Media: main image and multiple additional images
- Attributes: color, size, country code, custom attributes

## Testing

Testing is limited to business functionality as testing code became seriously bloated.
The only tests we have now are for customer but more will be added in the future:

```bash
go test -v ./cmd/customer... ./internal/service/customer/...

or 

gmake customer-test
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