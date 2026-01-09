```markdown
# Kubernetes ConfigMaps and Secrets Migration Plan

## Executive Summary
This migration plan implements Kubernetes ConfigMaps and Secrets following the current per-service directory structure, where each service deployment has its own ConfigMap in its directory. This approach maintains clear service boundaries while providing proper configuration management.

## Migration Goals
- **Environment-Only Configuration**: Services read from environment variables only
- **Per-Service ConfigMaps**: Each service has its own ConfigMap in its deployment directory
- **Centralized Secrets**: Sensitive data in shared `secrets/` directory
- **Development Template**: `.env.example` for local development reference
- **Docker Compose**: Optional local development setup
- **Complete Documentation**: Deployment guides for all scenarios

## Directory Structure

### Target Structure
```
deployment/k8s/
├── customer/
│   ├── customer-deploy.yaml (MODIFY - add envFrom)
│   └── customer-configmap.yaml (NEW)
├── product/
│   ├── product-deploy.yaml (MODIFY - add envFrom)
│   └── product-configmap.yaml (NEW)
├── eventreader/
│   ├── eventreader-deploy.yaml (MODIFY - add envFrom)
│   └── eventreader-configmap.yaml (NEW)
├── websocket/
│   ├── websocket-deployment.yaml (MODIFY - add envFrom)
│   ├── websocket-ingress.yaml
│   ├── websocket-service.yaml
│   └── websocket-configmap.yaml (NEW)
├── eventwriter/
│   ├── eventwriter-deployment.yaml (MODIFY - add envFrom)
│   └── eventwriter-configmap.yaml (NEW)
├── postgresql/
│   ├── postgresql-deploy.yaml (MODIFY - add envFrom)
│   └── postgresql-configmap.yaml (NEW)
├── kafka/
│   └── kafka-deploy.yaml (no changes needed)
├── minio/
│   └── minio-deploy.yaml (MODIFY - add secretRef)
└── keycloak/
    ├── keycloak-deploy.yaml (MODIFY - add envFrom)
    └── keycloak-configmap.yaml (NEW)
```

### Secrets Directory
```
deployment/k8s/secrets/ (NEW)
├── customer-secret.yaml (NEW)
├── product-secret.yaml (NEW)
├── postgres-secret.yaml (NEW)
├── minio-secret.yaml (NEW)
└── keycloak-secret.yaml (NEW)
```

## Phase 1: Create Service-Specific ConfigMaps

### Customer Service ConfigMap
**File**: `deployment/k8s/customer/customer-configmap.yaml`

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: customer-config
  namespace: shopping
  labels:
    app: customer
data:
  CUSTOMER_SERVICE_PORT: ":8080"
  CUSTOMER_WRITE_TOPIC: "CustomerEvents"
  CUSTOMER_GROUP: "CustomerGroup"
  CUSTOMER_OUTBOX_PROCESSING_INTERVAL: "10s"
  CUSTOMER_READ_TOPICS: ""
  
  # Platform database configuration
  DATABASE_MAX_OPEN_CONNS: "25"
  DATABASE_MAX_IDLE_CONNS: "25"
  DATABASE_CONN_MAX_LIFETIME: "5m"
  DATABASE_CONN_MAX_IDLE_TIME: "5m"
  DATABASE_CONNECT_TIMEOUT: "30s"
  DATABASE_QUERY_TIMEOUT: "60s"
  DATABASE_HEALTH_CHECK_INTERVAL: "30s"
  DATABASE_HEALTH_CHECK_TIMEOUT: "5s"
  
  # Platform Kafka configuration
  KAFKA_BROKERS: "kafka-clusterip.kafka.svc.cluster.local:9092"
  
  # Platform Outbox configuration
  OUTBOX_BATCH_SIZE: "10"
  OUTBOX_DELETE_BATCH_SIZE: "10"
  OUTBOX_PROCESS_INTERVAL: "10s"
  OUTBOX_MAX_RETRIES: "3"
  
  # Platform CORS configuration
  CORS_ALLOWED_ORIGINS: "http://localhost:4200,http://localhost:3000"
  CORS_ALLOWED_METHODS: "GET,POST,PUT,DELETE,PATCH,OPTIONS"
  CORS_ALLOWED_HEADERS: "Content-Type,Authorization,X-Requested-With"
  CORS_ALLOW_CREDENTIALS: "true"
  CORS_MAX_AGE: "3600"
```

### Product Service ConfigMap
**File**: `deployment/k8s/product/product-configmap.yaml`

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: product-config
  namespace: shopping
  labels:
    app: product
data:
  PRODUCT_SERVICE_PORT: ":8080"
  IMAGE_CACHE_DIR: "/tmp/product-cache"
  IMAGE_CACHE_MAX_AGE: "24h"
  IMAGE_CACHE_MAX_SIZE: "0"
  CSV_BATCH_SIZE: "100"
  MINIO_BUCKET: "productimages"
  MINIO_ENDPOINT_KUBERNETES: "minio.minio.svc.cluster.local:9000"
  MINIO_ENDPOINT_LOCAL: "localhost:9000"
  MINIO_TLS_VERIFY: "false"
  
  # Platform database configuration
  DATABASE_MAX_OPEN_CONNS: "25"
  DATABASE_MAX_IDLE_CONNS: "25"
  DATABASE_CONN_MAX_LIFETIME: "5m"
  DATABASE_CONN_MAX_IDLE_TIME: "5m"
  DATABASE_CONNECT_TIMEOUT: "30s"
  DATABASE_QUERY_TIMEOUT: "60s"
  DATABASE_HEALTH_CHECK_INTERVAL: "30s"
  DATABASE_HEALTH_CHECK_TIMEOUT: "5s"
  
  # Platform Kafka configuration
  KAFKA_BROKERS: "kafka-clusterip.kafka.svc.cluster.local:9092"
  
  # Platform Outbox configuration
  OUTBOX_BATCH_SIZE: "10"
  OUTBOX_DELETE_BATCH_SIZE: "10"
  OUTBOX_PROCESS_INTERVAL: "10s"
  OUTBOX_MAX_RETRIES: "3"
  
  # Platform CORS configuration
  CORS_ALLOWED_ORIGINS: "http://localhost:4200,http://localhost:3000"
  CORS_ALLOWED_METHODS: "GET,POST,PUT,DELETE,PATCH,OPTIONS"
  CORS_ALLOWED_HEADERS: "Content-Type,Authorization,X-Requested-With"
  CORS_ALLOW_CREDENTIALS: "true"
  CORS_MAX_AGE: "3600"
```

### EventReader Service ConfigMap
**File**: `deployment/k8s/eventreader/eventreader-configmap.yaml`

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: eventreader-config
  namespace: shopping
  labels:
    app: eventreader
data:
  EVENT_READER_WRITE_TOPIC: "ReaderEvents"
  EVENT_READER_READ_TOPICS: "CustomerEvents"
  EVENT_READER_GROUP: "EventReaderGroup"
  KAFKA_BROKERS: "kafka-clusterip.kafka.svc.cluster.local:9092"
```

### WebSocket Service ConfigMap
**File**: `deployment/k8s/websocket/websocket-configmap.yaml`

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: websocket-config
  namespace: shopping
  labels:
    app: websocket
data:
  WEBSOCKET_URL: "ws://localhost:8080/ws"
  WEBSOCKET_TIMEOUT: "30s"
  WEBSOCKET_READ_BUFFER: "1024"
  WEBSOCKET_WRITE_BUFFER: "1024"
  WEBSOCKET_PORT: ":8080"
  WEBSOCKET_PATH: "/ws"
  WEBSOCKET_ALLOWED_ORIGINS: "http://localhost:4200,http://localhost:3000"
```

### EventWriter Service ConfigMap
**File**: `deployment/k8s/eventwriter/eventwriter-configmap.yaml`

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: eventwriter-config
  namespace: shopping
  labels:
    app: eventwriter
data:
  EVENT_WRITER_WRITE_TOPIC: "WriterEvents"
  EVENT_WRITER_GROUP: "EventWriterGroup"
  KAFKA_BROKERS: "kafka-clusterip.kafka.svc.cluster.local:9092"
```

## Phase 2: Create Kubernetes Secrets

### Customer Service Secret
**File**: `deployment/k8s/secrets/customer-secret.yaml`

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: customer-secret
  namespace: shopping
  labels:
    app: customer
type: Opaque
stringData:
  PSQL_CUSTOMER_DB_URL: "postgres://customersuser:changeme_customer_password@postgres.postgres.svc.cluster.local:5432/customersdb?sslmode=disable"
