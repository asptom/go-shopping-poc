# Product Service Refactoring Plan - Refined

**Document Version**: 1.2
**Created**: January 22, 2026
**Updated**: January 22, 2026
**Status**: Refined - Ready for Implementation
**Estimated Effort**: 10-14 hours of development time

---

## Executive Summary

This document outlines a comprehensive refactoring of product service with detailed architectural decisions for API nomenclature, authentication, configuration optimization, and frontend integration. The refactoring will be performed incrementally with side-by-side deployment to ensure no disruption.

### Key Objectives
1. **Reduce Code Bloat**: Split monolithic files into focused, maintainable units
2. **Optimize Dependencies**: Services only load components they actually need
3. **Separate Concerns**: Public catalog (read-only) vs admin operations (write + auth)
4. **Add Authentication**: Keycloak-based JWT authentication for admin operations
5. **API Standardization**: Establish consistent API nomenclature for future services
6. **Configuration Refactoring**: Implement selective configuration loading across all services
7. **Event-Driven**: Catalog publishes view events for marketing/analysis
8. **Zero Downtime**: Side-by-side deployment with validation period

---

## Section 1: Keycloak Configuration (FIXED)

### 1.1 Realm Information
- **Realm Name**: `pocstore-realm`
- **Namespace**: `keycloak` (NOT `minio` - corrected from v1.0 plan)
- **JWKS Endpoint**: `http://keycloak.keycloak.svc.cluster.local:8080/realms/pocstore-realm/protocol/openid-connect/certs`

### 1.2 Required Keycloak Configuration

#### A. Product-Admin Client (Machine-to-Machine Authentication)
**Purpose**: Allow product-admin service to authenticate itself (service account) without user interaction

**Configuration Steps**:
1. Create Confidential Client in Keycloak Admin Console:
   ```
   Client ID: product-admin
   Client Type: confidential
   Access Type: service account
   Standard Flow Enabled: OFF
   Direct Access Grants: ON
   Service Accounts Enabled: ON
   Authorization Enabled: OFF
   ```

2. Configure Client Permissions:
   - Client Authenticator: client-secret
   - Valid Redirect URIs: (leave empty for service account)
   - Web Origins: (leave empty for service account)

3. Generate Service Account Secret:
   - Navigate to Service Accounts tab
   - Select `product-admin` client
   - Generate a new client secret
   - Save secret for Kubernetes Secret

#### B. Product-Admin Role
**Purpose**: Define administrative capabilities for product management

**Configuration Steps**:
1. Create Role in Keycloak Admin Console:
   ```
   Name: product-admin
   Description: Administrative access to product management operations
   Composite: No
   Client Role: No
   ```

2. Add Realm Roles to `product-admin`:
   - Assign necessary permissions (future: create separate fine-grained roles)
   - For now: use this single role for all admin operations

3. Assign Role to Service Account:
   - Navigate to Service Accounts → product-admin
   - Role Mappings → Assign Realm Roles
   - Select `product-admin` role
   - Effective: `product-admin` service account will have `product-admin` role

#### C. Frontend Client (End User Authentication)
**Purpose**: Allow end users to authenticate on website

**Configuration Steps**:
1. Ensure existing frontend client exists (from `pocstore-realm`):
   ```
   Client ID: go-shopping-poc-frontend (or similar)
   Client Type: public
   Access Type: public
   Standard Flow Enabled: ON
   Implicit Flow Enabled: OFF
   Direct Access Grants: OFF
   ```

2. Create/Verify Frontend Roles:
   - `customer`: Basic customer access
   - `order-placer`: Can place orders
   - `reviewer`: Can write reviews
   (Note: These roles would be needed for frontend authorization on future admin endpoints)

### 1.3 Kubernetes Secret Configuration

**Create Secret for Product-Admin Service**:
```bash
kubectl -n shopping create secret generic product-admin-keycloak-secret \
  --from-literal=KEYCLOAK_CLIENT_ID=product-admin \
  --from-literal=KEYCLOAK_CLIENT_SECRET=<generated-secret>
```

**Platform ConfigMap Updates**:
Add to `deploy/k8s/platform/config/platform-configmap-for-services.yaml`:
```yaml
data:
  # Keycloak Configuration (shared by authenticated services)
  KEYCLOAK_ISSUER: "http://keycloak.keycloak.svc.cluster.local:8080/realms/pocstore-realm"
  KEYCLOAK_JWKS_URL: "http://keycloak.keycloak.svc.cluster.local:8080/realms/pocstore-realm/protocol/openid-connect/certs"
```

