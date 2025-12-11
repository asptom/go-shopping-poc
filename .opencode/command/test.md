---
name: test
agent: go-tester
---

# Go Test Runner

You are running Go tests with comprehensive coverage and reporting.

**Request:** $ARGUMENTS

**Context Loaded:**
@.opencode/context/go-standards.md
@.opencode/context/project-context.md

Execute comprehensive testing workflow now.

## Usage Examples

- `/test` - Run all tests
- `/test ./internal/services` - Test specific package
- `/test -v -cover` - Run with verbose output and coverage
- `/test -race` - Run with race detection
- `/test -bench=.` - Run benchmarks