```

### Product Service Secret
**File**: `deployment/k8s/secrets/product-secret.yaml`

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: product-secret
  namespace: shopping
  labels:
    app: product
type: Opaque
stringData:
  PSQL_PRODUCT_DB_URL: "postgres://productsuser:changeme_product_password@postgres.postgres.svc.cluster.local:5432/productsdb?sslmode=disable"
  MINIO_ACCESS_KEY: "changeme_minio_access_key"
  MINIO_SECRET_KEY: "changeme_minio_secret_key"
```

### PostgreSQL Admin Secret
**File**: `deployment/k8s/secrets/postgres-secret.yaml`

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: postgres-secret
  namespace: postgres
  labels:
    app: postgres
type: Opaque
stringData:
  POSTGRES_PASSWORD: "changeme_postgres_admin"
```

### MinIO Credentials Secret
**File**: `deployment/k8s/secrets/minio-secret.yaml`

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: minio-credentials
  namespace: minio
  labels:
    app: minio
type: Opaque
stringData:
  MINIO_USER: "changeme_minio_root_user"
  MINIO_PASSWORD: "changeme_minio_root_password"
```

### Keycloak Secret
**File**: `deployment/k8s/secrets/keycloak-secret.yaml`

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: keycloak-secret
  namespace: keycloak
  labels:
    app: keycloak
type: Opaque
stringData:
  KC_BOOTSTRAP_ADMIN_USERNAME: "admin"
  KC_BOOTSTRAP_ADMIN_PASSWORD: "changeme_keycloak_admin"
  KC_DB_PASSWORD: "changeme_keycloak_db_password"
```

## Phase 3: Create .env.example Template

**File**: `.env.example`

```bash
# =============================================================================
# Shopping POC Environment Variables Template
# =============================================================================
# Copy this file to .env and update with your actual values
# DO NOT commit .env to version control

# =============================================================================
# PostgreSQL Database Configuration
# =============================================================================
# Customer database
PSQL_CUSTOMER_DB=customersdb
PSQL_CUSTOMER_ROLE=customersuser
PSQL_CUSTOMER_PASSWORD=your_customer_password_here
PSQL_CUSTOMER_DB_URL=postgres://$PSQL_CUSTOMER_ROLE:$PSQL_CUSTOMER_PASSWORD@localhost:30432/$PSQL_CUSTOMER_DB?sslmode=disable

# Product database
PSQL_PRODUCT_DB=productsdb
PSQL_PRODUCT_ROLE=productsuser
PSQL_PRODUCT_PASSWORD=your_product_password_here
PSQL_PRODUCT_DB_URL=postgres://$PSQL_PRODUCT_ROLE:$PSQL_PRODUCT_PASSWORD@localhost:30432/$PSQL_PRODUCT_DB?sslmode=disable

# Platform database settings (shared by all services)
DATABASE_MAX_OPEN_CONNS=25
DATABASE_MAX_IDLE_CONNS=25
DATABASE_CONN_MAX_LIFETIME=5m
DATABASE_CONN_MAX_IDLE_TIME=5m
DATABASE_CONNECT_TIMEOUT=30s
DATABASE_QUERY_TIMEOUT=60s
DATABASE_HEALTH_CHECK_INTERVAL=30s
DATABASE_HEALTH_CHECK_TIMEOUT=5s

# =============================================================================
# Kafka Configuration
# =============================================================================
KAFKA_BROKERS=kafka-clusterip.kafka.svc.cluster.local:9092

# Customer service Kafka settings
CUSTOMER_WRITE_TOPIC=CustomerEvents
CUSTOMER_READ_TOPICS=
CUSTOMER_GROUP=CustomerGroup

# EventReader service Kafka settings
EVENT_READER_WRITE_TOPIC=ReaderEvents
EVENT_READER_READ_TOPICS=CustomerEvents
EVENT_READER_GROUP=EventReaderGroup

# =============================================================================
# Outbox Pattern Configuration
# =============================================================================
OUTBOX_BATCH_SIZE=10
OUTBOX_DELETE_BATCH_SIZE=10
OUTBOX_PROCESS_INTERVAL=10s
OUTBOX_MAX_RETRIES=3

# Customer outbox interval
CUSTOMER_OUTBOX_PROCESSING_INTERVAL=10s

# =============================================================================
# MinIO Object Storage Configuration
# =============================================================================
# Platform MinIO settings
MINIO_ENDPOINT_KUBERNETES=minio.minio.svc.cluster.local:9000
MINIO_ENDPOINT_LOCAL=localhost:9000
MINIO_ACCESS_KEY=your_minio_access_key
MINIO_SECRET_KEY=your_minio_secret_key
MINIO_TLS_VERIFY=false

# Product service MinIO settings
MINIO_BUCKET=productimages

# =============================================================================
# CORS Configuration
# =============================================================================
CORS_ALLOWED_ORIGINS=http://localhost:4200,http://localhost:3000
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,PATCH,OPTIONS
CORS_ALLOWED_HEADERS=Content-Type,Authorization,X-Requested-With
CORS_ALLOW_CREDENTIALS=true
CORS_MAX_AGE=3600

# =============================================================================
# WebSocket Configuration
# =============================================================================
WEBSOCKET_URL=ws://localhost:8080/ws
WEBSOCKET_TIMEOUT=30s
WEBSOCKET_READ_BUFFER=1024
WEBSOCKET_WRITE_BUFFER=1024
WEBSOCKET_PORT=:8080
WEBSOCKET_PATH=/ws
WEBSOCKET_ALLOWED_ORIGINS=http://localhost:4200,http://localhost:3000

# =============================================================================
# Customer Service Configuration
# =============================================================================
CUSTOMER_SERVICE_PORT=:8080

# =============================================================================
# Product Service Configuration
# =============================================================================
PRODUCT_SERVICE_PORT=:8080
IMAGE_CACHE_DIR=/tmp/product-cache
IMAGE_CACHE_MAX_AGE=24h
IMAGE_CACHE_MAX_SIZE=0
CSV_BATCH_SIZE=100

# =============================================================================
# PostgreSQL Configuration (Platform Service)
# =============================================================================
POSTGRES_DB=postgresdb
POSTGRES_USER=postgresadmin
POSTGRES_PASSWORD=your_postgres_password

# =============================================================================
# Keycloak Configuration (Platform Service)
# =============================================================================
KEYCLOAK_ADMIN_USERNAME=admin
KEYCLOAK_ADMIN_PASSWORD=your_keycloak_admin_password
KEYCLOAK_DB_USERNAME=keycloak
KEYCLOAK_DB_PASSWORD=your_keycloak_db_password
KEYCLOAK_DB_URL_DATABASE=postgres://keycloak:keycloak@postgres.postgres.svc.cluster.local:5432/keycloak?sslmode=disable

# =============================================================================
# MinIO Configuration (Platform Service)
# =============================================================================
MINIO_ROOT_USER=your_minio_root_user
MINIO_ROOT_PASSWORD=your_minio_root_password
```

## Phase 4: Update Service Deployments

### Customer Service Deployment
**File**: `deployment/k8s/customer/customer-deploy.yaml` - ADD TO spec.template.containers[0].envFrom

```yaml
containers:
  - name: store-customer
    image: localhost:5000/go-shopping-poc/customer:1.0
    imagePullPolicy: Always
    envFrom:
      - configMapRef:
          name: customer-config
      - secretRef:
          name: customer-secret
    ports:
      - containerPort: 8080
    readinessProbe:
      httpGet:
        path: /health
        port: 8080
```

### Product Service Deployment
**File**: `deployment/k8s/product/product-deploy.yaml` - ADD TO spec.template.containers[0].envFrom AND spec.template.volumes[0]

```yaml
containers:
  - name: store-product
    image: localhost:5000/go-shopping-poc/product:1.0
    imagePullPolicy: Always
    envFrom:
      - configMapRef:
          name: product-config
      - secretRef:
          name: product-secret
    ports:
      - containerPort: 8080
    readinessProbe:
      httpGet:
        path: /health
        port: 8080
    volumeMounts:
      - name: product-cache
        mountPath: /tmp/product-cache
```

```yaml
volumes:
  - name: product-cache
    emptyDir: {}
```

### EventReader Service Deployment
**File**: `deployment/k8s/eventreader/eventreader-deploy.yaml` - ADD TO spec.template.containers[0].envFrom

```yaml
containers:
  - name: store-eventreader
    image: localhost:5000/go-shopping-poc/eventreader:1.0
    imagePullPolicy: Always
    envFrom:
      - configMapRef:
          name: eventreader-config