### 1.4 Authentication Flow

#### Admin Service (Product-Admin)
```
1. Service startup: Load KEYCLOAK_CLIENT_SECRET from Kubernetes secret
2. On each request: Validate JWT from Authorization header using JWKS endpoint
3. Extract roles from token
4. Verify `product-admin` role is present
5. Allow or deny request based on role check
```

#### Frontend (End User)
```
1. User logs in via frontend
2. Frontend authenticates with Keycloak using implicit or authorization code flow
3. Keycloak returns JWT access token
4. Frontend stores token and includes in Authorization header for admin requests
```

---

## Section 2: API Nomenclature Decision

### 2.1 Recommended Approach: `/api/v1/` with Path-Based Separation

**Rationale**:
- **Versioning Support**: Easy to introduce `/api/v2/` in future
- **Service Separation**: Clear separation between catalog and admin concerns
- **Single Domain**: No need for multiple domains or certificates
- **Traefik Compatibility**: Works with existing ingress setup
- **RESTful Pattern**: Follows industry best practices (Shopify, Stripe, GitHub all use `/api/`)

### 2.2 Proposed API Structure

```
Catalog (Public, No Auth):
  GET    /api/v1/products                          # List all products (paginated)
  GET    /api/v1/products/{id}                   # Get single product
  GET    /api/v1/products/search                  # Search endpoint
  GET    /api/v1/products/search?q=shirt&limit=50&offset=0
  GET    /api/v1/products/category/{category}      # Get by category
  GET    /api/v1/products/brand/{brand}          # Get by brand
  GET    /api/v1/products/in-stock                # Get in-stock products
  GET    /health                                  # Health check (public)

Admin (Protected, Requires `product-admin` role):
  GET    /health                                  # Health check (public)
  POST   /api/v1/admin/products                    # Create product
  PUT    /api/v1/admin/products/{id}               # Update product
  DELETE /api/v1/admin/products/{id}               # Delete product
  POST   /api/v1/admin/products/{id}/images     # Add image
  PUT    /api/v1/admin/products/images/{id}        # Update image
  DELETE /api/v1/admin/products/images/{id}        # Delete image
  PUT    /api/v1/admin/products/{id}/main-image/{imgId}  # Set main image
```

### 2.3 Future Multi-Tenant Support

**Design for Future**:
```
/api/v1/tenants/{tenant-id}/products           # Per-tenant products
/api/v1/tenants/{tenant-id}/admin/products     # Per-tenant admin
```

**Current**: Single tenant (`pocstore`), so tenant ID is implicit.

### 2.4 Ingress Configuration

**Update `deploy/k8s/service/product-catalog/product-catalog-ingress.yaml`**:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: product-catalog-ingress
  namespace: shopping
spec:
  rules:
    - host: pocstore.local
      http:
        paths:
          - path: /api/v1/products
            pathType: Prefix
            backend:
              service:
                name: product-catalog-svc
                port:
                  number: 80
```

**Update `deploy/k8s/service/product-admin/product-admin-ingress.yaml`**:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: product-admin-ingress
  namespace: shopping
spec:
  rules:
    - host: pocstore.local
      http:
        paths:
          - path: /api/v1/admin/products
            pathType: Prefix
            backend:
              service:
                name: product-admin-svc
                port:
                  number: 80
```

**Note**: Traefik will route `/api/v1/` paths to appropriate services based on path prefix.

---

## Section 3: Configuration Refactoring (BACKWARD COMPATIBLE)

### 3.1 Current Problem Analysis

**Current Approach**:
```go
// config.LoadConfig[T](serviceName) loads ALL environment variables
// Kubernetes uses envFrom: to load ALL ConfigMaps/Secrets
```

**Current Kubernetes Deployment Pattern**:
```yaml
containers:
  - name: customer
    envFrom:
      - configMapRef:
          name: platform-config      # Loads ALL platform vars
      - configMapRef:
          name: customer-config      # Loads ALL customer vars
      - secretRef:
          name: customer-db-secret  # Loads ALL database secrets
```

**Problems**:
1. Services have access to environment variables they don't need
2. Services can accidentally use wrong environment variables (e.g., customer service using `PRODUCT_WRITE_TOPIC`)
3. Hard to audit what each service actually needs
4. Security risk: MinIO credentials loaded into all services

### 3.2 Recommended Approach: Service-Specific ConfigMaps + Explicit env Declarations

**Goal**: Zero breaking changes to existing services (customer, eventreader, product-loader) while enabling selective loading.

