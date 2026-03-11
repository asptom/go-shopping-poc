# Concurrency

This document describes concurrency patterns used in the project, including goroutines, channels, context usage, and synchronization.

## Overview

Go's concurrency model is based on goroutines (lightweight threads) and channels (typed conduits for communication). This project uses concurrency for:
- Background event processing
- Concurrent request handling
- Timeout and cancellation
- Graceful shutdown

For concurrent workflows, use structured `slog` logging with stable keys (`component`, `operation`, `duration_ms`, `error`) instead of ad-hoc text logging.

## Core Principles

### 1. Always Use Context

Pass context.Context through the entire call chain:

```go
// Good - context propagation
func (s *Service) Process(ctx context.Context, input *Input) error {
    return s.repo.Query(ctx, "SELECT ...")
}

// Bad - no context support
func (s *Service) Process(input *Input) error {
    return s.repo.Query("SELECT ...")  // Can't cancel or timeout
}
```

### 2. Never Store Context

Do not store context in structs:

```go
// Bad - storing context
type Service struct {
    ctx context.Context  // Don't do this
}

// Good - pass as parameter
func (s *Service) Process(ctx context.Context) error { ... }
```

### 3. Prefer Channels Over Shared State

Use channels for communication between goroutines:

```go
// Good - using channels
func ProcessItems(items []Item) <-chan Result {
    results := make(chan Result, len(items))
    
    go func() {
        defer close(results)
        for _, item := range items {
            results <- process(item)
        }
    }()
    
    return results
}
```

## Context Usage

### Cancellation

Use context for operation cancellation:

```go
func (s *EventReaderService) Start(ctx context.Context) error {
    log := s.logger.With("operation", "start_event_reader")
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()  // Graceful exit
        case msg := <-messages:
            if err := s.process(msg); err != nil {
                log.Error("Process message failed", "error", err.Error())
            }
        }
    }
}
```

### Timeout

Set timeouts for operations:

```go
func (s *Service) QueryWithTimeout(ctx context.Context) error {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    return s.repo.Query(ctx, "SELECT ...")
}
```

## Goroutines

### Starting Goroutines

Always know when a goroutine will exit:

```go
// Good - bounded lifetime
go func() {
    defer wg.Done()
    processItem(item)
}()

// Bad - unbounded goroutine
go func() {
    for {  // Never exits!
        process()
    }
}()
```

### Waiting for Goroutines

Use sync.WaitGroup to wait for goroutines:

```go
func ProcessParallel(items []Item) error {
    var wg sync.WaitGroup
    errChan := make(chan error, len(items))
    
    for _, item := range items {
        wg.Add(1)
        go func(i Item) {
            defer wg.Done()
            if err := process(i); err != nil {
                errChan <- err
            }
        }(item)
    }
    
    wg.Wait()
    close(errChan)
    
    for err := range errChan {
        if err != nil {
            return err
        }
    }
    return nil
}
```

### Error Handling in Goroutines

Never silently drop errors from goroutines:

```go
// Good - capture errors
go func() {
    defer wg.Done()
    if err := process(); err != nil {
        errChan <- err
    }
}()

// Bad - silent failure
go func() {
    process()  // Error lost!
}()
```

## Channels

### Channel Direction

Specify channel direction in function signatures:

```go
// Send-only channel
func Producer(out chan<- Item) { ... }

// Receive-only channel
func Consumer(in <-chan Item) { ... }

// Bidirectional channel
func Processor(ch chan Item) { ... }
```

### Buffered Channels

Use buffered channels when you know the capacity:

```go
// Good - pre-sized buffer
results := make(chan Result, len(items))

// Avoid if size is unknown
results := make(chan Result)  // Unbuffered - synchronizes on each send
```

### Closing Channels

Only the sender should close a channel:

```go
// Good - sender closes
func Producer(items []Item) <-chan Item {
    ch := make(chan Item, len(items))
    go func() {
        defer close(ch)  // Sender closes
        for _, item := range items {
            ch <- item
        }
    }()
    return ch
}
```

### Select Statement

Use select for coordinating multiple channels:

```go
func (p *Publisher) run(ctx context.Context) {
    ticker := time.NewTicker(p.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := p.publishPending(); err != nil {
                logger.Error("Publish pending failed", "operation", "publish_pending", "error", err.Error())
            }
        case <-p.quit:
            return
        }
    }
}
```

## Graceful Shutdown

### Signal Handling

Handle OS signals for graceful shutdown:

```go
func main() {
    logger := slog.Default().With("component", "service_main")
    // Setup signal handling
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    // Start service
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    go func() {
        if err := service.Start(ctx); err != nil {
            logger.Error("Service stopped unexpectedly", "operation", "start_service", "error", err.Error())
        }
    }()
    
    // Wait for shutdown signal
    sig := <-sigChan
    logger.Info("Shutdown signal received", "operation", "handle_signal", "signal", sig.String())
    
    // Graceful shutdown
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer shutdownCancel()
    
    if err := service.Stop(shutdownCtx); err != nil {
        logger.Error("Service shutdown failed", "operation", "stop_service", "error", err.Error())
    }
}
```

**Reference:** cmd/customer/main.go, cmd/eventreader/main.go

## Synchronization

### sync.Mutex

Use mutex for protecting shared state:

```go
type Counter struct {
    mu    sync.Mutex
    count int
}

func (c *Counter) Inc() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count++
}

func (c *Counter) Get() int {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.count
}
```

### sync.RWMutex

Use RWMutex when reads outnumber writes:

```go
type Cache struct {
    mu    sync.RWMutex
    data  map[string]Item
}

func (c *Cache) Get(key string) (Item, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    item, ok := c.data[key]
    return item, ok
}

func (c *Cache) Set(key string, item Item) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.data[key] = item
}
```

### sync.Once

Use Once for one-time initialization:

```go
type Singleton struct {
    once     sync.Once
    instance *Service
}

func (s *Singleton) Get() *Service {
    s.once.Do(func() {
        s.instance = NewService()
    })
    return s.instance
}
```

## Best Practices

### DO:
- Pass context through the entire call chain
- Use channels for goroutine communication
- Close channels only from sender
- Use WaitGroup to wait for goroutines
- Handle errors from goroutines
- Set timeouts on operations
- Use signal handling for graceful shutdown
- Protect shared state with mutexes

### DON'T:
- Store context in structs
- Share memory without synchronization
- Leave goroutines running indefinitely
- Close channels from receiver
- Ignore errors from goroutines
- Use global variables for shared state
- Create unbounded goroutines
- Use panic for normal error handling

## Common Patterns Summary

```go
// Worker pool
func WorkerPool(jobs <-chan Job, workers int) <-chan Result {
    results := make(chan Result, workers)
    var wg sync.WaitGroup
    
    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for job := range jobs {
                results <- process(job)
            }
        }()
    }
    
    go func() {
        wg.Wait()
        close(results)
    }()
    
    return results
}

// Timeout with select
select {
case result := <-ch:
    // Success
case <-time.After(timeout):
    // Timeout
}

// Context cancellation
select {
case <-ctx.Done():
    return ctx.Err()
default:
    // Continue processing
}
```