```

### WebSocket Service Deployment
**File**: `deployment/k8s/websocket/websocket-deployment.yaml` - ADD TO spec.template.containers[0].envFrom

```yaml
containers:
  - name: store-websocket
    image: localhost:5000/go-shopping-poc/websocket:1.0
    imagePullPolicy: Always
    envFrom:
      - configMapRef:
          name: websocket-config
```

## Phase 5: Update Platform Service Deployments

### PostgreSQL Deployment
**File**: `deployment/k8s/postgresql/postgresql-deploy.yaml` - REPLACE env section with envFrom

```yaml
containers:
  - name: postgres
    image: docker.io/postgres:18.0
    imagePullPolicy: IfNotPresent
    ports:
      - containerPort: 5432
        name: postgres
    envFrom:
      - configMapRef:
          name: postgresql-config
      - secretRef:
          name: postgres-secret
          optional: true
```

### MinIO Deployment
**File**: `deployment/k8s/minio/minio-deploy.yaml` - MODIFY spec.template.containers[0].envFrom (uncomment secretRef)

```yaml
containers:
  - name: minio
    image: minio/minio:latest
    imagePullPolicy: IfNotPresent
    args:
    - server
    - /data
    - --console-address=:9001
    envFrom:
      - secretRef:
          name: minio-credentials
```

### Keycloak Deployment
**File**: `deployment/k8s/keycloak/keycloak-deploy.yaml` - ADD TO spec.template.containers[0].envFrom

```yaml
containers:
  - name: keycloak
    image: quay.io/keycloak/keycloak:26.4.2
    args: ["start", "--import-realm"]
    envFrom:
      - configMapRef:
          name: keycloak-config
      - secretRef:
          name: keycloak-secret
          optional: true
```

## Phase 6: Create Docker Compose (Optional)

**File**: `docker-compose.yml` (NEW)

```yaml
version: '3.8'

services:
  # PostgreSQL Database
  postgres:
    image: postgres:18.0
    container_name: go-shopping-postgres
    environment:
      POSTGRES_DB: ${POSTGRES_DB:-postgresdb}
      POSTGRES_USER: ${POSTGRES_USER:-postgresadmin}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-postgresadminpw}
    ports:
      - "${POSTGRES_PORT:-30432}:5432"
    volumes:
      - postgres-data:/var/lib/postgresql/data
      - ./resources/postgresql/models:/docker-entrypoint-initdb.d:ro
    networks:
      - shopping-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-postgresadmin}"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Kafka
  kafka:
    image: apache/kafka:latest
    container_name: go-shopping-kafka
    environment:
      KAFKA_NODE_ID: "1"
      KAFKA_PROCESS_ROLES: "broker,controller"
      KAFKA_LISTENERS: "PLAINTEXT://0.0.0.0:9092,CONTROLLER://0.0.0.0:9093"
      KAFKA_ADVERTISED_LISTENERS: "PLAINTEXT://localhost:9092"
      KAFKA_CONTROLLER_LISTENER_NAMES: "CONTROLLER"
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: "CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT"
      KAFKA_CONTROLLER_QUORUM_VOTERS: "1@kafka:9093"
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: "1"
      KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR: "1"
      KAFKA_TRANSACTION_STATE_LOG_MIN_ISR: "1"
      KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS: "0"
      KAFKA_NUM_PARTITIONS: "3"
    ports:
      - "${KAFKA_PORT:-9092}:9092"
    networks:
      - shopping-network
    healthcheck:
      test: ["CMD", "kafka-topics.sh", "--bootstrap-server", "localhost:9092", "--list"]
      interval: 30s
      timeout: 10s
      retries: 5

  # MinIO
  minio:
    image: minio/minio:latest
    container_name: go-shopping-minio
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER:-minioadmin}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD:-minioadmin}
    command: server /data --console-address ":9001"
    ports:
      - "${MINIO_API_PORT:-9000}:9000"
      - "${MINIO_CONSOLE_PORT:-9001}:9001"
    volumes:
      - minio-data:/data
    networks:
      - shopping-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 30s
      timeout: 20s
      retries: 3

  # Keycloak
  keycloak:
    image: quay.io/keycloak/keycloak:26.4.2
    container_name: go-shopping-keycloak
    environment:
      KC_BOOTSTRAP_ADMIN_USERNAME: ${KC_BOOTSTRAP_ADMIN_USERNAME:-admin}
      KC_BOOTSTRAP_ADMIN_PASSWORD: ${KC_BOOTSTRAP_ADMIN_PASSWORD:-admin}
      KC_DB_URL_DATABASE: ${KC_DB_URL_DATABASE}
      KC_DB_USERNAME: ${KC_DB_USERNAME:-keycloak}
      KC_DB_PASSWORD: ${KC_DB_PASSWORD:-keycloak}
      KC_PROXY_HEADERS: "xforwarded"
      KC_HTTP_ENABLED: "true"
      KC_HOSTNAME_STRICT: "false"
      KC_HEALTH_ENABLED: "true"
      KC_CACHE: "ispn"
    command: start-dev
    ports:
      - "${KEYCLOAK_PORT:-8080}:8080"
    depends_on:
      - postgres
    networks:
      - shopping-network
    healthcheck:
      test: ["CMD-SHELL", "exec 3<>/dev/tcp/localhost/8080 && echo -e 'GET /health/ready HTTP/1.1\\r\\nHost: localhost\\r\\n\\r\\n' >&3"]
      interval: 30s
      timeout: 10s
      retries: 5

volumes:
  postgres-data:
    driver: local
  minio-data:
    driver: local

networks:
  shopping-network:
    driver: bridge
```

## Phase 7: Documentation

### Update README.md

**Add to README.md:**

```markdown
## Deployment

### Environment Variables

This project uses environment variables for all configuration. Copy `.env.example` to `.env` and update with your values.

```bash
cp .env.example .env
# Edit .env with your actual values
# DO NOT commit .env to version control
```

### Kubernetes Deployment

#### Quick Start

```bash
# 1. Create namespaces
kubectl create namespace shopping
kubectl create namespace kafka
kubectl create namespace postgres
kubectl create namespace minio
kubectl create namespace keycloak

# 2. Apply platform services
kubectl apply -f deployment/k8s/postgresql/
kubectl apply -f deployment/k8s/kafka/
kubectl apply -f deployment/k8s/minio/
kubectl apply -f deployment/k8s/keycloak/

# 3. Create and apply secrets
kubectl apply -f deployment/k8s/secrets/customer-secret.yaml
kubectl apply -f deployment/k8s/secrets/product-secret.yaml
kubectl apply -f deployment/k8s/secrets/postgres-secret.yaml
kubectl apply -f deployment/k8s/secrets/minio-secret.yaml
kubectl apply -f deployment/k8s/secrets/keycloak-secret.yaml

# 4. Apply application services
kubectl apply -f deployment/k8s/customer/
kubectl apply -f deployment/k8s/product/
kubectl apply -f deployment/k8s/eventreader/
kubectl apply -f deployment/k8s/websocket/
kubectl apply -f deployment/k8s/eventwriter/
```

#### Verify Deployment

```bash
# Check pods
kubectl get pods --all-namespaces

# Check services
kubectl get svc --all-namespaces

# Check ConfigMaps
kubectl get configmaps --all-namespaces

# Check Secrets
kubectl get secrets --all-namespaces

# View pod environment
kubectl describe pod -n shopping <pod-name>
```

#### Update Configuration

```bash
# Edit ConfigMap
kubectl edit configmap customer-config -n shopping

# Edit Secret
kubectl edit secret customer-secret -n shopping

# Restart pods to pick up changes
kubectl rollout restart deployment store-customer -n shopping
```

### Local Development with Docker Compose

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
```

### Create Documentation File

**File**: `docs/kubernetes-deployment-guide.md` (NEW)

```markdown
# Kubernetes Deployment Guide

## Overview

This guide explains how to deploy the Shopping POC using Kubernetes ConfigMaps and Secrets.

## Architecture

The project uses environment variables exclusively for configuration:
- **ConfigMaps**: Store non-sensitive configuration (ports, timeouts, topics)
- **Secrets**: Store sensitive data (passwords, database URLs, API keys)
- **Per-Service Structure**: Each service has its own ConfigMap in its deployment directory
- **Centralized Secrets**: Cross-service secrets in shared `secrets/` directory

## Prerequisites

- Kubernetes cluster (Rancher Desktop, Minikube, or cloud provider)
- kubectl configured to connect to your cluster
- Docker registry available (localhost:5000 for local development)