**Approach**:
1. Keep existing `config.LoadConfig[T]()` - NO CODE CHANGES
2. Change Kubernetes deployments to use explicit `env:` declarations instead of `envFrom:`
3. Only load environment variables each service actually needs
4. Maintain backward compatibility by keeping same variable names

### 3.3 Proposed Configuration Changes

#### Step 1: Update Kubernetes Deployments (NO CODE CHANGES)

**Customer Service** (`deploy/k8s/service/customer/customer-deploy.yaml`):
```yaml
containers:
  - name: customer
    env:
      # Platform config (selective)
      - name: LOG_LEVEL
        valueFrom:
          configMapKeyRef:
            name: platform-config
            key: LOG_LEVEL
      
      # Service config (selective)
      - name: CUSTOMER_SERVICE_PORT
        valueFrom:
          configMapKeyRef:
            name: customer-config
            key: CUSTOMER_SERVICE_PORT
      - name: CUSTOMER_WRITE_TOPIC
        valueFrom:
          configMapKeyRef:
            name: customer-config
            key: CUSTOMER_WRITE_TOPIC
      - name: CUSTOMER_GROUP
        valueFrom:
          configMapKeyRef:
            name: customer-config
            key: CUSTOMER_GROUP
      
      # Secrets (selective)
      - name: DB_URL
        valueFrom:
          secretKeyRef:
            name: customer-db-secret
            key: DB_URL
```

**Before** (envFrom approach): Customer loads 42 env vars (entire platform-config + customer-config + secrets)
**After** (explicit env approach): Customer loads 5 env vars (only what it needs)

**Product-Catalog Service** (`deploy/k8s/service/product-catalog/product-catalog-deploy.yaml`):
```yaml
containers:
  - name: product-catalog
    env:
      # Platform config (minimal - only database pool settings)
      - name: DATABASE_MAX_OPEN_CONNS
        valueFrom:
          configMapKeyRef:
            name: platform-config
            key: DATABASE_MAX_OPEN_CONNS
      - name: DATABASE_MAX_IDLE_CONNS
        valueFrom:
          configMapKeyRef:
            name: platform-config
            key: DATABASE_MAX_IDLE_CONNS
      - name: DATABASE_CONN_MAX_LIFETIME
        valueFrom:
          configMapKeyRef:
            name: platform-config
            key: DATABASE_CONN_MAX_LIFETIME
      - name: DATABASE_CONN_MAX_IDLE_TIME
        valueFrom:
          configMapKeyRef:
            name: platform-config
            key: DATABASE_CONN_MAX_IDLE_TIME
      - name: DATABASE_CONNECT_TIMEOUT
        valueFrom:
          configMapKeyRef:
            name: platform-config
            key: DATABASE_CONNECT_TIMEOUT
      - name: DATABASE_QUERY_TIMEOUT
        valueFrom:
          configMapKeyRef:
            name: platform-config
            key: DATABASE_QUERY_TIMEOUT
      
      # Service config
      - name: PRODUCT_CATALOG_PORT
        valueFrom:
          configMapKeyRef:
            name: product-catalog-config
            key: PRODUCT_CATALOG_PORT
      
      # Secret
      - name: DB_URL
        valueFrom:
          secretKeyRef:
            name: product-catalog-db-secret
            key: DB_URL
```

