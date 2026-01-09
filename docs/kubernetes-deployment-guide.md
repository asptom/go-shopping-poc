# Kubernetes Deployment Guide

## Overview

This guide explains how to deploy Shopping POC using Kubernetes ConfigMaps and Secrets with simplified bash deployment scripts.

## Architecture

The project uses environment variables exclusively for configuration:
- **ConfigMaps**: Store non-sensitive configuration (ports, timeouts, topics)
- **Secrets**: Store sensitive data (passwords, database URLs, API keys)
- **Bash Scripts**: Simplified deployment without Kustomize complexity
- **envFrom Pattern**: Kubernetes pattern for injecting ConfigMaps/Secrets as environment variables

## Directory Structure

```
deployment/k8s/
├── base/                           # Centralized configuration
│   ├── configmaps/                 # All ConfigMaps
│   │   ├── platform-configmap.yaml  # Shared platform settings
│   │   ├── customer-configmap.yaml  # Service-specific
│   │   ├── product-configmap.yaml
│   │   ├── eventreader-configmap.yaml
│   │   ├── websocket-configmap.yaml
│   │   ├── eventwriter-configmap.yaml
│   │   ├── postgresql-configmap.yaml
│   │   ├── minio-configmap.yaml
│   │   └── keycloak-configmap.yaml
│   ├── secrets/                   # Secret templates + actual secrets
│   │   ├── customer-secret.yaml.example
│   │   ├── product-secret.yaml.example
│   │   ├── postgres-secret.yaml.example
│   │   ├── minio-secret.yaml.example
│   │   ├── keycloak-secret.yaml.example
│   │   └── *.yaml (actual secrets, don't commit)
│   └── services/kafka/namespace.yaml
├── customer/                        # Customer service deployment
│   └── customer-deploy.yaml
├── product/                         # Product service deployment
│   └── product-deploy.yaml
├── eventreader/                     # EventReader service deployment
│   └── eventreader-deploy.yaml
├── websocket/                       # WebSocket service deployment
│   ├── websocket-deployment.yaml
│   ├── websocket-ingress.yaml
│   └── websocket-service.yaml
├── eventwriter/                     # EventWriter service deployment
│   └── eventwriter-deployment.yaml
├── postgresql/                      # PostgreSQL deployment
│   └── postgresql-deploy.yaml
├── kafka/                           # Kafka deployment
│   └── kafka-deploy.yaml
├── minio/                           # MinIO deployment
│   └── minio-deploy.yaml
├── keycloak/                        # Keycloak deployment
│   └── keycloak-deploy.yaml
├── deploy-dev.sh                   # Development deployment script
└── deploy-prod.sh                  # Production deployment script
```

## Deployment Steps

### 1. Prepare Secrets

**Why**: Secrets contain sensitive data (passwords, database URLs) and should never be committed.

```bash
# Navigate to secrets directory
cd deployment/k8s/base/secret/

# Copy example templates
cp customer-secret.yaml.example customer-secret.yaml
cp product-secret.yaml.example product-secret.yaml
cp postgres-secret.yaml.example postgres-secret.yaml
cp minio-secret.yaml.example minio-secret.yaml
cp keycloak-secret.yaml.example keycloak-secret.yaml

# Edit each file and replace CHANGE_ME_* with actual values
vim customer-secret.yaml  # or your preferred editor
```

**Important**: `.yaml.example` files are safe to commit. Actual secret files (without `.example`) should NOT be committed.

### 2. Deploy to Development

```bash
cd deployments/kubernetes

# Run development deployment
./deploy-dev.sh
```

**What the script does:**
1. **Secret Setup**: Checks if secret files exist, copies from .example if not (interactive)
2. **ConfigMaps**: Applies all 9 ConfigMaps
3. **Secrets**: Applies all 5 secrets
4. **Namespaces**: Creates Kafka namespace
5. **Platform Services**: Deploys PostgreSQL, Kafka, MinIO, Keycloak
6. **Application Services**: Deploys customer, product, eventreader, websocket, eventwriter
7. **Verification**: Shows pod, ConfigMap, and Secret status

### 3. Deploy to Production

```bash
cd deployments/kubernetes

# Edit secrets with PRODUCTION values first!
# IMPORTANT: Make sure secrets contain production passwords/URLs

# Run production deployment
./deploy-prod.sh
```

**What the script does:**
1. **ConfigMaps**: Applies all 9 ConfigMaps (same as dev)
2. **Secrets**: Applies production secrets
3. **Platform Services**: Deploys PostgreSQL, Kafka, MinIO, Keycloak
4. **Application Services**: Deploys customer, product, eventreader, websocket, eventwriter
5. **Production Replicas**: Sets 2 replicas for customer and product services
6. **Production Labels**: Adds `env=production` label to all pocstore namespace services
7. **Verification**: Shows deployment status

