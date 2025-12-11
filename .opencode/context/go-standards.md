# Go Standards & Patterns

## Quick Reference

**Core Philosophy**: Simple, Readable, Efficient
**Golden Rule**: If it's complex, there's probably a simpler Go way

**Critical Patterns** (use these):
- ✅ Explicit error handling
- ✅ Interfaces for behavior
- ✅ Composition over inheritance
- ✅ Small, focused packages
- ✅ Clear naming conventions

**Anti-Patterns** (avoid these):
- ❌ Panic for expected errors
- ❌ Deep nesting
- ❌ Global state
- ❌ Unnecessary pointers

---

## Package Structure

### Standard Layout
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

### Package Naming
- **Lowercase, single word** when possible
- **Descriptive but concise**: `httpserver`, `userauth`
- **No "util" packages** - be specific about functionality
- **Internal packages**: Use `internal/` for private code

---

## Error Handling

### Standard Error Pattern
```go
// ✅ Good: Explicit error handling
func GetUser(id int) (*User, error) {
    if id <= 0 {
        return nil, fmt.Errorf("invalid user id: %d", id)
    }
    
    user, err := db.FindUser(id)
    if err != nil {
        return nil, fmt.Errorf("failed to find user: %w", err)
    }
    
    return user, nil
}

// ❌ Bad: Panic for expected errors
func GetUser(id int) *User {
    if id <= 0 {
        panic("invalid user id")
    }
    // ...
}
```

### Error Wrapping
```go
// ✅ Good: Wrap with context
if err := validateInput(data); err != nil {
    return fmt.Errorf("input validation failed: %w", err)
}

// ✅ Good: Custom error types
type ValidationError struct {
    Field string
    Value string
    Msg   string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation failed for %s: %s", e.Field, e.Msg)
}
```

---

## Interfaces

### Interface Design
```go
// ✅ Good: Small, focused interfaces
type Reader interface {
    Read([]byte) (int, error)
}

type Writer interface {
    Write([]byte) (int, error)
}

// ✅ Good: Accept interfaces, return structs
func ProcessData(r Reader, w Writer) error {
    data := make([]byte, 1024)
    n, err := r.Read(data)
    if err != nil {
        return err
    }
    _, err = w.Write(data[:n])
    return err
}

// ❌ Bad: Large interfaces
type Database interface {
    CreateUser(user User) error
    GetUser(id int) (*User, error)
    UpdateUser(user User) error
    DeleteUser(id int) error
    CreatePost(post Post) error
    // ... 20 more methods
}
```

### Interface Composition
```go
// ✅ Good: Compose small interfaces
type ReadWriter interface {
    Reader
    Writer
}

type ReadCloser interface {
    Reader
    io.Closer
}
```

---

## Concurrency

### Goroutines
```go
// ✅ Good: Use channels for communication
func ProcessItems(items []Item) []Result {
    results := make(chan Result, len(items))
    
    for _, item := range items {
        go func(i Item) {
            results <- processItem(i)
        }(item)
    }
    
    var output []Result
    for i := 0; i < len(items); i++ {
        output = append(output, <-results)
    }
    
    return output
}

// ❌ Bad: Shared memory without synchronization
var counter int

func Increment() {
    counter++ // Race condition!
}
```

### Worker Pool Pattern
```go
func worker(jobs <-chan Job, results chan<- Result) {
    for j := range jobs {
        results <- processJob(j)
    }
}

func StartWorkers(numWorkers int, jobs []Job) []Result {
    jobsChan := make(chan Job, len(jobs))
    resultsChan := make(chan Result, len(jobs))
    
    // Start workers
    for i := 0; i < numWorkers; i++ {
        go worker(jobsChan, resultsChan)
    }
    
    // Send jobs
    for _, job := range jobs {
        jobsChan <- job
    }
    close(jobsChan)
    
    // Collect results
    var results []Result
    for i := 0; i < len(jobs); i++ {
        results = append(results, <-resultsChan)
    }
    
    return results
}
```

---

## Structs and Methods

### Struct Design
```go
// ✅ Good: Group related fields
type User struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// ✅ Good: Pointer receivers for mutation
func (u *User) UpdateEmail(email string) {
    u.Email = email
    u.UpdatedAt = time.Now()
}

// ✅ Good: Value receivers for immutable operations
func (u User) IsValid() bool {
    return u.Email != "" && u.Name != ""
}
```