**Product-Admin Service** (`deploy/k8s/service/product-admin/product-admin-deploy.yaml`):
```yaml
containers:
  - name: product-admin
    env:
      # Platform config (selective)
      - name: DATABASE_MAX_OPEN_CONNS
        valueFrom:
          configMapKeyRef:
            name: platform-config
            key: DATABASE_MAX_OPEN_CONNS
      - name: DATABASE_MAX_IDLE_CONNS
        valueFrom:
          configMapKeyRef:
            name: platform-config
            key: DATABASE_MAX_IDLE_CONNS
      - name: DATABASE_CONN_MAX_LIFETIME
        valueFrom:
          configMapKeyRef:
            name: platform-config
            key: DATABASE_CONN_MAX_LIFETIME
      - name: DATABASE_CONNECT_TIMEOUT
        valueFrom:
          configMapKeyRef:
            name: platform-config
            key: DATABASE_CONNECT_TIMEOUT
      - name: CORS_ALLOWED_ORIGINS
        valueFrom:
          configMapKeyRef:
            name: platform-config
            key: CORS_ALLOWED_ORIGINS
      - name: CORS_ALLOWED_METHODS
        valueFrom:
          configMapKeyRef:
            name: platform-config
            key: CORS_ALLOWED_METHODS
      - name: CORS_ALLOWED_HEADERS
        valueFrom:
          configMapKeyRef:
            name: platform-config
            key: CORS_ALLOWED_HEADERS
      - name: CORS_ALLOW_CREDENTIALS
        valueFrom:
          configMapKeyRef:
            name: platform-config
            key: CORS_ALLOW_CREDENTIALS
      - name: CORS_MAX_AGE
        valueFrom:
          configMapKeyRef:
            name: platform-config
            key: CORS_MAX_AGE
      - name: KEYCLOAK_ISSUER
        valueFrom:
          configMapKeyRef:
            name: platform-config
            key: KEYCLOAK_ISSUER
      - name: KEYCLOAK_JWKS_URL
        valueFrom:
          configMapKeyRef:
            name: platform-config
            key: KEYCLOAK_JWKS_URL
      
      # Service config
      - name: PRODUCT_ADMIN_PORT
        valueFrom:
          configMapKeyRef:
            name: product-admin-config
            key: PRODUCT_ADMIN_PORT
      - name: PRODUCT_WRITE_TOPIC
        valueFrom:
          configMapKeyRef:
            name: product-admin-config
            key: PRODUCT_WRITE_TOPIC
      - name: PRODUCT_ADMIN_GROUP
        valueFrom:
          configMapKeyRef:
            name: product-admin-config
            key: PRODUCT_ADMIN_GROUP
      - name: MINIO_BUCKET
        valueFrom:
          configMapKeyRef:
            name: product-admin-config
            key: MINIO_BUCKET
      
      # Secrets (selective)
      - name: DB_URL
        valueFrom:
          secretKeyRef:
            name: product-admin-db-secret
            key: DB_URL
      - name: KEYCLOAK_CLIENT_SECRET
        valueFrom:
          secretKeyRef:
            name: product-admin-keycloak-secret
            key: KEYCLOAK_CLIENT_SECRET
      - name: MINIO_ACCESS_KEY
        valueFrom:
          secretKeyRef:
            name: minio-bootstrap-secret
            key: MINIO_ACCESS_KEY
      - name: MINIO_SECRET_KEY
        valueFrom:
          secretKeyRef:
            name: minio-bootstrap-secret
            key: MINIO_SECRET_KEY
```

### 3.4 Platform ConfigMap Updates

**Update `deploy/k8s/platform/config/platform-configmap-for-services.yaml`**:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: platform-config
  namespace: shopping
data:
  # General Settings (shared by all services)
  LOG_LEVEL: "info"
  
  # Database Connection Pool Settings (shared by all services)
  DATABASE_MAX_OPEN_CONNS: "25"
  DATABASE_MAX_IDLE_CONNS: "25"
  DATABASE_CONN_MAX_LIFETIME: "5m"
  DATABASE_CONN_MAX_IDLE_TIME: "5m"
  DATABASE_CONNECT_TIMEOUT: "30s"
  DATABASE_QUERY_TIMEOUT: "60s"
  DATABASE_HEALTH_CHECK_INTERVAL: "30s"
  DATABASE_HEALTH_CHECK_TIMEOUT: "5s"
  
  # Kafka Configuration (shared by services that need it)
  KAFKA_BROKERS: "kafka-clusterip.kafka.svc.cluster.local:9092"
  
  # Outbox Pattern Configuration (shared by services that publish events)
  OUTBOX_BATCH_SIZE: "10"
  OUTBOX_DELETE_BATCH_SIZE: "10"
  OUTBOX_PROCESS_INTERVAL: "30s"
  OUTBOX_MAX_RETRIES: "3"
  
  # CORS Configuration (shared by services that need it)
  CORS_ALLOWED_ORIGINS: "http://localhost:4200,http://localhost:3000"
  CORS_ALLOWED_METHODS: "GET,POST,PUT,DELETE,PATCH,OPTIONS"
  CORS_ALLOWED_HEADERS: "Content-Type,Authorization,X-Requested-With"
  CORS_ALLOW_CREDENTIALS: "true"
  CORS_MAX_AGE: "3600"
  
  # MinIO Platform Settings (shared by services that need it)
  MINIO_ENDPOINT_KUBERNETES: "minio.minio.svc.cluster.local:9000"
  MINIO_ENDPOINT_LOCAL: "http://api.minio.local"
  MINIO_TLS_VERIFY: "false"
  
  # Keycloak Configuration (shared by authenticated services)
  KEYCLOAK_ISSUER: "http://keycloak.keycloak.svc.cluster.local:8080/realms/pocstore-realm"
  KEYCLOAK_JWKS_URL: "http://keycloak.keycloak.svc.cluster.local:8080/realms/pocstore-realm/protocol/openid-connect/certs"