### 4. Verify Deployment

```bash
# Check all pods
kubectl get pods --all-namespaces

# Expected output (dev):
# customer-0       1/1     Running   ...
# product-0        1/1     Running   ...
# eventreader-0    1/1     Running   ...
# postgres-0       1/1     Running   ...
# kafka-0          1/1     Running   ...
# ...

# Check ConfigMaps
kubectl get configmaps --all-namespaces

# Check Secrets (names only, values are hidden)
kubectl get secrets --all-namespaces

# View pod environment
kubectl describe pod -n pocstore store-customer-0 | grep -A 20 "Environment"

# View specific environment variable
kubectl describe pod -n pocstore store-customer-0 | grep PSQL_CUSTOMER_DB_URL
```

## Configuration Management

### View Configuration

```bash
# View ConfigMap
kubectl get configmap platform-config -n pocstore -o yaml

# View Secret (decoded)
kubectl get secret customer-secret -n pocstore -o jsonpath='{.data.PSQL_CUSTOMER_DB_URL}' | base64 -d
```

### Update Configuration

```bash
# Edit ConfigMap directly
kubectl edit configmap platform-config -n pocstore

# Edit Secret
kubectl edit secret customer-secret -n pocstore

# Restart pods to apply changes
kubectl rollout restart statefulset/store-customer -n pocstore
```

### Rolling Update vs Full Restart

- **ConfigMap changes**: Pods don't automatically reload (need restart)
- **Secret changes**: Pods don't automatically reload (need restart)
- **Deployment changes**: Automatic rolling update with `kubectl apply`

## Troubleshooting

### Secret File Not Found

```bash
# Verify secret file exists
ls deployment/k8s/base/secret/customer-secret.yaml

# Create from template if missing
cd deployment/k8s/base/secret/
cp customer-secret.yaml.example customer-secret.yaml
```

### ConfigMap Not Found

```bash
# Verify ConfigMap exists
kubectl get configmap customer-config -n pocstore

# Check pod logs
kubectl logs -n pocstore store-customer-0

# Common issues:
# - ConfigMap name mismatch (check deployment references ConfigMap name)
# - Namespace mismatch (ensure both in same namespace)
# - Environment variable name typo (check ConfigMap key names)
```

### Secret Not Found

```bash
# Verify Secret exists
kubectl get secret customer-secret -n pocstore

# Check pod status
kubectl get pods -n pocstore

# Common issues:
# - Secret name mismatch (check deployment references Secret name)
# - Namespace mismatch (ensure both in same namespace)
# - Secret not created from .example template
```

### Environment Variables Not Loaded

```bash
# Describe pod to see environment
kubectl describe pod -n pocstore store-customer-0

# Search for specific variable
kubectl describe pod -n pocstore store-customer-0 | grep KAFKA_BROKERS

# Common issues:
# - Variable name mismatch (ConfigMap key vs environment variable name)
# - Variable not set in ConfigMap/Secret
# - Secret referenced but not mounted in envFrom
```

### Pods Not Starting

```bash
# Check pod status
kubectl get pods -n pocstore

# View pod events
kubectl describe pod -n pocstore store-customer-0

# Check pod logs
kubectl logs -n pocstore store-customer-0 --previous

# Common issues:
# - Secret not found (check secret exists)
# - ConfigMap not found (check ConfigMap exists)
# - Image pull error (check image name/tag)
# - Resource limits (check pod resource requests)
```

## Security Best Practices

1. **Never Commit Secrets**: Secret files should never be committed to version control
2. **Use Different Credentials**: Use different passwords for development, staging, production
3. **Limit Secret Access**: Use RBAC to limit which services can access which Secrets (future enhancement)
4. **Encrypt Secrets**: Enable Kubernetes encryption at rest for production clusters (future enhancement)
5. **Rotate Credentials**: Change passwords regularly and update Secrets
6. **Delete Unused Secrets**: Remove Secrets that are no longer needed
7. **Check .gitignore**: Ensure `*.yaml` (without `.example`) is ignored in secrets directory

## Configuration Reference

### Platform ConfigMap

**File**: `deployment/k8s/base/configmap/platform-configmap.yaml`