## Directory Structure

```
deployment/k8s/
├── customer/
│   ├── customer-deploy.yaml
│   └── customer-configmap.yaml
├── product/
│   ├── product-deploy.yaml
│   └── product-configmap.yaml
├── eventreader/
│   ├── eventreader-deploy.yaml
│   └── eventreader-configmap.yaml
├── websocket/
│   ├── websocket-deployment.yaml
│   └── websocket-configmap.yaml
├── eventwriter/
│   ├── eventwriter-deployment.yaml
│   └── eventwriter-configmap.yaml
├── postgresql/
│   ├── postgresql-deploy.yaml
│   └── postgresql-configmap.yaml
├── kafka/
│   └── kafka-deploy.yaml
├── minio/
│   └── minio-deploy.yaml
├── keycloak/
│   └── keycloak-deploy.yaml
└── secrets/
    ├── customer-secret.yaml
    ├── product-secret.yaml
    ├── postgres-secret.yaml
    ├── minio-secret.yaml
    └── keycloak-secret.yaml
```

## Deployment Steps

### 1. Create Namespaces

```bash
kubectl create namespace shopping
kubectl create namespace kafka
kubectl create namespace postgres
kubectl create namespace minio
kubectl create namespace keycloak
```

### 2. Apply Platform Services

```bash
# PostgreSQL
kubectl apply -f deployment/k8s/postgresql/

# Kafka
kubectl apply -f deployment/k8s/kafka/

# MinIO
kubectl apply -f deployment/k8s/minio/

# Keycloak
kubectl apply -f deployment/k8s/keycloak/
```

### 3. Apply Application Service Secrets

```bash
kubectl apply -f deployment/k8s/secrets/customer-secret.yaml
kubectl apply -f deployment/k8s/secrets/product-secret.yaml
kubectl apply -f deployment/k8s/secrets/postgres-secret.yaml
kubectl apply -f deployment/k8s/secrets/minio-secret.yaml
kubectl apply -f deployment/k8s/secrets/keycloak-secret.yaml
```

### 4. Apply Application Services

```bash
kubectl apply -f deployment/k8s/customer/
kubectl apply -f deployment/k8s/product/
kubectl apply -f deployment/k8s/eventreader/
kubectl apply -f deployment/k8s/websocket/
kubectl apply -f deployment/k8s/eventwriter/
```

### 5. Verify Deployment

```bash
# Check all pods
kubectl get pods --all-namespaces

# Check ConfigMaps
kubectl get configmaps --all-namespaces

# Check Secrets
kubectl get secrets --all-namespaces

# Check specific pod
kubectl describe pod -n shopping store-customer-0
```

## Updating Configuration

### Edit ConfigMaps

```bash
# Edit customer ConfigMap
kubectl edit configmap customer-config -n shopping

# Edit product ConfigMap
kubectl edit configmap product-config -n shopping

# Edit eventreader ConfigMap
kubectl edit configmap eventreader-config -n shopping

# Edit websocket ConfigMap
kubectl edit configmap websocket-config -n shopping
```

### Edit Secrets

```bash
# Edit customer Secret
kubectl edit secret customer-secret -n shopping

# Edit product Secret
kubectl edit secret product-secret -n shopping

# Edit postgres Secret
kubectl edit secret postgres-secret -n postgres

# Edit minio Secret
kubectl edit secret minio-credentials -n minio

# Edit keycloak Secret
kubectl edit secret keycloak-secret -n keycloak
```

### Restart Services

```bash
# Restart customer service
kubectl rollout restart deployment store-customer -n shopping

# Restart product service
kubectl rollout restart deployment store-product -n shopping

# Restart eventreader service
kubectl rollout restart statefulset store-eventreader -n shopping

# Restart websocket service
kubectl rollout restart deployment store-websocket -n shopping
```

## Troubleshooting

### ConfigMap Not Found

```bash
# Verify ConfigMap exists
kubectl get configmap customer-config -n shopping

# Check pod logs
kubectl logs -n shopping deploy/store-customer-0

# Common issues:
# - ConfigMap name mismatch (check deployment references ConfigMap name)
# - Namespace mismatch (ensure both in same namespace)
# - Environment variable name typo (check ConfigMap key names)
```

### Secret Not Found

```bash
# Verify Secret exists
kubectl get secret customer-secret -n shopping

# Check pod status
kubectl get pods -n shopping

# Common issues:
# - Secret name mismatch (check deployment references Secret name)
# - Namespace mismatch (ensure both in same namespace)
# - Base64 encoding errors (Secret values must be base64 encoded)
```

### Environment Variables Not Loaded

```bash
# Describe pod to see environment
kubectl describe pod -n shopping store-customer-0

# Search for specific variable
kubectl describe pod -n shopping store-customer-0 | grep PSQL_CUSTOMER_DB_URL

# Common issues:
# - Variable name mismatch (ConfigMap key name vs. environment variable name)
# - Variable not set in ConfigMap/Secret
# - Secret referenced but not mounted in envFrom
```

## Security Best Practices

1. **Never Commit Secrets**: Secret files should never be committed to version control
2. **Rotate Credentials**: Change passwords regularly and update Secrets
3. **Use Different Credentials**: Use different passwords for development, staging, production
4. **Limit Secret Access**: Use RBAC to limit which services can access which Secrets
5. **Encrypt Secrets**: Enable Kubernetes encryption at rest for production clusters
6. **Audit Access**: Regularly review Secret access logs
7. **Delete Unused Secrets**: Remove Secrets that are no longer needed
```

### Update AGENTS.md

Add to recent session summary:

```markdown
## Recent Session Summary (Today)

### Kubernetes ConfigMaps and Secrets Migration Plan Created ✅

**Problem Identified**: Environment-only configuration code changes completed, but Kubernetes deployment configuration missing (ConfigMaps, Secrets, Docker Compose, documentation).

**Plan Created**: Comprehensive 7-phase migration plan to complete environment-only configuration:
- **Phase 1**: Create service-specific ConfigMaps in per-service directories
- **Phase 2**: Create Secrets for sensitive data (database URLs, passwords)
- **Phase 3**: Create .env.example template for local development
- **Phase 4**: Update service deployments to use envFrom
- **Phase 5**: Update platform deployments (PostgreSQL, MinIO, Keycloak)
- **Phase 6**: Create docker-compose.yml for local development
- **Phase 7**: Update documentation with deployment instructions

**Implementation Scope**:
- **13 ConfigMap/Secret files** to create (7 ConfigMaps, 5 Secrets, 1 .env.example, 1 docker-compose.yml)
- **8 deployment files** to modify with envFrom configuration
- **2 documentation files** to create (README updates, deployment guide)

**Files to Create**:
- `deployment/k8s/customer/customer-configmap.yaml`
- `deployment/k8s/product/product-configmap.yaml`
- `deployment/k8s/eventreader/eventreader-configmap.yaml`
- `deployment/k8s/websocket/websocket-configmap.yaml`
- `deployment/k8s/eventwriter/eventwriter-configmap.yaml`
- `deployment/k8s/postgresql/postgresql-configmap.yaml`
- `deployment/k8s/keycloak/keycloak-configmap.yaml`
- `deployment/k8s/secrets/customer-secret.yaml`
- `deployment/k8s/secrets/product-secret.yaml`
- `deployment/k8s/secrets/postgres-secret.yaml`
- `deployment/k8s/secrets/minio-secret.yaml`
- `deployment/k8s/secrets/keycloak-secret.yaml`
- `.env.example` (repository root)
- `docker-compose.yml` (repository root, optional)
- `docs/kubernetes-deployment-guide.md` (new)

**Files to Modify**:
- `deployment/k8s/customer/customer-deploy.yaml`
- `deployment/k8s/product/product-deploy.yaml`
- `deployment/k8s/eventreader/eventreader-deploy.yaml`
- `deployment/k8s/websocket/websocket-deployment.yaml`
- `deployment/k8s/eventwriter/eventwriter-deployment.yaml`
- `deployment/k8s/postgresql/postgresql-deploy.yaml`
- `deployment/k8s/minio/minio-deploy.yaml`
- `deployment/k8s/keycloak/keycloak-deploy.yaml`
- `README.md`
- `AGENTS.md`

**Status**: ✅ Migration plan complete - ready for implementation

**Note**: This plan maintains current per-service directory structure while providing proper Kubernetes ConfigMaps and Secrets for environment-only configuration.
```
```</think>Here is the complete Kubernetes ConfigMaps and Secrets migration plan in a markdown code block that you can manually add to the manifest file:

```markdown
# Kubernetes ConfigMaps and Secrets Migration Plan

## Executive Summary
This migration plan implements Kubernetes ConfigMaps and Secrets following the current per-service directory structure, where each service deployment has its own ConfigMap in its directory. This approach maintains clear service boundaries while providing proper configuration management.

## Migration Goals
- **Environment-Only Configuration**: Services read from environment variables only
- **Per-Service ConfigMaps**: Each service has its own ConfigMap in its deployment directory
- **Centralized Secrets**: Sensitive data in shared `secrets/` directory
- **Development Template**: `.env.example` for local development reference
- **Docker Compose**: Optional local development setup
- **Complete Documentation**: Deployment guides for all scenarios

## Directory Structure

### Target Structure
```
deployment/k8s/
├── customer/
│   ├── customer-deploy.yaml (MODIFY - add envFrom)
│   └── customer-configmap.yaml (NEW)
├── product/
│   ├── product-deploy.yaml (MODIFY - add envFrom)
│   └── product-configmap.yaml (NEW)
├── eventreader/
│   ├── eventreader-deploy.yaml (MODIFY - add envFrom)
│   └── eventreader-configmap.yaml (NEW)
├── websocket/
│   ├── websocket-deployment.yaml (MODIFY - add envFrom)
│   ├── websocket-ingress.yaml
│   ├── websocket-service.yaml
│   └── websocket-configmap.yaml (NEW)
├── eventwriter/
│   ├── eventwriter-deployment.yaml (MODIFY - add envFrom)
│   └── eventwriter-configmap.yaml (NEW)
├── postgresql/
│   ├── postgresql-deploy.yaml (MODIFY - add envFrom)
│   └── postgresql-configmap.yaml (NEW)
├── kafka/
│   └── kafka-deploy.yaml (no changes needed)
├── minio/
│   └── minio-deploy.yaml (MODIFY - add secretRef)
└── keycloak/
    ├── keycloak-deploy.yaml (MODIFY - add envFrom)
    └── keycloak-configmap.yaml (NEW)
```

### Secrets Directory
```
deployment/k8s/secrets/ (NEW)
├── customer-secret.yaml (NEW)
├── product-secret.yaml (NEW)
├── postgres-secret.yaml (NEW)
├── minio-secret.yaml (NEW)
└── keycloak-secret.yaml (NEW)
```

## Phase 1: Create Service-Specific ConfigMaps

### Customer Service ConfigMap
**File**: `deployment/k8s/customer/customer-configmap.yaml`

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: customer-config
  namespace: shopping
  labels:
    app: customer
data:
  CUSTOMER_SERVICE_PORT: ":8080"
  CUSTOMER_WRITE_TOPIC: "CustomerEvents"
  CUSTOMER_GROUP: "CustomerGroup"
  CUSTOMER_OUTBOX_PROCESSING_INTERVAL: "10s"
  CUSTOMER_READ_TOPICS: ""
  
  # Platform database configuration
  DATABASE_MAX_OPEN_CONNS: "25"
  DATABASE_MAX_IDLE_CONNS: "25"
  DATABASE_CONN_MAX_LIFETIME: "5m"
  DATABASE_CONN_MAX_IDLE_TIME: "5m"
  DATABASE_CONNECT_TIMEOUT: "30s"
  DATABASE_QUERY_TIMEOUT: "60s"
  DATABASE_HEALTH_CHECK_INTERVAL: "30s"
  DATABASE_HEALTH_CHECK_TIMEOUT: "5s"
  
  # Platform Kafka configuration
  KAFKA_BROKERS: "kafka-clusterip.kafka.svc.cluster.local:9092"
  
  # Platform Outbox configuration
  OUTBOX_BATCH_SIZE: "10"
  OUTBOX_DELETE_BATCH_SIZE: "10"
  OUTBOX_PROCESS_INTERVAL: "10s"
  OUTBOX_MAX_RETRIES: "3"
  
  # Platform CORS configuration
  CORS_ALLOWED_ORIGINS: "http://localhost:4200,http://localhost:3000"
  CORS_ALLOWED_METHODS: "GET,POST,PUT,DELETE,PATCH,OPTIONS"
  CORS_ALLOWED_HEADERS: "Content-Type,Authorization,X-Requested-With"
  CORS_ALLOW_CREDENTIALS: "true"
  CORS_MAX_AGE: "3600"
```

### Product Service ConfigMap
**File**: `deployment/k8s/product/product-configmap.yaml`

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: product-config
  namespace: shopping
  labels:
    app: product
data:
  PRODUCT_SERVICE_PORT: ":8080"
  IMAGE_CACHE_DIR: "/tmp/product-cache"
  IMAGE_CACHE_MAX_AGE: "24h"
  IMAGE_CACHE_MAX_SIZE: "0"
  CSV_BATCH_SIZE: "100"
  MINIO_BUCKET: "productimages"
  MINIO_ENDPOINT_KUBERNETES: "minio.minio.svc.cluster.local:9000"
  MINIO_ENDPOINT_LOCAL: "localhost:9000"
  MINIO_TLS_VERIFY: "false"
  
  # Platform database configuration
  DATABASE_MAX_OPEN_CONNS: "25"
  DATABASE_MAX_IDLE_CONNS: "25"
  DATABASE_CONN_MAX_LIFETIME: "5m"
  DATABASE_CONN_MAX_IDLE_TIME: "5m"
  DATABASE_CONNECT_TIMEOUT: "30s"
  DATABASE_QUERY_TIMEOUT: "60s"
  DATABASE_HEALTH_CHECK_INTERVAL: "30s"
  DATABASE_HEALTH_CHECK_TIMEOUT: "5s"
  
  # Platform Kafka configuration
  KAFKA_BROKERS: "kafka-clusterip.kafka.svc.cluster.local:9092"
  
  # Platform Outbox configuration
  OUTBOX_BATCH_SIZE: "10"
  OUTBOX_DELETE_BATCH_SIZE: "10"
  OUTBOX_PROCESS_INTERVAL: "10s"
  OUTBOX_MAX_RETRIES: "3"
  
  # Platform CORS configuration
  CORS_ALLOWED_ORIGINS: "http://localhost:4200,http://localhost:3000"
  CORS_ALLOWED_METHODS: "GET,POST,PUT,DELETE,PATCH,OPTIONS"
  CORS_ALLOWED_HEADERS: "Content-Type,Authorization,X-Requested-With"
  CORS_ALLOW_CREDENTIALS: "true"
  CORS_MAX_AGE: "3600"
```

### EventReader Service ConfigMap
**File**: `deployment/k8s/eventreader/eventreader-configmap.yaml`

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: eventreader-config
  namespace: shopping
  labels:
    app: eventreader
data:
  EVENT_READER_WRITE_TOPIC: "ReaderEvents"
  EVENT_READER_READ_TOPICS: "CustomerEvents"
  EVENT_READER_GROUP: "EventReaderGroup"
  KAFKA_BROKERS: "kafka-clusterip.kafka.svc.cluster.local:9092"
```

### WebSocket Service ConfigMap
**File**: `deployment/k8s/websocket/websocket-configmap.yaml`

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: websocket-config
  namespace: shopping
  labels:
    app: websocket
data:
  WEBSOCKET_URL: "ws://localhost:8080/ws"
  WEBSOCKET_TIMEOUT: "30s"
  WEBSOCKET_READ_BUFFER: "1024"
  WEBSOCKET_WRITE_BUFFER: "1024"
  WEBSOCKET_PORT: ":8080"
  WEBSOCKET_PATH: "/ws"
  WEBSOCKET_ALLOWED_ORIGINS: "http://localhost:4200,http://localhost:3000"
```

### EventWriter Service ConfigMap
**File**: `deployment/k8s/eventwriter/eventwriter-configmap.yaml`

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: eventwriter-config
  namespace: shopping
  labels:
    app: eventwriter
data:
  EVENT_WRITER_WRITE_TOPIC: "WriterEvents"
  EVENT_WRITER_GROUP: "EventWriterGroup"
  KAFKA_BROKERS: "kafka-clusterip.kafka.svc.cluster.local:9092"
```

## Phase 2: Create Kubernetes Secrets

### Customer Service Secret
**File**: `deployment/k8s/secrets/customer-secret.yaml`

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: customer-secret
  namespace: shopping
  labels:
    app: customer
type: Opaque
stringData:
  PSQL_CUSTOMER_DB_URL: "postgres://customersuser:changeme_customer_password@postgres.postgres.svc.cluster.local:5432/customersdb?sslmode=disable"
