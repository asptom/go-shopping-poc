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
- **EventReader Service**: Example of event processing and handler orchestration
- **Product Service**: Product catalog query
- **Product Admin: Administrative services for products
- **Product Loader**: Command-line tool for bulk product ingestion from CSV files
- **WebSocket Service**: Example of real-time communication (future implementation)

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
- Product: topics, group, port
- Product: topic, group, port
- WebSocket: URL, timeout, path
- Location: `deply/k8s/service/<service>/<service>-configmap.yaml

## Testing

Testing is limited to business functionality as testing code became seriously bloated.
The only tests we have now are for customer but more will be added in the future:

```bash
go test -v ./cmd/customer... ./internal/service/customer/...

or 

gmake customer-test
```