| Key | Description | Example |
|-----|-------------|---------|
| DATABASE_MAX_OPEN_CONNS | Maximum open database connections | 25 |
| DATABASE_MAX_IDLE_CONNS | Maximum idle connections | 25 |
| DATABASE_CONN_MAX_LIFETIME | Maximum connection lifetime | 5m |
| DATABASE_CONN_MAX_IDLE_TIME | Maximum idle time | 5m |
| DATABASE_CONNECT_TIMEOUT | Connection establishment timeout | 30s |
| DATABASE_QUERY_TIMEOUT | Query execution timeout | 60s |
| DATABASE_HEALTH_CHECK_INTERVAL | Health check interval | 30s |
| DATABASE_HEALTH_CHECK_TIMEOUT | Health check timeout | 5s |
| KAFKA_BROKERS | Kafka broker addresses | kafka:9092 |
| OUTBOX_BATCH_SIZE | Outbox batch size | 10 |
| OUTBOX_DELETE_BATCH_SIZE | Outbox delete batch size | 10 |
| OUTBOX_PROCESS_INTERVAL | Outbox processing interval | 10s |
| OUTBOX_MAX_RETRIES | Maximum retry attempts | 3 |
| CORS_ALLOWED_ORIGINS | Allowed CORS origins | http://localhost:4200,http://localhost:3000 |
| CORS_ALLOWED_METHODS | Allowed CORS methods | GET,POST,PUT,DELETE,PATCH,OPTIONS |
| CORS_ALLOWED_HEADERS | Allowed CORS headers | Content-Type,Authorization,X-Requested-With |
| CORS_ALLOW_CREDENTIALS | Allow credentials | true |
| CORS_MAX_AGE | CORS max age (seconds) | 3600 |
| MINIO_ENDPOINT_KUBERNETES | MinIO endpoint in Kubernetes | minio.minio.svc.cluster.local:9000 |
| MINIO_TLS_VERIFY | Verify TLS certificate | false |

**Used By**: customer, product, eventreader, eventwriter services

### Service ConfigMaps

#### Customer Config
**File**: `deployment/k8s/base/configmap/customer-configmap.yaml`

| Key | Description | Example |
|-----|-------------|---------|
| PSQL_CUSTOMER_DB | Customer database name | customersdb |
| PSQL_CUSTOMER_ROLE | Customer database role | customersuser |
| CUSTOMER_SERVICE_PORT | Service port | :8080 |
| CUSTOMER_WRITE_TOPIC | Kafka write topic | CustomerEvents |
| CUSTOMER_READ_TOPICS | Kafka read topics (comma-separated) | "" |
| CUSTOMER_GROUP | Kafka consumer group | CustomerGroup |
| CUSTOMER_OUTBOX_PROCESSING_INTERVAL | Outbox processing interval | 10s |

#### Product Config
**File**: `deployment/k8s/base/configmap/product-configmap.yaml`

| Key | Description | Example |
|-----|-------------|---------|
| PSQL_PRODUCT_DB | Product database name | productsdb |
| PSQL_PRODUCT_ROLE | Product database role | productsuser |
| PRODUCT_SERVICE_PORT | Service port | :8080 |
| IMAGE_CACHE_DIR | Image cache directory | /tmp/product-cache |
| IMAGE_CACHE_MAX_AGE | Cache max age | 24h |
| IMAGE_CACHE_MAX_SIZE | Cache max size (0 = unlimited) | 0 |
| CSV_BATCH_SIZE | CSV batch processing size | 100 |
| MINIO_BUCKET | MinIO bucket for images | productimages |
| PRODUCT_WRITE_TOPIC | Kafka write topic | ProductEvents |
| PRODUCT_READ_TOPICS | Kafka read topics (comma-separated) | "" |
| PRODUCT_GROUP | Kafka consumer group | ProductGroup |

#### EventReader Config
**File**: `deployment/k8s/base/configmap/eventreader-configmap.yaml`

| Key | Description | Example |
|-----|-------------|---------|
| EVENT_READER_WRITE_TOPIC | Kafka write topic | ReaderEvents |
| EVENT_READER_READ_TOPICS | Kafka read topics (comma-separated) | CustomerEvents |
| EVENT_READER_GROUP | Kafka consumer group | EventReaderGroup |

#### WebSocket Config
**File**: `deployment/k8s/base/configmap/websocket-configmap.yaml`

| Key | Description | Example |
|-----|-------------|---------|
| WEBSOCKET_URL | WebSocket URL | ws://localhost:8080/ws |
| WEBSOCKET_TIMEOUT | WebSocket timeout | 30s |
| WEBSOCKET_READ_BUFFER | Read buffer size | 1024 |
| WEBSOCKET_WRITE_BUFFER | Write buffer size | 1024 |
| WEBSOCKET_PORT | WebSocket port | :8080 |
| WEBSOCKET_PATH | WebSocket path | /ws |
| WEBSOCKET_ALLOWED_ORIGINS | Allowed origins | http://localhost:4200,http://localhost:3000 |