```

**Changes from current**:
- Added `KEYCLOAK_ISSUER`
- Added `KEYCLOAK_JWKS_URL`
- (MinIO credentials moved from platform-config to individual service secrets)

### 3.5 Service-Specific ConfigMaps

**Create `deploy/k8s/service/product-catalog/product-catalog-configmap.yaml`**:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: product-catalog-config
  namespace: shopping
data:
  # Catalog Service Port
  PRODUCT_CATALOG_PORT: ":8081"
```

**Create `deploy/k8s/service/product-admin/product-admin-configmap.yaml`**:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: product-admin-config
  namespace: shopping
data:
  # Admin Service Port
  PRODUCT_ADMIN_PORT: ":8082"
  
  # Admin Kafka Settings
  PRODUCT_WRITE_TOPIC: "ProductEvents"
  PRODUCT_ADMIN_GROUP: "ProductAdminGroup"
  
  # MinIO Bucket (for image uploads)
  MINIO_BUCKET: "productimages"
```

### 3.6 Backward Compatibility Assessment

**Services Affected**:
- ✅ **Customer Service**: NO breaking changes
  - Config struct unchanged
  - ConfigMap unchanged (`customer-config`)
  - Secret unchanged (`customer-db-secret`)
  - Only Kubernetes deployment YAML changes (explicit env declarations)
  
- ✅ **EventReader Service**: NO breaking changes
  - Config struct unchanged
  - ConfigMap unchanged (`eventreader-config`)
  - Secret unchanged (uses platform config for Kafka, no DB secret)
  - Only Kubernetes deployment YAML changes (explicit env declarations)

- ✅ **Product-Loader**: NO breaking changes
  - Config struct unchanged
  - ConfigMap unchanged (uses `product-loader` config)
  - Continues to load all dependencies it needs

**New Services**:
- ✅ **Product-Catalog**: Uses explicit env (Kubernetes changes only)
- ✅ **Product-Admin**: Uses explicit env (Kubernetes changes only)

### 3.7 Breaking Changes

**ZERO Breaking Changes**:
- No changes to Go code
- No changes to Config structs
- No changes to ConfigMaps (except new ones for new services)
- No changes to Secret data
- Only changes to Kubernetes deployment YAML (switching from `envFrom` to explicit `env`)

---

## Section 4: Frontend Product Loading Strategy

### 4.1 Current Catalog Capabilities

**Existing Endpoints** (after refactor):
```
GET /api/v1/products                 # List all (paginated)
GET /api/v1/products/{id}            # Get single product
GET /api/v1/products/search           # Search products
GET /api/v1/products/category/{cat}   # Filter by category
GET /api/v1/products/brand/{brand}    # Filter by brand
GET /api/v1/products/in-stock         # Filter by availability
```

### 4.2 Recommended Frontend Loading Patterns

#### Pattern 1: Product Listing Page (Catalog Browse)
**Scenario**: User browses catalog, sees product grid
**Recommended Approach**:
```
1. Initial Load:
   GET /api/v1/products?limit=50&offset=0
   
2. Lazy Load / Infinite Scroll:
   GET /api/v1/products?limit=50&offset=50
   GET /api/v1/products?limit=50&offset=100
```

**Pros**:
- Fast initial page load
- Progressive loading as user scrolls
- Simple pagination logic
- Can be cached by browser

#### Pattern 2: Category/Brand Page
**Scenario**: User navigates to "Clothing" category or "Nike" brand page
**Recommended Approach**:
```
1. Initial Load:
   GET /api/v1/products/category/clothing?limit=50&offset=0
   
2. Filter/Search on Page:
   GET /api/v1/products/category/clothing/search?color=red&limit=50&offset=0
```

#### Pattern 3: Product Detail Page
**Scenario**: User clicks on a product to view details
**Recommended Approach**:
```
1. Single Request (already supported):
   GET /api/v1/products/12345
   
2. Optional: Publish View Event
   POST /api/v1/products/12345/view (optional, lightweight)
   # This would require new endpoint in catalog
```

#### Pattern 4: Search Results Page
**Scenario**: User searches for "blue running shoes"
**Recommended Approach**:
```
1. Search Query:
   GET /api/v1/products/search?q=blue+running+shoes&limit=20&offset=0
   