```

### Product Service Secret
**File**: `deployment/k8s/secrets/product-secret.yaml`

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: product-secret
  namespace: shopping
  labels:
    app: product
type: Opaque
stringData:
  PSQL_PRODUCT_DB_URL: "postgres://productsuser:changeme_product_password@postgres.postgres.svc.cluster.local:5432/productsdb?sslmode=disable"
  MINIO_ACCESS_KEY: "changeme_minio_access_key"
  MINIO_SECRET_KEY: "changeme_minio_secret_key"
```

### PostgreSQL Admin Secret
**File**: `deployment/k8s/secrets/postgres-secret.yaml`

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: postgres-secret
  namespace: postgres
  labels:
    app: postgres
type: Opaque
stringData:
  POSTGRES_PASSWORD: "changeme_postgres_admin"
```

### MinIO Credentials Secret
**File**: `deployment/k8s/secrets/minio-secret.yaml`

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: minio-credentials
  namespace: minio
  labels:
    app: minio
type: Opaque
stringData:
  MINIO_USER: "changeme_minio_root_user"
  MINIO_PASSWORD: "changeme_minio_root_password"
```

### Keycloak Secret
**File**: `deployment/k8s/secrets/keycloak-secret.yaml`

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: keycloak-secret
  namespace: keycloak
  labels:
    app: keycloak
type: Opaque
stringData:
  KC_BOOTSTRAP_ADMIN_USERNAME: "admin"
  KC_BOOTSTRAP_ADMIN_PASSWORD: "changeme_keycloak_admin"
  KC_DB_PASSWORD: "changeme_keycloak_db_password"
```

## Phase 3: Create .env.example Template

**File**: `.env.example`

```bash
# =============================================================================
# Shopping POC Environment Variables Template
# =============================================================================
# Copy this file to .env and update with your actual values
# DO NOT commit .env to version control

# =============================================================================
# PostgreSQL Database Configuration
# =============================================================================
# Customer database
PSQL_CUSTOMER_DB=customersdb
PSQL_CUSTOMER_ROLE=customersuser
PSQL_CUSTOMER_PASSWORD=your_customer_password_here
PSQL_CUSTOMER_DB_URL=postgres://$PSQL_CUSTOMER_ROLE:$PSQL_CUSTOMER_PASSWORD@localhost:30432/$PSQL_CUSTOMER_DB?sslmode=disable

# Product database
PSQL_PRODUCT_DB=productsdb
PSQL_PRODUCT_ROLE=productsuser
PSQL_PRODUCT_PASSWORD=your_product_password_here
PSQL_PRODUCT_DB_URL=postgres://$PSQL_PRODUCT_ROLE:$PSQL_PRODUCT_PASSWORD@localhost:30432/$PSQL_PRODUCT_DB?sslmode=disable

# Platform database settings (shared by all services)
DATABASE_MAX_OPEN_CONNS=25
DATABASE_MAX_IDLE_CONNS=25
DATABASE_CONN_MAX_LIFETIME=5m
DATABASE_CONN_MAX_IDLE_TIME=5m
DATABASE_CONNECT_TIMEOUT=30s
DATABASE_QUERY_TIMEOUT=60s
DATABASE_HEALTH_CHECK_INTERVAL=30s
DATABASE_HEALTH_CHECK_TIMEOUT=5s

# =============================================================================
# Kafka Configuration
# =============================================================================
KAFKA_BROKERS=kafka-clusterip.kafka.svc.cluster.local:9092

# Customer service Kafka settings
CUSTOMER_WRITE_TOPIC=CustomerEvents
CUSTOMER_READ_TOPICS=
CUSTOMER_GROUP=CustomerGroup

# EventReader service Kafka settings
EVENT_READER_WRITE_TOPIC=ReaderEvents
EVENT_READER_READ_TOPICS=CustomerEvents
EVENT_READER_GROUP=EventReaderGroup

# =============================================================================
# Outbox Pattern Configuration
# =============================================================================
OUTBOX_BATCH_SIZE=10
OUTBOX_DELETE_BATCH_SIZE=10
OUTBOX_PROCESS_INTERVAL=10s
OUTBOX_MAX_RETRIES=3

# Customer outbox interval
CUSTOMER_OUTBOX_PROCESSING_INTERVAL=10s

# =============================================================================
# MinIO Object Storage Configuration
# =============================================================================
# Platform MinIO settings
MINIO_ENDPOINT_KUBERNETES=minio.minio.svc.cluster.local:9000
MINIO_ENDPOINT_LOCAL=localhost:9000
MINIO_ACCESS_KEY=your_minio_access_key
MINIO_SECRET_KEY=your_minio_secret_key
MINIO_TLS_VERIFY=false

# Product service MinIO settings
MINIO_BUCKET=productimages

# =============================================================================
# CORS Configuration
# =============================================================================
CORS_ALLOWED_ORIGINS=http://localhost:4200,http://localhost:3000
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,PATCH,OPTIONS
CORS_ALLOWED_HEADERS=Content-Type,Authorization,X-Requested-With
CORS_ALLOW_CREDENTIALS=true
CORS_MAX_AGE=3600

# =============================================================================
# WebSocket Configuration
# =============================================================================
WEBSOCKET_URL=ws://localhost:8080/ws
WEBSOCKET_TIMEOUT=30s
WEBSOCKET_READ_BUFFER=1024
WEBSOCKET_WRITE_BUFFER=1024
WEBSOCKET_PORT=:8080
WEBSOCKET_PATH=/ws
WEBSOCKET_ALLOWED_ORIGINS=http://localhost:4200,http://localhost:3000

# =============================================================================
# Customer Service Configuration
# =============================================================================
CUSTOMER_SERVICE_PORT=:8080

# =============================================================================
# Product Service Configuration
# =============================================================================
PRODUCT_SERVICE_PORT=:8080
IMAGE_CACHE_DIR=/tmp/product-cache
IMAGE_CACHE_MAX_AGE=24h
IMAGE_CACHE_MAX_SIZE=0
CSV_BATCH_SIZE=100

# =============================================================================
# PostgreSQL Configuration (Platform Service)
# =============================================================================
POSTGRES_DB=postgresdb
POSTGRES_USER=postgresadmin
POSTGRES_PASSWORD=your_postgres_password

# =============================================================================
# Keycloak Configuration (Platform Service)
# =============================================================================
KEYCLOAK_ADMIN_USERNAME=admin
KEYCLOAK_ADMIN_PASSWORD=your_keycloak_admin_password
KEYCLOAK_DB_USERNAME=keycloak
KEYCLOAK_DB_PASSWORD=your_keycloak_db_password
KEYCLOAK_DB_URL_DATABASE=postgres://keycloak:keycloak@postgres.postgres.svc.cluster.local:5432/keycloak?sslmode=disable

# =============================================================================
# MinIO Configuration (Platform Service)
# =============================================================================
MINIO_ROOT_USER=your_minio_root_user
MINIO_ROOT_PASSWORD=your_minio_root_password
```

## Phase 4: Update Service Deployments

### Customer Service Deployment
**File**: `deployment/k8s/customer/customer-deploy.yaml` - ADD TO spec.template.containers[0].envFrom

```yaml
containers:
  - name: store-customer
    image: localhost:5000/go-shopping-poc/customer:1.0
    imagePullPolicy: Always
    envFrom:
      - configMapRef:
          name: customer-config
      - secretRef:
          name: customer-secret
    ports:
      - containerPort: 8080
    readinessProbe:
      httpGet:
        path: /health
        port: 8080
```

### Product Service Deployment
**File**: `deployment/k8s/product/product-deploy.yaml` - ADD TO spec.template.containers[0].envFrom AND spec.template.volumes[0]

```yaml
containers:
  - name: store-product
    image: localhost:5000/go-shopping-poc/product:1.0
    imagePullPolicy: Always
    envFrom:
      - configMapRef:
          name: product-config
      - secretRef:
          name: product-secret
    ports:
      - containerPort: 8080
    readinessProbe:
      httpGet:
        path: /health
        port: 8080
    volumeMounts:
      - name: product-cache
        mountPath: /tmp/product-cache
```

```yaml
volumes:
  - name: product-cache
    emptyDir: {}
```

### EventReader Service Deployment
**File**: `deployment/k8s/eventreader/eventreader-deploy.yaml` - ADD TO spec.template.containers[0].envFrom

```yaml
containers:
  - name: store-eventreader
    image: localhost:5000/go-shopping-poc/eventreader:1.0
    imagePullPolicy: Always
    envFrom:
      - configMapRef:
          name: eventreader-config