#### EventWriter Config
**File**: `deployment/k8s/base/configmap/eventwriter-configmap.yaml`

| Key | Description | Example |
|-----|-------------|---------|
| EVENT_WRITER_WRITE_TOPIC | Kafka write topic | WriterEvents |
| EVENT_WRITER_GROUP | Kafka consumer group | EventWriterGroup |

### Secrets Reference

| Secret | Namespace | Keys |
|--------|-----------|-------|
| customer-secret | pocstore | PSQL_CUSTOMER_DB_URL |
| product-secret | pocstore | PSQL_PRODUCT_DB_URL, MINIO_ACCESS_KEY, MINIO_SECRET_KEY |
| postgres-secret | postgres | POSTGRES_PASSWORD |
| minio-credentials | minio | MINIO_USER, MINIO_PASSWORD |
| keycloak-secret | keycloak | KC_BOOTSTRAP_ADMIN_USERNAME, KC_BOOTSTRAP_ADMIN_PASSWORD, KC_DB_PASSWORD |

## Adding New Services

### Step 1: Create Service ConfigMap

```bash
# Create new ConfigMap
cat > deployment/k8s/base/configmap/newservice-configmap.yaml << EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: newservice-config
  namespace: pocstore
  labels:
    app: go-shopping-poc
    component: newservice
data:
  NEWSERVICE_PORT: ":8080"
  NEWSERVICE_TOPIC: "NewServiceEvents"
  NEWSERVICE_GROUP: "NewServiceGroup"
EOF
```

### Step 2: Create Secret Template

```bash
# Create secret template
cat > deployment/k8s/base/secret/newservice-secret.yaml.example << EOF
apiVersion: v1
kind: Secret
metadata:
  name: newservice-secret
  namespace: pocstore
  labels:
    app: go-shopping-poc
    component: newservice
type: Opaque
stringData:
  NEWSERVICE_API_KEY: "CHANGE_ME_API_KEY"
  NEWSERVICE_DB_URL: "CHANGE_ME_DB_URL"
EOF
```

### Step 3: Update Deployment

```yaml
# In your deployment file, add envFrom to container spec
containers:
  - name: newservice
    image: localhost:5000/go-shopping-poc/newservice:1.0
    envFrom:
      - configMapRef:
          name: platform-config
      - configMapRef:
          name: newservice-config
      - secretRef:
          name: newservice-secret
```

### Step 4: Update Deployment Scripts

```bash
# Add to deploy-dev.sh after other ConfigMaps:
#   kubectl apply -f base/configmaps/newservice-configmap.yaml

# Add to deploy-dev.sh after other Secrets:
#   kubectl apply -f base/secrets/newservice-secret.yaml

# Add to deploy-dev.sh after other application services:
#   kubectl apply -f newservice/newservice-deploy.yaml

# Add similar entries to deploy-prod.sh
```

### Step 5: Test Deployment

```bash
cd deployments/kubernetes
./deploy-dev.sh

# Verify service is running
kubectl get pods -n pocstore | grep newservice
```

## Production Considerations

### When Ready for Production

1. **External Secrets Operator**: Consider integrating External Secrets Operator to sync secrets from AWS Secrets Manager, HashiCorp Vault, or similar
2. **RBAC Policies**: Implement Role and RoleBinding to restrict service access to specific secrets
3. **Encryption at Rest**: Enable Kubernetes secret encryption in etcd
4. **Secret Rotation**: Implement automated secret rotation policies
5. **Monitoring**: Add Prometheus metrics for configuration changes and deployment status
6. **CI/CD Integration**: Integrate deployment scripts into CI/CD pipeline
7. **GitOps**: Consider ArgoCD or Flux for Git-based deployments (can replace bash scripts)

## Advanced Topics

### Environment Variable Expansion

The platform ConfigMap uses environment variable expansion for database URLs:

```yaml
# In ConfigMap:
PSQL_CUSTOMER_DB_URL: "postgres://$PSQL_CUSTOMER_ROLE:$PSQL_CUSTOMER_PASSWORD@postgres:5432/customersdb"

# When loaded, becomes:
# postgres://customersuser:password@postgres:5432/customersdb
```

This allows defining role, password, database once and reusing them.

### Replica Management

**Development**: 1 replica (default in deployment files)
**Production**: 2 replicas (set by deploy-prod.sh)