2. Faceted Filtering:
   GET /api/v1/products/search?q=blue+running+shoes&category=shoes&brand=nike
```

### 4.3 Missing Endpoint: Get All Products

**Current Status**: No explicit "get all products" endpoint with pagination
**Recommendation**: Add endpoint for catalog browse page

**New Endpoint to Add**:
```go
// In handler.go
func (h *CatalogHandler) GetAllProducts(w http.ResponseWriter, r *http.Request) {
    limit, offset := parsePagination(r)
    products, err := h.service.GetAllProducts(r.Context(), limit, offset)
    // ...
}
```

**Route**:
```go
router.Get("/api/v1/products", handler.GetAllProducts)
```

**Response Format**:
```json
{
  "products": [...],
  "limit": 50,
  "offset": 0,
  "count": 1234,
  "total_pages": 25
}
```

### 4.4 Advanced Frontend Optimizations (Future)

**Add These After Initial Refactor**:

1. **Cursor-Based Pagination**:
   ```
   GET /api/v1/products?cursor=abc123&limit=50
   ```
   - More efficient than offset-based for large datasets
   - Better for infinite scroll

2. **Product Recommendation Endpoint**:
   ```
   GET /api/v1/products/recommended?user_id=xyz
   ```
   - Collaborative filtering
   - Similar items

3. **Product Comparison Endpoint**:
   ```
   GET /api/v1/products/compare?ids=123,456,789
   ```
   - Side-by-side comparison data

4. **Batch Product Lookup**:
   ```
   POST /api/v1/products/batch
   Body: {"product_ids": [123, 456, 789]}
   ```
   - Reduce round trips for product cards grid

---

## Section 5: Catalog Event Publishing

### 5.1 Event Types to Add

**Add to `internal/contracts/events/product.go`**:
```go
const (
    // Existing events...
    ProductCreated ProductEventType = "product.created"
    ProductUpdated ProductEventType = "product.updated"
    ProductDeleted ProductEventType = "product.deleted"
    
    // New view events
    ProductViewed         ProductEventType = "product.viewed"
    ProductSearchExecuted  ProductEventType = "product.search.executed"
    ProductCategoryViewed  ProductEventType = "product.category.viewed"
)
```

### 5.2 Event Publishing Implementation

**In `internal/service/product/service.go`**:
```go
func (s *CatalogService) GetProductByID(ctx context.Context, productID int64) (*Product, error) {
    log.Printf("[INFO] CatalogService: Fetching product by ID: %d", productID)
    
    product, err := s.repo.GetProductByID(ctx, productID)
    if err != nil {
        log.Printf("[ERROR] CatalogService: Failed to get product %d: %v", productID, err)
        return nil, fmt.Errorf("failed to get product: %w", err)
    }
    
    // Publish product viewed event (non-blocking, best effort)
    go func() {
        event := events.NewProductViewedEvent(fmt.Sprintf("%d", productID), map[string]string{
            "product_id": fmt.Sprintf("%d", product.ID),
            "name":        product.Name,
            "brand":       product.Brand,
            "category":    product.Category,
        })
        if err := s.publishEvent(ctx, event); err != nil {
            log.Printf("[WARN] CatalogService: Failed to publish product viewed event: %v", err)
        }
    }()
    
    return product, nil
}
```

### 5.3 Event Outbox Integration

**Catalog Service Infrastructure**:
```go
type CatalogInfrastructure struct {
    Database       database.Database
    OutboxWriter   *outbox.Writer  // Optional - add if catalog publishes events
}

