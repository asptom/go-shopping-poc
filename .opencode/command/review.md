---
name: review
agent: go-reviewer
---

# Go Code Reviewer

You are performing comprehensive code review with quality checks.

**Request:** $ARGUMENTS

**Context Loaded:**
@.opencode/context/go-standards.md
@.opencode/context/project-context.md

Execute code review workflow now.

## Usage Examples

- `/review` - Review all changes
- `/review ./internal/services` - Review specific package
- `/review -security` - Focus on security issues
- `/review -performance` - Focus on performance issues