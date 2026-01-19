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

### Local Development
TODO: This needs to be looked at.  Deploying to k8s is the only current way to 
deploy using make.  The .env files need to be reviewed and updated.  They are out of date and refect the old way to deploy.

#### Using .env File

```bash
# Copy environment template
cp .env.example .env
# Edit .env with your values

# Source environment (optional - services load automatically)
source .env

# Run services in separate terminals
go run ./cmd/customer
go run ./cmd/product
```

### Environment Variables

This project uses environment variables for all configuration.

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

**Local Development**: `.env.example` template for local environment
- Copy to `.env` and fill in actual values
- Environment variable expansion supported: `$VAR` in values
- Note:  this needs to be looked at env.example may need to be updated

### Adding New Services

## Quick Start

### Prerequisites
- Go 1.24+
- Rancher Desktop (Kubernetes)
- kubectl configured
- Docker

### Local Development Setup

TODO:  Revamp

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

TODO: Testing needs to be redone and simplified

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