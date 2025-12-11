# Project Context - Go Configuration

## Project Information

**Project Name**: go-shopping-poc   

**Description**: This project is being used as a proof-of-concept to learn the Go programming language and related concepts.  We are building the microservices needed to support a fictitious shopping application: customer, shoppingcart, order, product, etc.

To support the application we will be using Keycloak (OIDC authentication and authorization), Postgres (database), Kafka (event management), and Minio (S3 storage for product images).  

The microservices and the supporting services will all run inside of a local kubernetes instance from Rancher Desktop running on a a local Mac development machine. 

The front-end application that accesses these services is being written in Angular and is housed in a separate project.  

We will be using the Saga and Outbox patterns to ensure the microservices remain independent. 

We want to learn and use best practices for Go.  

**Go Version**: 1.24+  

**Primary Domain**: Event-driven Go microservices

## Project Structure

This project follows the standard Go project layout:

```
project/
├── go.mod
├── go.sum
├── Makefile
├── README.md
|
├── cmd/                                # Entrypoints for each microservice
│   ├── customer/
│   │   └── main.go
│   │   └── config.yaml
│   ├── order/
│   │   └── main.go
│   │   └── config.yaml
│   └── ...
│
├── deployments/                         # Kubernetes, Helm, Docker Compose, etc.
│   ├── k8s/
│   ├── helm/
│   └── docker-compose/
│
├── docs/                                # Architecture, specs, decision records
│
├── internal/                            # Private code (not importable)
│   │
│   ├── contracts/                       # Pure DTOs (events, requests, responses)
│   │   ├── events/
│   │   │   ├── customer.go
│   │   │   ├── order.go
│   │   │   └── ...
│   │   └── common.go
│   │
│   ├── platform/                        # Shared infrastructure (non-domain-specific)
│   │   ├── config/                      # Config loading, env, options
│   │   ├── logging/                     # Logging utilities / wrappers
│   │   ├── outbox/                      # Outbox pattern implementation
│   │   ├── event/
│   │   │   ├── bus/                     # Kafka (or other bus) implementation
│   │   │   │   ├── kafka/
│   │   │   │   │   ├── event.go
│   │   │   │   │   └── eventbus.go
│   │   │   ├── handler/                 # Common reusable handlers
│   │   │   │   ├── customer_handler.go  # Only if generic / reusable
│   │   │   │   ├── order_handler.go
│   │   │   │   └── ...
│   │   │   └── middleware.go            # Retry, dead letter, etc.
│   │   └── db/                          # DB utilities (migrations, connections)
│   │
│   ├── service/                         # Domain services (the actual microservices)
│   │   ├── customer/
│   │   │   ├── entity.go
│   │   │   ├── repository.go            # Persistence
│   │   │   ├── service.go               # Business logic
│   │   │   ├── handler.go               # HTTP/RPC handler
│   │   │   ├── eventhandlers/           # Reactions to events from other domains
│   │   │   │   └── on_order_created.go
│   │   │   └── test/
│   │   │   └── ...
│   │   ├── order/
│   │   │   ├── entity.go
│   │   │   ├── repository.go
│   │   │   ├── service.go
│   │   │   ├── handler.go
│   │   │   ├── eventhandlers/
│   │   │   │  └── on_customer_created.go
│   │   │   └── test/
│   │   │   └── ...
│   │
│   └── shared/                          # Optional: helpers shared across services
│       └── ...
│
├── pkg/                                 # Stable public APIs (importable by others)
│   └── (keep minimal)
│
├── resources/                           # Setup/bootstrap resources for local dev
│   ├── kafka/
│   ├── keycloak/
│   ├── make/
│   ├── postgresql/
│   │   ├── schemas/
│   │   │   ├── customer_db.sql
│   │   │   └── order_db.sql
│   │   └── init/
│   └── certificates/
│
└── scripts/                             # Utility scripts (bash, migrations, hooks)
    └── ...

```

## Technology Stack

### Core Technologies
- **Language**: Go 1.24+
- **Web Framework**: Chi, gorilla/websocket
- **Database**: PostgreSQL
- **ORM**: sqlx
- **Authentication**: OIDC(Keycloak)

### Development Tools
- **Testing**: Go testing package + [testify if needed]
- **Linting**: golangci-lint
- **Documentation**: godoc
- **Build**: Make

## Configuration

### Environment Variables

Environment variables for the project can be found in the following files in the root directory of the project:

``` bash
.env
.env.local
```