To manually scale:
```bash
kubectl scale statefulset/store-customer -n pocstore --replicas=3
kubectl scale statefulset/store-product -n pocstore --replicas=3
```

### Rolling Updates

When ConfigMaps or Secrets change, restart pods:

```bash
# Restart customer pods
kubectl rollout restart statefulset/store-customer -n pocstore

# Wait for rollout to complete
kubectl rollout status statefulset/store-customer -n pocstore
```

### Namespace Management

The project uses multiple namespaces for separation:

- **pocstore**: Application services (customer, product, eventreader, websocket, eventwriter)
- **postgres**: PostgreSQL database
- **kafka**: Kafka message broker
- **minio**: MinIO object storage
- **keycloak**: Keycloak authentication

```bash
# Create all namespaces
kubectl create namespace pocstore
kubectl create namespace postgres
kubectl create namespace kafka
kubectl create namespace minio
kubectl create namespace keycloak

# List all namespaces
kubectl get namespaces
```

## Learning Checklist

- [ ] Understand difference between ConfigMaps and Secrets
- [ ] Know how to create secrets from .yaml.example templates
- [ ] Can deploy to development using `./deploy-dev.sh`
- [ ] Can deploy to production using `./deploy-prod.sh`
- [ ] Can update configuration and restart pods
- [ ] Can troubleshoot ConfigMap/Secret issues
- [ ] Understand platform vs service configuration separation
- [ ] Can add a new service with ConfigMap and Secret
- [ ] Understand envFrom pattern
- [ ] Can verify deployment status

## Additional Resources

- [Kubernetes ConfigMaps Documentation](https://kubernetes.io/docs/concepts/configuration/configmap/)
- [Kubernetes Secrets Documentation](https://kubernetes.io/docs/concepts/configuration/secret/)
- [kubectl Cheat Sheet](https://kubernetes.io/docs/reference/kubectl/cheatsheet/)
- [Clean Architecture in Kubernetes](https://kubernetes.io/docs/concepts/architecture/)

## Appendix: Deployment Script Reference

### deploy-dev.sh

**Purpose**: Development deployment script
**Replicas**: 1 (uses deployment defaults)
**Labels**: None (development environment)
**Interactive**: Pauses to let users edit secrets
**Output**: Step-by-step deployment with verification

**Script Flow**:
1. Check/create secret files from .yaml.example templates (interactive)
2. Apply all 9 ConfigMaps from base/configmaps/
3. Apply all 5 Secrets from base/secrets/
4. Apply Kafka namespace
5. Apply platform services (PostgreSQL, Kafka, MinIO, Keycloak)
6. Apply application services (customer, product, eventreader, websocket, eventwriter)
7. Verify deployment (show pods, ConfigMaps, Secrets)

### deploy-prod.sh

**Purpose**: Production deployment script
**Replicas**: 2 for customer, 2 for product
**Labels**: `env=production` on pocstore namespace services
**Warnings**: Reminds about production values
**Output**: Step-by-step deployment with verification

**Script Flow**:
1. Apply all 9 ConfigMaps from base/configmaps/
2. Apply all 5 Secrets from base/secrets/ (warns to check production values)
3. Apply Kafka namespace
4. Apply platform services (PostgreSQL, Kafka, MinIO, Keycloak)
5. Apply application services (customer, product, eventreader, websocket, eventwriter)
6. Scale to 2 replicas for customer and product services
7. Add `env=production` label to all pocstore namespace services
8. Verify deployment (show pods, ConfigMaps, Secrets)

### Common kubectl Commands

```bash
# Get resources
kubectl get pods -n pocstore
kubectl get configmaps -n pocstore
kubectl get secrets -n pocstore
kubectl get statefulsets -n pocstore
kubectl get deployments -n pocstore

# Describe resources
kubectl describe pod -n pocstore store-customer-0
kubectl describe configmap platform-config -n pocstore
kubectl describe secret customer-secret -n pocstore

# Edit resources
kubectl edit configmap platform-config -n pocstore
kubectl edit secret customer-secret -n pocstore
kubectl edit statefulset store-customer -n pocstore

# Scale resources
kubectl scale statefulset/store-customer -n pocstore --replicas=3

# Restart resources
kubectl rollout restart statefulset/store-customer -n pocstore

# View logs
kubectl logs -n pocstore store-customer-0
kubectl logs -n pocstore store-customer-0 --previous

# Exec into pod
kubectl exec -it -n pocstore store-customer-0 -- sh

# Port forward
kubectl port-forward -n pocstore store-customer-0 8080:8080
```
