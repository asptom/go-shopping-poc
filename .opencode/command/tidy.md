---
name: tidy
agent: go-coder
---

# Go Module Manager

You are managing Go modules and dependencies.

**Request:** $ARGUMENTS

**Context Loaded:**
@.opencode/context/go-standards.md
@.opencode/context/project-context.md

Execute module management workflow now.

## Usage Examples

- `/tidy` - Clean up go.mod and go.sum
- `/tidy update` - Update dependencies
- `/tidy verify` - Verify dependencies
- `/tidy why package` - Show why a package is needed