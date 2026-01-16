# WARNING:  This is out of date - updates to come soon


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
- Bash shell (for deployment scripts)

#### Step 1: Prepare Secrets

```bash
# Navigate to base secrets directory
cd deployment/k8s/base/secret/

# Copy example files and rename them
cp customer-secret.yaml.example customer-secret.yaml
cp product-secret.yaml.example product-secret.yaml
cp postgres-secret.yaml.example postgres-secret.yaml
cp minio-secret.yaml.example minio-secret.yaml
cp keycloak-secret.yaml.example keycloak-secret.yaml

# Edit each file and replace CHANGE_ME_* with actual values
# DO NOT commit actual secrets to version control
```

#### Step 2: Deploy to Development

```bash
# Navigate to kubernetes directory
cd deployments/kubernetes

# Run development deployment script
./deploy-dev.sh

# The script will:
# 1. Check/create secrets from templates (interactive)
# 2. Apply all ConfigMaps
# 3. Apply all Secrets
# 4. Deploy platform services (PostgreSQL, Kafka, MinIO, Keycloak)
# 5. Deploy application services (customer, product, etc.)
# 6. Show deployment status
```

#### Step 3: Deploy to Production

```bash
# Navigate to kubernetes directory
cd deployments/kubernetes

# Edit secrets with PRODUCTION values first!
# IMPORTANT: Make sure secrets contain production passwords/URLs

# Run production deployment script
./deploy-prod.sh

# The script will:
# 1. Apply all ConfigMaps (same as dev)
# 2. Apply production secrets
# 3. Deploy platform services
# 4. Deploy application services
# 5. Set production replicas (2 for customer, 2 for product)
# 6. Add production labels
# 7. Show deployment status
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

#### Update Configuration

```bash
# Edit ConfigMap
kubectl edit configmap platform-config -n pocstore

# Edit Secret
kubectl edit secret customer-secret -n pocstore

# Restart pods to apply changes
kubectl rollout restart statefulset/customer -n pocstore
```

### Local Development

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

#### Using Docker Compose for Infrastructure

```bash
# Copy environment template
cp .env.example .env
# Edit .env with your values

# Start platform services
docker-compose up -d postgres kafka minio keycloak

# Run services in separate terminals (with .env sourced)
source .env
go run ./cmd/customer
go run ./cmd/product
```

### Environment Variables

This project uses environment variables for all configuration.

#### Configuration Architecture

The project follows clean architecture with proper configuration separation:

**Platform ConfigMap**: Shared infrastructure (database pools, Kafka brokers, CORS, outbox)
- Used by: customer, product, eventreader, eventwriter services
- Location: `deployment/k8s/base/configmap/platform-configmap.yaml`

**Service ConfigMaps**: Service-specific settings (topics, ports, cache settings)
- Customer: topics, group, port
- Product: cache, MinIO bucket, CSV batch size
- EventReader: topics, group
- WebSocket: URL, timeout, path
- EventWriter: topics, group

**Secrets**: Sensitive data (passwords, database URLs, API keys)
- Stored in: `deployment/k8s/base/secret/`
- Templates: `.yaml.example` files (safe to commit)
- Actual secrets: `.yaml` files (never commit)

**Local Development**: `.env.example` template for local environment
- Copy to `.env` and fill in actual values
- Environment variable expansion supported: `$VAR` in values

### Adding New Services

To add a new service to the deployment:

1. **Create Service ConfigMap** in `deployment/k8s/base/configmap/`
2. **Create Service Secret Template** in `deployment/k8s/base/secret/`
3. **Add envFrom to Deployment** (reference platform-config + service-config + service-secret)
4. **Update Deployment Scripts** to apply new ConfigMap, Secret, and Deployment
5. **Test with `./deploy-dev.sh`**

See `docs/kubernetes-deployment-guide.md` for detailed instructions.

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
    make product-deploy  # Deploy product service
    ```

4. **Load product data (optional)**
    ```bash
    go run ./cmd/product-loader -csv=/path/to/products.csv  # Run product ingestion from CSV
    ```

5. **Verify deployment**
   ```bash
   kubectl get pods
   kubectl logs -f deployment/customer
   kubectl logs -f deployment/product
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
- `go run ./cmd/customer` - Run customer service locally
- `go run ./cmd/product-loader -csv=/path/to/products.csv` - Run product loader locally

## Contributing

Follows Clean Architecture principles. See `AGENTS.md` for detailed guidelines. 
