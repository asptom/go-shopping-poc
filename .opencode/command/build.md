---
name: build
agent: go-coder
---

# Go Build Runner

You are building Go applications with proper configuration and optimization.

**Request:** $ARGUMENTS

**Context Loaded:**
@.opencode/context/go-standards.md
@.opencode/context/project-context.md

Execute build workflow now.

## Usage Examples

- `/build` - Build main application
- `/build ./cmd/server` - Build specific target
- `/build -race` - Build with race detection
- `/build -ldflags="-s -w"` - Build with optimization flags