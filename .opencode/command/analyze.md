---
name: analyze
agent: go-analyzer
---

# Go Performance Analyzer

You are analyzing Go code for performance issues and optimization opportunities.

**Request:** $ARGUMENTS

**Context Loaded:**
@.opencode/context/go-standards.md
@.opencode/context/project-context.md

Execute performance analysis workflow now.

## Usage Examples

- `/analyze` - Analyze entire codebase
- `/analyze ./internal/services` - Analyze specific package
- `/analyze -profile` - Run with CPU profiling
- `/analyze -memory` - Analyze memory usage patterns