### Configuration Structure
This is the current structure as defined in the file config.go:
```go
type Config struct {
	// Kafka configuration
	EventBroker           string
	EventWriterWriteTopic string
	EventWriterReadTopics []string
	EventWriterGroup      string
	EventReaderWriteTopic string
	EventReaderReadTopics []string
	EventReaderGroup      string

	// WebSocket configuration
	WebSocketURL         string
	WebSocketTimeoutMs   int
	WebSocketReadBuffer  int
	WebSocketWriteBuffer int
	WebSocketPort        string

	// Customer service configuration
	CustomerDBURL          string
	CustomerDBURLLocal     string
	CustomerServicePort    string
	CustomerWriteTopic     string
	CustomerReadTopics     []string
	CustomerGroup          string
	CustomerOutboxInterval time.Duration

	// CORS configuration
	CORSAllowedOrigins   string
	CORSAllowedMethods   string
	CORSAllowedHeaders   string
	CORSAllowCredentials bool
	CORSMaxAge           string
}
```

## Coding Standards

### Naming Conventions
- **Package names**: `lowercase`, `singleword` when possible
- **File names**: `snake_case.go`
- **Constants**: `UPPER_SNAKE_CASE`
- **Variables**: `camelCase`
- **Functions**: `PascalCase` for exported, `camelCase` for unexported
- **Interfaces**: Usually `-er` suffix: `Reader`, `Writer`, `Service`

### Project-Specific Patterns

#### Error Handling
```go
// Use custom error types for better error handling
type ValidationError struct {
    Field   string
    Message string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation failed for %s: %s", e.Field, e.Message)
}

// Wrap errors with context
if err := validateInput(input); err != nil {
    return fmt.Errorf("input validation failed: %w", err)
}
```

#### Database Operations
```go
// Use repository pattern for data access
type UserRepository interface {
    Create(user *User) error
    FindByID(id int) (*User, error)
    Update(user *User) error
    Delete(id int) error
    FindByEmail(email string) (*User, error)
}

// Implement with transactions for complex operations
func (s *UserService) CreateUserWithProfile(userData UserData, profileData ProfileData) error {
    return s.repo.WithTransaction(func(tx *sql.Tx) error {
        user := &User{...}
        if err := tx.Create(user); err != nil {
            return err
        }
        
        profile := &Profile{UserID: user.ID, ...}
        return tx.CreateProfile(profile)
    })
}
```

#### HTTP Handlers
```go
// Use consistent response format
type Response struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}

// Middleware pattern for common functionality
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Authentication logic
        next.ServeHTTP(w, r)
    })
}
```

## Testing Strategy

### Test Organization
- **Unit tests**: `*_test.go` files alongside source code
- **Integration tests**: `tests/integration/` directory
- **End-to-end tests**: `tests/e2e/` directory
- **Benchmark tests**: `*_bench_test.go` files

### Test Data Management
```go
// Use test fixtures for consistent test data
type TestFixture struct {
    Users []User
    Posts []Post
}

func SetupTestFixture(db *sql.DB) (*TestFixture, error) {
    // Create test data
    return &TestFixture{
        Users: []User{...},
        Posts: []Post{...},
    }, nil
}

func CleanupTestFixture(db *sql.DB, fixture *TestFixture) error {
    // Clean up test data
    return nil
}
```

### Mock Strategy
```go
// Use interfaces for easy mocking
type MockUserService struct {
    users map[int]*User
    mu    sync.RWMutex
}

func (m *MockUserService) CreateUser(name, email string) (*User, error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    user := &User{
        ID:    len(m.users) + 1,
        Name:  name,
        Email: email,
    }
    m.users[user.ID] = user
    return user, nil
}
```

## API Standards

### REST API Conventions
- **URLs**: `/api/v1/resource` (plural nouns)
- **HTTP Methods**: GET (read), POST (create), PUT/PATCH (update), DELETE (delete)
- **Status Codes**: 200 (success), 201 (created), 400 (bad request), 404 (not found), 500 (server error)
- **Response Format**: Consistent JSON structure

### Response Format
```json
{
  "success": true,
  "data": {
    "id": 1,
    "name": "John Doe",
    "email": "john@example.com"
  }
}
```

```json
{
  "success": false,
  "error": "User not found"
}
```

## Development Workflow

### Git Workflow
- **Main branch**: `main`
- **Feature branches**: `feature/description`
- **Release branches**: `release/version`
- **Hotfix branches**: `hotfix/description`