```

### WebSocket Service Deployment
**File**: `deployment/k8s/websocket/websocket-deployment.yaml` - ADD TO spec.template.containers[0].envFrom

```yaml
containers:
  - name: store-websocket
    image: localhost:5000/go-shopping-poc/websocket:1.0
    imagePullPolicy: Always
    envFrom:
      - configMapRef:
          name: websocket-config
```

## Phase 5: Update Platform Service Deployments

### PostgreSQL Deployment
**File**: `deployment/k8s/postgresql/postgresql-deploy.yaml` - REPLACE env section with envFrom

```yaml
containers:
  - name: postgres
    image: docker.io/postgres:18.0
    imagePullPolicy: IfNotPresent
    ports:
      - containerPort: 5432
        name: postgres
    envFrom:
      - configMapRef:
          name: postgresql-config
      - secretRef:
          name: postgres-secret
          optional: true
```

### MinIO Deployment
**File**: `deployment/k8s/minio/minio-deploy.yaml` - MODIFY spec.template.containers[0].envFrom (uncomment secretRef)

```yaml
containers:
  - name: minio
    image: minio/minio:latest
    imagePullPolicy: IfNotPresent
    args:
    - server
    - /data
    - --console-address=:9001
    envFrom:
      - secretRef:
          name: minio-credentials
```

### Keycloak Deployment
**File**: `deployment/k8s/keycloak/keycloak-deploy.yaml` - ADD TO spec.template.containers[0].envFrom

```yaml
containers:
  - name: keycloak
    image: quay.io/keycloak/keycloak:26.4.2
    args: ["start", "--import-realm"]
    envFrom:
      - configMapRef:
          name: keycloak-config
      - secretRef:
          name: keycloak-secret
          optional: true
```

## Phase 6: Create Docker Compose (Optional)

**File**: `docker-compose.yml` (NEW)

```yaml
version: '3.8'

services:
  # PostgreSQL Database
  postgres:
    image: postgres:18.0
    container_name: go-shopping-postgres
    environment:
      POSTGRES_DB: ${POSTGRES_DB:-postgresdb}
      POSTGRES_USER: ${POSTGRES_USER:-postgresadmin}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-postgresadminpw}
    ports:
      - "${POSTGRES_PORT:-30432}:5432"
    volumes:
      - postgres-data:/var/lib/postgresql/data
      - ./resources/postgresql/models:/docker-entrypoint-initdb.d:ro
    networks:
      - shopping-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-postgresadmin}"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Kafka
  kafka:
    image: apache/kafka:latest
    container_name: go-shopping-kafka
    environment:
      KAFKA_NODE_ID: "1"
      KAFKA_PROCESS_ROLES: "broker,controller"
      KAFKA_LISTENERS: "PLAINTEXT://0.0.0.0:9092,CONTROLLER://0.0.0.0:9093"
      KAFKA_ADVERTISED_LISTENERS: "PLAINTEXT://localhost:9092"
      KAFKA_CONTROLLER_LISTENER_NAMES: "CONTROLLER"
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: "CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT"
      KAFKA_CONTROLLER_QUORUM_VOTERS: "1@kafka:9093"
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: "1"
      KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR: "1"
      KAFKA_TRANSACTION_STATE_LOG_MIN_ISR: "1"
      KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS: "0"
      KAFKA_NUM_PARTITIONS: "3"
    ports:
      - "${KAFKA_PORT:-9092}:9092"
    networks:
      - shopping-network
    healthcheck:
      test: ["CMD", "kafka-topics.sh", "--bootstrap-server", "localhost:9092", "--list"]
      interval: 30s
      timeout: 10s
      retries: 5

  # MinIO
  minio:
    image: minio/minio:latest
    container_name: go-shopping-minio
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER:-minioadmin}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD:-minioadmin}
    command: server /data --console-address ":9001"
    ports:
      - "${MINIO_API_PORT:-9000}:9000"
      - "${MINIO_CONSOLE_PORT:-9001}:9001"
    volumes:
      - minio-data:/data
    networks:
      - shopping-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 30s
      timeout: 20s
      retries: 3

  # Keycloak
  keycloak:
    image: quay.io/keycloak/keycloak:26.4.2
    container_name: go-shopping-keycloak
    environment:
      KC_BOOTSTRAP_ADMIN_USERNAME: ${KC_BOOTSTRAP_ADMIN_USERNAME:-admin}
      KC_BOOTSTRAP_ADMIN_PASSWORD: ${KC_BOOTSTRAP_ADMIN_PASSWORD:-admin}
      KC_DB_URL_DATABASE: ${KC_DB_URL_DATABASE}
      KC_DB_USERNAME: ${KC_DB_USERNAME:-keycloak}
      KC_DB_PASSWORD: ${KC_DB_PASSWORD:-keycloak}
      KC_PROXY_HEADERS: "xforwarded"
      KC_HTTP_ENABLED: "true"
      KC_HOSTNAME_STRICT: "false"
      KC_HEALTH_ENABLED: "true"
      KC_CACHE: "ispn"
    command: start-dev
    ports:
      - "${KEYCLOAK_PORT:-8080}:8080"
    depends_on:
      - postgres
    networks:
      - shopping-network
    healthcheck:
      test: ["CMD-SHELL", "exec 3<>/dev/tcp/localhost/8080 && echo -e 'GET /health/ready HTTP/1.1\\r\\nHost: localhost\\r\\n\\r\\n' >&3"]
      interval: 30s
      timeout: 10s
      retries: 5

volumes:
  postgres-data:
    driver: local
  minio-data:
    driver: local

networks:
  shopping-network:
    driver: bridge
```

## Phase 7: Documentation

### Update README.md

**Add to README.md:**

```markdown
## Deployment

### Environment Variables

This project uses environment variables for all configuration. Copy `.env.example` to `.env` and update with your values.

```bash
cp .env.example .env
# Edit .env with your actual values
# DO NOT commit .env to version control
```

### Kubernetes Deployment

#### Quick Start

```bash
# 1. Create namespaces
kubectl create namespace shopping
kubectl create namespace kafka
kubectl create namespace postgres
kubectl create namespace minio
kubectl create namespace keycloak

# 2. Apply platform services
kubectl apply -f deployment/k8s/postgresql/
kubectl apply -f deployment/k8s/kafka/
kubectl apply -f deployment/k8s/minio/
kubectl apply -f deployment/k8s/keycloak/

# 3. Create and apply secrets
kubectl apply -f deployment/k8s/secrets/customer-secret.yaml
kubectl apply -f deployment/k8s/secrets/product-secret.yaml
kubectl apply -f deployment/k8s/secrets/postgres-secret.yaml
kubectl apply -f deployment/k8s/secrets/minio-secret.yaml
kubectl apply -f deployment/k8s/secrets/keycloak-secret.yaml

# 4. Apply application services
kubectl apply -f deployment/k8s/customer/
kubectl apply -f deployment/k8s/product/
kubectl apply -f deployment/k8s/eventreader/
kubectl apply -f deployment/k8s/websocket/
kubectl apply -f deployment/k8s/eventwriter/
```

#### Verify Deployment

```bash
# Check pods
kubectl get pods --all-namespaces

# Check services
kubectl get svc --all-namespaces

# Check ConfigMaps
kubectl get configmaps --all-namespaces

# Check Secrets
kubectl get secrets --all-namespaces

# View pod environment
kubectl describe pod -n shopping <pod-name>
```

#### Update Configuration

```bash
# Edit ConfigMap
kubectl edit configmap customer-config -n shopping

# Edit Secret
kubectl edit secret customer-secret -n shopping

# Restart pods to pick up changes
kubectl rollout restart deployment store-customer -n shopping
```

### Local Development with Docker Compose

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
```

### Create Documentation File

**File**: `docs/kubernetes-deployment-guide.md` (NEW)

```markdown
# Kubernetes Deployment Guide

## Overview

This guide explains how to deploy the Shopping POC using Kubernetes ConfigMaps and Secrets.

## Architecture

The project uses environment variables exclusively for configuration:
- **ConfigMaps**: Store non-sensitive configuration (ports, timeouts, topics)
- **Secrets**: Store sensitive data (passwords, database URLs, API keys)
- **Per-Service Structure**: Each service has its own ConfigMap in its deployment directory
- **Centralized Secrets**: Cross-service secrets in shared `secrets/` directory

## Prerequisites