func NewCatalogServiceWithEvents(repo ProductRepository, outbox *outbox.Writer) *CatalogService {
    return &CatalogService{
        repo: repo,
        outbox: outbox,  // Add outbox reference
    }
}
```

**Optional Event Publishing**:
- Events are non-blocking (goroutine)
- Event publication failures don't block product retrieval
- Best-effort event delivery (asynchronous)

---

## Section 6: Repository Split (NO CHANGES FROM V1.0)

Split repository into focused files (same as v1.0 plan):
- `repository_crud.go` (~250 lines)
- `repository_query.go` (~200 lines)
- `repository_image.go` (~300 lines)
- `repository_bulk.go` (~70 lines)
- `repository_util.go` (~100 lines)

---

## Section 7: Service & Handler Split (NO CHANGES FROM V1.0)

Same approach as v1.0 plan:
- `service.go` - Read-only service
- `service_admin.go` - Write service with event publishing
- `handler.go` - Read handlers
- `handler_admin.go` - Write handlers with auth middleware

---

## Section 8: Updated Implementation Plan

### Phase 1: Keycloak Setup (1 hour)
1. Create `product-admin` client in Keycloak (service account)
2. Create `product-admin` role in Keycloak
3. Assign role to service account
4. Generate client secret
5. Create Kubernetes secret `product-admin-keycloak-secret`
6. Update platform ConfigMap with Keycloak settings

### Phase 2: Repository Split (2 hours)
Same as v1.0 plan - split `repository.go` into focused files

### Phase 3: Configuration Refactoring (2 hours)
1. Update all Kubernetes deployment YAMLs (customer, eventreader, product-catalog, product-admin)
2. Switch from `envFrom:` to explicit `env:` declarations
3. Create new service-specific ConfigMaps (product-catalog, product-admin)
4. Update platform ConfigMap with Keycloak settings
5. Test that all services still work (no breaking changes)

### Phase 4: Catalog Event Publishing (1 hour)
1. Add new event types to `internal/contracts/events/product.go`
2. Update `CatalogService` to publish view events
3. Add outbox dependency to catalog infrastructure (optional)

### Phase 5: Handler & Service Split (3 hours)
Same as v1.0 plan - create catalog/admin handlers and services

### Phase 6: Authentication Implementation (3 hours)
1. Install `github.com/zitadel/oidc` library
2. Create `internal/platform/auth/` package
3. Implement JWT validation middleware
4. Add authentication to product-admin routes
5. Add `product-admin` role check in middleware

### Phase 7: Create New Services (2 hours)
Same as v1.0 plan - create product-catalog and product-admin main.go

### Phase 8: Kubernetes Deployment (3 hours)
1. Create deployment YAMLs for new services
2. Create ingress YAMLs with `/api/v1/` paths
3. Update Makefile with new build/deploy targets
4. Test deployment locally

### Phase 9: Testing & Validation (1.5 hours)
Same as v1.0 plan + test:
- Event publishing from catalog
- Frontend loading patterns
- Keycloak authentication flow

### Phase 10: Cleanup (1 hour)
Same as v1.0 plan

**Total Estimated Effort**: 10-14 hours (restored from v1.0)

---

## Section 9: Risk Assessment

### Configuration Refactoring Risks
| Risk | Impact | Likelihood | Mitigation |
|-------|---------|-------------|------------|
| Missing environment variable breaks service | High | Low | Test each service after config changes |
| Typo in env key name causes silent failure | Medium | Medium | Validate all env keys match mapstructure tags |
| Platform ConfigMap changes affect multiple services | Medium | Low | Only add new keys, don't remove existing ones |

### API Nomenclature Risks
| Risk | Impact | Likelihood | Mitigation |
|-------|---------|-------------|------------|
| Frontend breaks due to URL changes | High | Medium | Deploy catalog first, validate, then update frontend |
| Path conflicts between services | Medium | Low | Test ingress routing after deployment |

---

## Section 10: Success Criteria

### Configuration Refactoring
- ✅ Zero breaking changes to existing services (customer, eventreader, product-loader)
- ✅ Services only load environment variables they need
- ✅ MinIO credentials not loaded into catalog service
- ✅ All existing tests pass

### API Standardization
- ✅ All new endpoints use `/api/v1/` prefix
- ✅ Clear separation between catalog and admin paths
- ✅ Health endpoints remain accessible
- ✅ Ingress routing works correctly

### Event Publishing
- ✅ Catalog service publishes view events
- ✅ Event publication is non-blocking (async)
- ✅ Event publication failures don't block product retrieval

### Frontend Integration
- ✅ Frontend can load products with pagination
- ✅ Search and filter endpoints work as expected
- ✅ Single product detail retrieval works
- ✅ Recommended patterns documented

---

## Appendix A: Configuration Migration Checklist

### Customer Service
- [ ] Update `customer-deploy.yaml` to use explicit `env:` declarations
- [ ] Verify only these env vars loaded: `LOG_LEVEL`, `CUSTOMER_SERVICE_PORT`, `CUSTOMER_WRITE_TOPIC`, `CUSTOMER_GROUP`, `DB_URL`
- [ ] Test customer service after deployment
- [ ] Verify customer still publishes events to Kafka

### EventReader Service
- [ ] Update `eventreader-deploy.yaml` to use explicit `env:` declarations
- [ ] Verify only these env vars loaded: `LOG_LEVEL`, `EVENT_READER_WRITE_TOPIC`, `EVENT_READER_READ_TOPICS`, `EVENT_READER_GROUP`, `KAFKA_BROKERS`
- [ ] Test eventreader after deployment

### Product-Catalog Service
- [ ] Create `product-catalog-configmap.yaml`
- [ ] Create `product-catalog-db-secret`
- [ ] Create `product-catalog-deploy.yaml` with explicit env
- [ ] Verify MinIO credentials NOT loaded
- [ ] Test catalog endpoints
- [ ] Test event publishing

### Product-Admin Service
- [ ] Create `product-admin-configmap.yaml`
- [ ] Create `product-admin-db-secret`
- [ ] Create `product-admin-keycloak-secret`
- [ ] Create `product-admin-deploy.yaml` with explicit env
- [ ] Verify MinIO credentials loaded (for image uploads)
- [ ] Verify Keycloak credentials loaded
- [ ] Test admin endpoints with authentication
- [ ] Test admin endpoints WITHOUT authentication (should fail)

---

## Appendix B: Frontend Integration Example

### React/Vue/Angular Example

```javascript
// API Client Setup
const API_BASE = 'https://pocstore.local/api/v1';

