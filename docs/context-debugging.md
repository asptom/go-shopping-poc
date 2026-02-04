## User

That was a great find.  Describe the steps that I should have used to find that.  I never thought to look for a context change in postgresql.go.  It was not at all obvious to me.  Is there a good way to go about tracing context through a series of function calls?  How should I be thinking about that?

---

## Assistant (Build · kimi-k2.5-free · 17.3s)

_Thinking:_

The user is asking for debugging methodology - how to trace context issues through Go code. This is a great teaching opportunity. Let me think through the systematic approach I used:

1. Start with the error message and trace backwards
2. Look at the call stack from logs
3. Identify where context is created vs where it's used
4. Look for context wrapping patterns (WithTimeout, WithCancel)
5. Understand the lifecycle of deferred functions

Let me provide a structured approach they can use in the future.

Great question! Here's a systematic approach to tracing context issues:

## 1. Start with the Error and Work Backwards

**Error:** `context deadline exceeded` when checking product existence

**Trace backwards from the error:**
```
repository_image.go:23 → calls ProductExists()
repository_query.go:21 → calls r.db.QueryRow()
postgresql.go:157 → QueryRow() returns *sql.Row (but doesn't execute yet)
```

## 2. Map the Context Lifecycle

Create a timeline of context creation/modification:

```
product-loader/main.go:263    ctx := context.Background()  ← Root context
                    ↓
service_admin.go:304          IngestProductsFromCSV(ctx)   ← Same ctx passed through
                    ↓
service_admin.go:418          processProductBatch(ctx)     ← Still same ctx
                    ↓
service_admin.go:489          AddProductImage(ctx)         ← Still same ctx
                    ↓
repository_image.go:16        AddProductImage(ctx)         ← Still same ctx
                    ↓
repository_image.go:23        ProductExists(ctx)           ← Still same ctx
                    ↓
repository_query.go:21        r.db.QueryRow(ctx, ...)      ← ctx passed to DB layer
                    ↓
postgresql.go:146            QueryRow(ctx, ...)            ← HERE'S THE BUG!
                              └─► queryCtx, cancel := context.WithTimeout(ctx, 30s)
                              └─► defer cancel()           ← IMMEDIATE CANCEL!
                              └─► return c.db.QueryRowContext(queryCtx, ...)
                    ↓
postgresql.go:157            returns *sql.Row with DEAD context
                    ↓
repository_query.go:21       .Scan(&exists)                ← FAILS: context cancelled!
```

## 3. Look for These Red Flags

When you see `context deadline exceeded`, check for:

1. **Deferred cancels with lazy evaluation:**
   ```go
   queryCtx, cancel := context.WithTimeout(ctx, timeout)
   defer cancel()  // ⚠️ Dangerous if function returns something that uses ctx later!
   return c.db.QueryRowContext(queryCtx, ...)  // QueryRow is LAZY - doesn't execute immediately
   ```

2. **Functions returning objects that use context later:**
   - `QueryRow()` returns `*sql.Row` - actual query happens on `.Scan()`
   - `Query()` returns `*sql.Rows` - iteration uses context
   - Any function returning a lazy evaluator, iterator, or promise

3. **Context wrapping patterns:**
   ```go
   // GOOD: Blocking call - defer cancel() is fine
   func (c *Client) Exec(ctx context.Context) error {
       ctx, cancel := context.WithTimeout(ctx, timeout)
       defer cancel()
       return c.db.ExecContext(ctx, query)  // Executes immediately, returns error
   }
   
   // BAD: Non-blocking call - defer cancel() kills context before use
   func (c *Client) QueryRow(ctx context.Context) *sql.Row {
       ctx, cancel := context.WithTimeout(ctx, timeout)
       defer cancel()
       return c.db.QueryRowContext(ctx, query)  // Returns immediately, query runs later
   }
   ```

## 4. Use Logging to Trace Context State

Add context deadline logging at key points:

```go
func logContextState(ctx context.Context, location string) {
    deadline, ok := ctx.Deadline()
    log.Printf("[DEBUG] Context at %s: hasDeadline=%v, deadline=%v, remaining=%v",
        location, ok, deadline, time.Until(deadline))
}
```

## 5. The "Deferred Cancel" Rule

**Golden rule:** Only use `defer cancel()` when:
- The function executes the operation immediately (blocking)
- The function doesn't return anything that uses the context later

**When to use parent context directly:**
- Returning lazy evaluators (QueryRow, iterators)
- Returning objects that will perform I/O later
- Functions that return channels or callbacks

## 6. Visual Debugging Technique

Draw arrows showing context flow:

```
caller ──ctx──┐
              ▼
         QueryRow(ctx)
              │
              ├─► queryCtx, cancel := WithTimeout(ctx, 30s)
              │             │
              │             ▼
              │        defer cancel() ◄── Runs when QueryRow returns!
              │             │
              └─► return *sql.Row with queryCtx
              
Later: .Scan() tries to use cancelled context ──► DEADLINE EXCEEDED
```

## 7. Pattern Recognition

Common context bug patterns in Go:

| Pattern | Safe? | Why |
|---------|-------|-----|
| `defer cancel()` in blocking function | ✅ | Context used immediately |
| `defer cancel()` in non-blocking function | ❌ | Context may be used after return |
| Passing ctx to goroutine | ⚠️ | Must be careful about parent cancellation |
| `WithTimeout` + return lazy object | ❌ | Same as defer cancel issue |
| `WithValue` + return | ✅ | Values don't have timeouts |

## 8. Tool: grep for Context Wrappers

When debugging context issues:

```bash
# Find all places context is wrapped with timeout
grep -rn "context.WithTimeout" --include="*.go" | head -20

# Find all defer cancel() calls  
grep -rn "defer cancel()" --include="*.go" | head -20

# Look for QueryRow/Query calls
grep -rn "QueryRow\|Query(" --include="*.go" | grep -v test
```

## Summary Checklist for Context Debugging

1. ✅ Identify where the error occurs (QueryRow.Scan)
2. ✅ Trace back where that context came from (passed through from main)
3. ✅ Look for context modifications in the DB layer (WithTimeout + defer cancel)
4. ✅ Check if the function is blocking or returns lazy object
5. ✅ Verify timing - when does the query actually execute vs when is context cancelled

The key insight: **`QueryRow()` doesn't execute the query - it returns a handle. The query executes when you call `.Scan()`.** So wrapping its context with a timeout + defer cancel is always wrong.

---