- Kubernetes cluster (Rancher Desktop, Minikube, or cloud provider)
- kubectl configured to connect to your cluster
- Docker registry available (localhost:5000 for local development)

## Directory Structure

```
deployment/k8s/
├── customer/
│   ├── customer-deploy.yaml
│   └── customer-configmap.yaml
├── product/
│   ├── product-deploy.yaml
│   └── product-configmap.yaml
├── eventreader/
│   ├── eventreader-deploy.yaml
│   └── eventreader-configmap.yaml
├── websocket/
│   ├── websocket-deployment.yaml
│   └── websocket-configmap.yaml
├── eventwriter/
│   ├── eventwriter-deployment.yaml
│   └── eventwriter-configmap.yaml
├── postgresql/
│   ├── postgresql-deploy.yaml
│   └── postgresql-configmap.yaml
├── kafka/
│   └── kafka-deploy.yaml
├── minio/
│   └── minio-deploy.yaml
├── keycloak/
│   └── keycloak-deploy.yaml
└── secrets/
    ├── customer-secret.yaml
    ├── product-secret.yaml
    ├── postgres-secret.yaml
    ├── minio-secret.yaml
    └── keycloak-secret.yaml
```

## Deployment Steps

### 1. Create Namespaces

```bash
kubectl create namespace shopping
kubectl create namespace kafka
kubectl create namespace postgres
kubectl create namespace minio
kubectl create namespace keycloak
```

### 2. Apply Platform Services

```bash
# PostgreSQL
kubectl apply -f deployment/k8s/postgresql/

# Kafka
kubectl apply -f deployment/k8s/kafka/

# MinIO
kubectl apply -f deployment/k8s/minio/

# Keycloak
kubectl apply -f deployment/k8s/keycloak/
```

### 3. Apply Application Service Secrets

```bash
kubectl apply -f deployment/k8s/secrets/customer-secret.yaml
kubectl apply -f deployment/k8s/secrets/product-secret.yaml
kubectl apply -f deployment/k8s/secrets/postgres-secret.yaml
kubectl apply -f deployment/k8s/secrets/minio-secret.yaml
kubectl apply -f deployment/k8s/secrets/keycloak-secret.yaml
```

### 4. Apply Application Services

```bash
kubectl apply -f deployment/k8s/customer/
kubectl apply -f deployment/k8s/product/
kubectl apply -f deployment/k8s/eventreader/
kubectl apply -f deployment/k8s/websocket/
kubectl apply -f deployment/k8s/eventwriter/
```

### 5. Verify Deployment

```bash
# Check all pods
kubectl get pods --all-namespaces

# Check ConfigMaps
kubectl get configmaps --all-namespaces

# Check Secrets
kubectl get secrets --all-namespaces

# Check specific pod
kubectl describe pod -n shopping store-customer-0
```

## Updating Configuration

### Edit ConfigMaps

```bash
# Edit customer ConfigMap
kubectl edit configmap customer-config -n shopping

# Edit product ConfigMap
kubectl edit configmap product-config -n shopping

# Edit eventreader ConfigMap
kubectl edit configmap eventreader-config -n shopping

# Edit websocket ConfigMap
kubectl edit configmap websocket-config -n shopping
```

### Edit Secrets

```bash
# Edit customer Secret
kubectl edit secret customer-secret -n shopping

# Edit product Secret
kubectl edit secret product-secret -n shopping

# Edit postgres Secret
kubectl edit secret postgres-secret -n postgres

# Edit minio Secret
kubectl edit secret minio-credentials -n minio

# Edit keycloak Secret
kubectl edit secret keycloak-secret -n keycloak
```

### Restart Services

```bash
# Restart customer service
kubectl rollout restart deployment store-customer -n shopping

# Restart product service
kubectl rollout restart deployment store-product -n shopping

# Restart eventreader service
kubectl rollout restart statefulset store-eventreader -n shopping

# Restart websocket service
kubectl rollout restart deployment store-websocket -n shopping
```

## Troubleshooting

### ConfigMap Not Found

```bash
# Verify ConfigMap exists
kubectl get configmap customer-config -n shopping

# Check pod logs
kubectl logs -n shopping deploy/store-customer-0

# Common issues:
# - ConfigMap name mismatch (check deployment references ConfigMap name)
# - Namespace mismatch (ensure both in same namespace)
# - Environment variable name typo (check ConfigMap key names)
```

### Secret Not Found

```bash
# Verify Secret exists
kubectl get secret customer-secret -n shopping

# Check pod status
kubectl get pods -n shopping

# Common issues:
# - Secret name mismatch (check deployment references Secret name)
# - Namespace mismatch (ensure both in same namespace)
# - Base64 encoding errors (Secret values must be base64 encoded)
```

### Environment Variables Not Loaded

```bash
# Describe pod to see environment
kubectl describe pod -n shopping store-customer-0

# Search for specific variable
kubectl describe pod -n shopping store-customer-0 | grep PSQL_CUSTOMER_DB_URL

# Common issues:
# - Variable name mismatch (ConfigMap key name vs. environment variable name)
# - Variable not set in ConfigMap/Secret
# - Secret referenced but not mounted in envFrom
```

## Security Best Practices

1. **Never Commit Secrets**: Secret files should never be committed to version control
2. **Rotate Credentials**: Change passwords regularly and update Secrets
3. **Use Different Credentials**: Use different passwords for development, staging, production
4. **Limit Secret Access**: Use RBAC to limit which services can access which Secrets
5. **Encrypt Secrets**: Enable Kubernetes encryption at rest for production clusters
6. **Audit Access**: Regularly review Secret access logs
7. **Delete Unused Secrets**: Remove Secrets that are no longer needed
```

### Update AGENTS.md

Add to recent session summary:

```markdown
## Recent Session Summary (Today)

### Kubernetes ConfigMaps and Secrets Migration Plan Created ✅

**Problem Identified**: Environment-only configuration code changes completed, but Kubernetes deployment configuration missing (ConfigMaps, Secrets, Docker Compose, documentation).

**Plan Created**: Comprehensive 7-phase migration plan to complete environment-only configuration:
- **Phase 1**: Create service-specific ConfigMaps in per-service directories
- **Phase 2**: Create Secrets for sensitive data (database URLs, passwords)
- **Phase 3**: Create .env.example template for local development
- **Phase 4**: Update service deployments to use envFrom
- **Phase 5**: Update platform deployments (PostgreSQL, MinIO, Keycloak)
- **Phase 6**: Create docker-compose.yml for local development
- **Phase 7**: Update documentation with deployment instructions

**Implementation Scope**:
- **13 ConfigMap/Secret files** to create (7 ConfigMaps, 5 Secrets, 1 .env.example, 1 docker-compose.yml)
- **8 deployment files** to modify with envFrom configuration
- **2 documentation files** to create (README updates, deployment guide)

**Files to Create**:
- `deployment/k8s/customer/customer-configmap.yaml`
- `deployment/k8s/product/product-configmap.yaml`
- `deployment/k8s/eventreader/eventreader-configmap.yaml`
- `deployment/k8s/websocket/websocket-configmap.yaml`
- `deployment/k8s/eventwriter/eventwriter-configmap.yaml`
- `deployment/k8s/postgresql/postgresql-configmap.yaml`
- `deployment/k8s/keycloak/keycloak-configmap.yaml`
- `deployment/k8s/secrets/customer-secret.yaml`
- `deployment/k8s/secrets/product-secret.yaml`
- `deployment/k8s/secrets/postgres-secret.yaml`
- `deployment/k8s/secrets/minio-secret.yaml`
- `deployment/k8s/secrets/keycloak-secret.yaml`
- `.env.example` (repository root)
- `docker-compose.yml` (repository root, optional)
- `docs/kubernetes-deployment-guide.md` (new)

**Files to Modify**:
- `deployment/k8s/customer/customer-deploy.yaml`
- `deployment/k8s/product/product-deploy.yaml`
- `deployment/k8s/eventreader/eventreader-deploy.yaml`
- `deployment/k8s/websocket/websocket-deployment.yaml`
- `deployment/k8s/eventwriter/eventwriter-deployment.yaml`
- `deployment/k8s/postgresql/postgresql-deploy.yaml`
- `deployment/k8s/minio/minio-deploy.yaml`
- `deployment/k8s/keycloak/keycloak-deploy.yaml`
- `README.md`
- `AGENTS.md`

**Status**: ✅ Migration plan complete - ready for implementation

**Note**: This plan maintains current per-service directory structure while providing proper Kubernetes ConfigMaps and Secrets for environment-only configuration.
```