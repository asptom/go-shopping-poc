Here is a **Markdown “LLM context file”** built from a summary of Effective Go (the document at [https://go.dev/doc/effective_go](https://go.dev/doc/effective_go)). You can place this in your Go project (e.g., `LLM_CONTEXT.md`) and use it as the foundational prompt for a Golang-expert agent.

---

# LLM_CONTEXT.md

> *For a Golang expert-agent: read this first, and then use it as the guiding style & idiom reference when generating or refactoring Go code.*

---

## Purpose

This file encapsulates the key idioms, conventions and style guidelines of Go as described in Effective Go. An expert-agent should use it to generate **clear, idiomatic, maintainable** Go code — not simply translate patterns from other languages.

---

## High-Level Principles

* Think *Go*, not Java/C++ translated into Go. The language has distinct properties and idioms. ([Go][1])
* Code should be easy for other Go programmers to understand — follow established conventions (naming, formatting, package structure, etc.). ([Go][1])
* The zero value of types should be useful where possible (i.e., types should be usable immediately with their zero value). ([Go][1])

---

## Formatting & Style

* Use `gofmt` / `go fmt` for formatting; don’t invent alternate indentation or alignment rules. ([Go][1])
* Tabs for indentation; spaces only if absolutely necessary. ([Go][1])
* No need to worry about strict line-length limits; wrap if it improves readability. ([Go][1])
* Prefer simple clean syntax: Go has fewer parentheses than C/Java; don’t overuse parentheses. ([Go][1])

---

## Naming & Comments

### Package names

* Use short, lower-case, single-word names; the import path’s base name is typically the package name. ([Go][1])
* Avoid underscores or mixedCaps in package names. ([Go][1])

### Exported names & getters/setters

* Visibility is determined by the first character being uppercase (exported) or lowercase (unexported). ([Go][1])
* For a field `owner`, the getter should be `Owner()` (not `GetOwner()`) when exporting; setters may be `SetOwner()`. ([Go][1])

### Interface naming

* One‐method interfaces are commonly named with an `-er` suffix: `Reader`, `Writer`, `Closer`, etc. ([Go][1])
* Don’t give a method a canonical name (e.g., `String`, `Read`, `Write`) if it doesn’t match the expected signature / meaning. ([Go][1])

### Commenting

* Use doc comments for exported identifiers: comment begins with the name of the thing being described; sentences should be full sentences ending in a period. ([Rose-Hulman Institute of Technology][2])
* Package comment: a block preceding the `package` clause, describing the package as a whole. ([Rose-Hulman Institute of Technology][2])

---

## Control Structures & Language Constructs

### If, For, Switch

* Use `if err := …; err != nil { … return err }` style; avoid unnecessary `else` when the `if` returns. ([Go][1])
* `for` is Go’s only loop form: `for init; condition; post {}`, `for condition {}`, or `for {}` (infinite). ([Go][1])
* Use `range` when iterating slices, maps, strings, channels. Use blank identifier `_` when ignoring values. ([Go][1])
* `switch` is more general: no expression means `switch true { … }`. No automatic fall-through unless `fallthrough` is used. ([Go][1])

### Defer

* `defer` schedules a call to run just before the surrounding function returns. It’s useful for cleanup (e.g., `f.Close()`). ([Go][1])
* Arguments to the deferred function are evaluated at the time of the `defer`, but the function call runs later. ([Go][1])

### Multiple Return Values

* Functions may return multiple values (e.g., `func Foo() (value, err error)`). Use this for normal result + error. ([Go][1])
* Named result parameters are allowed and can improve clarity. ([Go][1])

---

## Data, Types & Allocation

### Zero Value Usefulness

* Types should be designed so that their zero value is usable (“ready to use” without explicit initialization), where possible. ([Go][1])

### `new` vs `make`

* `new(T)` allocates zeroed storage for a new T and returns a `*T`. ([Go][1])
* `make(T, args…)` is used only for slices, maps and channels; these types need initialization beyond zero value. ([Go][1])

### Arrays, Slices, Maps

* Arrays are values; the size is part of the type; slices are the typical sequence type. ([Go][1])
* Maps hold references; when passing a map to a function, modifications by the function are visible to the caller. ([Go][1])
* Use composite literals to initialize arrays, slices, maps. ([Go][1])

---

## Methods, Interfaces & Embedding

### Pointers vs Values for receivers

* Choose receiver type carefully: value vs pointer depends on method semantics, mutability, interface satisfaction. ([Go][1])

### Interfaces

* Interfaces represent behavior (sets of methods). Use them to abstract implementation details. ([Go][1])
* Don’t add fields solely to satisfy interfaces; embed types when appropriate. (Further embedding discussion is in the doc.)

---

## Concurrency

* Use goroutines and channels to express concurrency; avoid shared state by default (share by communicating, not communicate by sharing). ([Go][1])
* Distinguish between concurrency (structuring independent components) and parallelism (utilizing multiple CPUs). Go is primarily a concurrent language. ([Go][3])

---

## Error Handling & Panics

* Returning `error` values is preferred to panicking; use `panic` and `recover` only in exceptional situations. ([Mathematics at BAS][4])
* Make errors meaningful and descriptive; use `fmt.Errorf` and error wrapping where appropriate. (While not deeply discussed in Effective Go, it is Go idiom.)

---

## When Generating/Refactoring Go Code — Agent Guidelines

* Use idiomatic Go naming: mixedCaps, no underscores for multi-word names; package names short, lower case.
* Respect zero-value usability: avoid unnecessary initialization when zero value suffices.
* Use `defer` for cleanup (file closes, mutex unlocks).
* Use error-checking early and early returns (avoid deep nesting).
* When implementing interfaces, use canonical method names if they mirror standard ones (e.g., `String() string`, `Read(b []byte) (int, error)`).
* Keep functions/methods concise; Go favours simplicity and clarity over complex inheritance or heavy scaffolding.
* Prioritize readability and maintainability: other Go developers should find the code natural.
* For concurrency: favour channel/goroutine patterns over heavy shared-mutable state; avoid over-complicated synchronization.
* Use built-in tools: `go fmt`, `go vet`, `golint` (or `staticcheck`) as part of workflow.
* Comment exported items with doc comments; keep them clear, first sentence starts with the identifier.

---

## Limitations & Notes

* The Effective Go document was written around Go’s early versions and **has not been updated significantly** to cover later language features (modules, generics, newer APIs). ([Go][1])
* This file **does not cover** all facets of Go (e.g., module system, build tools, advanced generics usage, performance optimization). Use other resources when needed.
* When newer language features are used (e.g., generics, `go:embed`, modules), apply analogous idiomatic reasoning: keep things simple, obvious, and consistent.

---

## Summary Table of Key Conventions

| Area                 | Convention                                                                   |
| -------------------- | ---------------------------------------------------------------------------- |
| Formatting           | `go fmt`, tabs for indent, no manual alignment                               |
| Package names        | short, one word, lower-case                                                  |
| Identifiers          | MixedCaps for multi-word names, no underscores                               |
| Getters/Setters      | `Foo()` for getter, `SetFoo()` for setter when appropriate                   |
| Comments             | Doc comment for exported identifiers, start with name, full sentence         |
| Control flow         | Early returns for error handling; avoid unnecessary `else` blocks            |
| Concurrency          | Use goroutines + channels; minimal shared mutable state                      |
| Methods/Interfaces   | Pointer vs value receiver chosen appropriately; interface names end in `-er` |
| Zero-value usability | Types should be usable with zero value if possible                           |
| Error handling       | Return `error`, avoid overusing `panic`                                      |

---

**Instruction to agent:**

> *Before generating or modifying code, internalize this context. Use the conventions above as your baseline. When producing Go code, explain why you’re choosing a pattern (if non-trivial) and verify you’re adhering to idiomatic Go style. Where trade-offs exist, prefer the simpler, more maintainable Go-style path. If a requested feature conflicts with Go idiom, suggest an alternate design aligned with the idioms above.*

---

Feel free to **customise** this file further by adding your project-specific rules (e.g., logging conventions, architecture patterns, dependency injection style, error handling wrapper functions) and integrate it with your LLM workflow.

Would you like me to generate a *template with placeholders* specific to your project (package names, services, modules) so you can drop it into your repo directly?

[1]: https://go.dev/doc/effective_go "Effective Go - The Go Programming Language"
[2]: https://www.rose-hulman.edu/class/cs/csse403/201110/SlidePDFs/EffectiveGo.pdf?utm_source=chatgpt.com "EFFECTIVE GO"
[3]: https://go.dev/doc/effective_go?utm_source=chatgpt.com "Effective Go"
[4]: https://www.math.bas.bg/softeng/bantchev/place/go/effective-go.pdf?utm_source=chatgpt.com "effective-go.pdf"