### Commit Message Format
```
type(scope): description

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `style`: Code style (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Tests
- `chore`: Maintenance

### Code Review Process
1. Create pull request
2. Automated checks (tests, linting)
3. Manual code review
4. Address feedback
5. Merge to main

## Performance Requirements

### Response Time Targets
- **API endpoints**: < 200ms (95th percentile)
- **Database queries**: < 100ms average
- **File uploads**: < 5s for 10MB files

### Throughput Targets
- **Concurrent users**: 1000+
- **Requests per second**: 500+
- **Database connections**: 20 max

### Monitoring
- **Metrics**: Response time, error rate, throughput
- **Logging**: Structured logging with correlation IDs
- **Alerting**: Response time > 500ms, error rate > 5%

## Security Requirements

### Authentication & Authorization
- **Authentication**: JWT tokens with expiration
- **Authorization**: Role-based access control (RBAC)
- **Password Security**: bcrypt hashing, minimum 8 characters
- **Session Management**: Secure cookie settings

### Data Protection
- **Input Validation**: All user inputs validated
- **SQL Injection**: Parameterized queries only
- **XSS Prevention**: Output encoding and CSP headers
- **HTTPS**: TLS 1.2+ required in production

### Security Headers
```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000
Content-Security-Policy: default-src 'self'
```

## Deployment

### Environment Configuration
- **Development**: Local development
- **Staging**: None at this time
- **Production**: None at this time

### Build Process
We use Make to manage the build and deployment.

We have the primary Makefile in the project root and have supporting sub-makefiles in the /resources/make directory.  The makefiles contain documentation as to the purpose of each target.

### Container Configuration
```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY ../../go.mod ../../go.sum ./
RUN go mod download

# Copy the entire source code
COPY ../../ ./

# Build the eventreader service
WORKDIR /app/cmd/customer
RUN CGO_ENABLED=0 GOOS=linux go build -o /customer

# Final image
#FROM gcr.io/distroless/base-debian11
FROM scratch
WORKDIR /
COPY --from=builder /customer .
COPY --from=builder /app/.env.local .
COPY --from=builder /etc/passwd /etc/passwd

# Note Be sure to expose ports that match your application just to ensure the container runtime allows them.
EXPOSE 8080

USER nobody

CMD ["/customer"]
```

## Monitoring & Observability

### Metrics Collection
- **Application metrics**: Custom business metrics
- **System metrics**: CPU, memory, disk usage
- **Database metrics**: Connection pool, query performance
- **API metrics**: Request count, response time, error rate

### Logging Strategy
- **Format**: Structured JSON logging
- **Levels**: DEBUG, INFO, WARN, ERROR
- **Context**: Request ID, user ID, correlation ID
- **Retention**: 30 days for logs, 90 days for errors

### Health Checks
```go
// Health check endpoint
func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
    status := map[string]interface{}{
        "status": "healthy",
        "timestamp": time.Now(),
        "version": os.Getenv("APP_VERSION"),
    }
    
    // Check database connectivity
    if err := h.db.Ping(); err != nil {
        status["status"] = "unhealthy"
        status["database"] = "disconnected"
        w.WriteHeader(http.StatusServiceUnavailable)
    }
    
    json.NewEncoder(w).Encode(status)
}
```

## Custom Patterns

### [Project-Specific Pattern 1]
We use the SAGA pattern for an event-driven architecture.

### [Project-Specific Pattern 2]
We use the Outbox pattern to ensure that events are triggered by completed database transactions.

## Integration Points

### External APIs
- **Email Service**: TBD
- **File Storage**: Local S3 - Minio

## Documentation Standards

### Code Documentation
- **Package comments**: Describe package purpose and usage
- **Function comments**: Document parameters, returns, and errors
- **Type comments**: Describe struct fields and usage
- **Example code**: Include usage examples in comments

### API Documentation
- **OpenAPI/Swagger**: Complete API specification
- **Postman Collection**: For manual testing
- **Examples**: Request/response examples for all endpoints

### Developer Documentation
- **README**: Project overview and setup instructions
- **Architecture**: System design and component interaction
- **Contributing**: Development guidelines and process

## Quality Gates

### Automated Checks
- **Tests**: Must pass with 80%+ coverage
- **Linting**: No golangci-lint violations
- **Security**: No vulnerabilities in dependency scan
- **Build**: Must compile without errors

### Manual Review
- **Code review**: Required for all changes
- **Design review**: For significant architectural changes
- **Security review**: For authentication/authorization changes

This project context provides the foundation for all Go development activities in this project. Customize it with your specific requirements, patterns, and conventions.