// Product Listing with Pagination
async function loadProducts(page = 0) {
  const offset = page * 50;
  const response = await fetch(`${API_BASE}/products?limit=50&offset=${offset}`);
  const data = await response.json();
  return data.products;
}

// Product Detail
async function loadProduct(productId) {
  const response = await fetch(`${API_BASE}/products/${productId}`);
  const data = await response.json();
  return data;
}

// Search
async function searchProducts(query) {
  const response = await fetch(`${API_BASE}/products/search?q=${encodeURIComponent(query)}&limit=20`);
  const data = await response.json();
  return data.products;
}

// Category Filter
async function loadByCategory(category) {
  const response = await fetch(`${API_BASE}/products/category/${encodeURIComponent(category)}?limit=50`);
  const data = await response.json();
  return data.products;
}
```

---

## Appendix C: Makefile Updates

### Required Changes (Just 2 Lines!)

**Update `resources/make/services.mk`**:

**Line 6** - Update SERVICES variable:
```makefile
# Change from:
SERVICES := customer eventreader product

# To:
SERVICES := customer eventreader product-catalog product-admin
```

**Lines 8-10** - Add service-specific DB flags:
```makefile
# Add after line 10:
SERVICE_HAS_DB_product-catalog := true
SERVICE_HAS_DB_product-admin := true
```

### That's It! No Explicit Targets Needed

**Automatic Target Generation** (via lines 87-89):
```makefile
# The foreach loop will automatically generate these:
product-catalog-build
product-catalog-test
product-catalog-lint
product-catalog-deploy
product-catalog-install
product-catalog-db-create
product-catalog-db-migrate

product-admin-build
product-admin-test
product-admin-lint
product-admin-deploy
product-admin-install
product-admin-db-create
product-admin-db-migrate
```

**Aggregate Targets** (via lines 93-115):
```makefile
# These are automatically generated:
services-build          # builds: customer, eventreader, product, product-catalog, product-admin
services-install        # installs: customer, eventreader, product, product-catalog, product-admin
services-db-create      # creates DBs for: customer, product, product-catalog, product-admin
services-db-migrate    # migrates DBs for: customer, product, product-catalog, product-admin
```

### Verification

After these 2 lines are added, you can run:

```bash
# Build new services
make product-catalog-build
make product-admin-build

# Deploy new services
make product-catalog-deploy
make product-admin-deploy

# Or deploy all services including new ones
make services-install
```

### What This Eliminates

From my original plan (v1.1), this **eliminates**:
- ❌ 50+ lines of explicit target definitions
- ❌ Duplicate logic that already exists in `services.mk`
- ❌ Maintenance burden of keeping targets in sync

### Why Your Approach Is Better

1. **DRY Principle**: Uses existing foreach loop
2. **Convention**: Follows exact same pattern as existing services
3. **Maintainable**: New services automatically get all targets
4. **Simple**: 2 lines vs 50+ lines

---

## Document History

| Version | Date | Changes |
|---------|-------|---------|
| 1.0 | Jan 22, 2026 | Initial plan draft |
| 1.1 | Jan 22, 2026 | Refined based on feedback: Fixed Keycloak namespace, detailed API nomenclature, configuration refactoring approach, frontend loading strategy, catalog event publishing |
| 1.2 | Jan 22, 2026 | Corrected Appendix C to use existing services.mk pattern instead of explicit target definitions (2 lines vs 50+ lines) |