### Constructor Pattern
```go
// ✅ Good: Named constructors
func NewUser(name, email string) (*User, error) {
    if name == "" {
        return nil, fmt.Errorf("name cannot be empty")
    }
    if email == "" {
        return nil, fmt.Errorf("email cannot be empty")
    }
    
    return &User{
        Name:      name,
        Email:     email,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }, nil
}
```

---

## Testing Patterns

### Table-Driven Tests
```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive", 2, 3, 5},
        {"negative", -1, -1, -2},
        {"zero", 0, 0, 0},
        {"mixed", -1, 1, 0},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Add(tt.a, tt.b)
            if result != tt.expected {
                t.Errorf("Add(%d, %d) = %d; want %d", tt.a, tt.b, result, tt.expected)
            }
        })
    }
}
```

### Mock Interfaces
```go
// Mock implementation
type MockRepository struct {
    users map[int]*User
    mu    sync.RWMutex
}

func (m *MockRepository) FindUser(id int) (*User, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    user, exists := m.users[id]
    if !exists {
        return nil, fmt.Errorf("user not found: %d", id)
    }
    return user, nil
}

// Test with mock
func TestUserService_GetUser(t *testing.T) {
    mockRepo := &MockRepository{
        users: map[int]*User{
            1: {ID: 1, Name: "John", Email: "john@example.com"},
        },
    }
    
    service := NewUserService(mockRepo)
    user, err := service.GetUser(1)
    
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    
    if user.Name != "John" {
        t.Errorf("expected name John, got %s", user.Name)
    }
}
```

---

## HTTP Handlers

### Standard Handler Pattern
```go
// ✅ Good: Dependency injection
type UserHandler struct {
    userService UserService
    logger      Logger
}

func NewUserHandler(userService UserService, logger Logger) *UserHandler {
    return &UserHandler{
        userService: userService,
        logger:      logger,
    }
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.Atoi(mux.Vars(r)["id"])
    if err != nil {
        http.Error(w, "invalid user id", http.StatusBadRequest)
        return
    }
    
    user, err := h.userService.GetUser(id)
    if err != nil {
        h.logger.Printf("failed to get user: %v", err)
        http.Error(w, "user not found", http.StatusNotFound)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}
```

### Response Helper
```go
type Response struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}

func WriteError(w http.ResponseWriter, status int, message string) {
    WriteJSON(w, status, Response{
        Success: false,
        Error:   message,
    })
}
```

---

## Configuration

### Environment-Based Config
```go
type Config struct {
    ServerPort string `env:"SERVER_PORT" default:"8080"`
    DatabaseURL string `env:"DATABASE_URL" required:"true"`
    LogLevel    string `env:"LOG_LEVEL" default:"info"`
    RedisURL    string `env:"REDIS_URL"`
}

func LoadConfig() (*Config, error) {
    cfg := &Config{}
    
    if err := env.Parse(cfg); err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }
    
    return cfg, nil
}
```

---

## Naming Conventions

### General Rules
- **Package names**: `lowercase`, `singleword`, `descriptive`
- **Constants**: `UPPER_SNAKE_CASE`
- **Variables**: `camelCase`, `descriptive`
- **Functions**: `PascalCase` for exported, `camelCase` for unexported
- **Interfaces**: Usually `-er` suffix: `Reader`, `Writer`, `Server`

### Examples
```go
// ✅ Good
const MaxRetries = 3
type Database interface { /* ... */ }
func NewUserService(db Database) *UserService { /* ... */ }
var userCache map[int]*User

// ❌ Bad
const max_retries = 3
type DB interface { /* ... */ }
func newuserservice(db Database) *UserService { /* ... */ }
var UserCache map[int]*User
```

---

## Best Practices Checklist

Before committing code, verify:
- ✅ Error handling is explicit and consistent
- ✅ Interfaces are small and focused
- ✅ Package structure follows Go conventions
- ✅ Naming is clear and consistent
- ✅ Tests cover main functionality
- ✅ No unnecessary global state
- ✅ Concurrency is handled safely
- ✅ Code is readable and simple

---

## Anti-Patterns to Avoid

❌ **Don't use panic for expected errors**
❌ **Don't create deep nesting** - use early returns
❌ **Don't use global variables** - pass dependencies explicitly
❌ **Don't create "util" packages** - be specific about functionality
❌ **Don't ignore errors** - always handle them
❌ **Don't overuse pointers** - use values when appropriate
❌ **Don't create large interfaces** - keep them small and focused

This standards document provides the foundation for writing clean, maintainable Go code that follows idiomatic patterns and best practices.