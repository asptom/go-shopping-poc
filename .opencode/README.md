# Go Configuration - Simplified & Specialized

## Overview

This `` directory contains a simplified, Go-specific configuration that focuses on efficiency and specialization. This setup uses an architect-style primary agent that delegates to specialized subagents for optimal focus and context maintenance.

## Configuration Structure

```
.opencode/                      # Root for opencode configuration
├── agent/
│   └── go-agent.md             # Primary architect & task manager
├── subagents/                  # Specialized subagents
│   ├── go-planner.md           # Task planning and breakdown
│   ├── go-coder.md             # Code implementation
│   ├── go-tester.md            # Testing specialist
│   ├── go-reviewer.md          # Code review and quality
│   ├── go-docs.md              # Documentation specialist
│   └── go-analyzer.md          # Analysis and optimization
├── context/                     # Knowledge base
│   ├── go-standards.md         # Go-specific patterns and conventions
│   └── project-context.md      # Your project-specific configuration
└── commands/                    # Common Go operations
    ├── test.md                 # Go test runner
    ├── build.md                # Go build commands
    ├── tidy.md                 # Go module management
    ├── analyze.md              # Performance analysis
    └── review.md               # Code review
```

## Approach

### Simplified Architecture
- **Primary Agent**: Single architect-style agent (go-agent) 
- **Specialized Subagents**: 6 focused specialists 
- **Go-Specific**: All patterns and examples are Go-focused
- **Efficiency**: Minimal overhead, direct delegation
- **Context Loading**: Simple, deterministic loading
- **Approval Process**: Streamlined vs multi-stage workflow

### Go-Focused Standards
- **Error Handling**: Go-specific error patterns
- **Concurrency**: Goroutines, channels, and sync patterns
- **Package Structure**: Go project layout and conventions
- **Testing**: Go testing package and table-driven tests
- **Performance**: Go profiling and optimization

## Usage

### Starting the Go Agent
```bash
# Load the Go agent configuration
opencode --agent /agent/go-agent.md
```

### Available Commands
```bash
/test [package] [flags]          # Run Go tests
/build [target] [flags]          # Build Go applications
/tidy [options]                  # Manage Go modules
/analyze [package] [flags]       # Performance analysis
/review [package] [flags]        # Code review
```

### Workflow Examples

#### Feature Development
```
User: "Create a REST API for user management"
Go Agent: 
1. Delegates to go-planner → Creates implementation plan
2. Delegates to go-coder → Implements the API
3. Delegates to go-tester → Writes comprehensive tests
4. Delegates to go-reviewer → Reviews code quality
5. Delegates to go-docs → Creates API documentation
```

#### Performance Optimization
```
User: "Analyze performance issues in the user service"
Go Agent:
1. Delegates to go-analyzer → Runs performance profiling
2. Delegates to go-coder → Implements optimizations
3. Delegates to go-tester → Validates improvements
4. Delegates to go-reviewer → Ensures quality
```

#### Code Review
```
User: "Review this PR for security issues"
Go Agent:
1. Delegates to go-reviewer → Comprehensive code review
2. Delegates to go-analyzer → Security and performance analysis
3. Provides consolidated report
```

## Benefits

### 1. Focus & Specialization
- Each subagent maintains deep context in their domain
- No context switching between different types of work
- Higher quality output due to specialization

### 2. Efficiency
- Minimal overhead and configuration
- Direct delegation without complex workflows
- Fast startup and execution

### 3. Go-Specific
- All patterns and examples are Go-idiomatic
- Go-specific best practices and conventions
- Relevant error handling and concurrency patterns

### 4. Maintainability
- Clear separation of concerns
- Easy to extend and modify
- Simple file structure

### 5. Quality
- Specialists ensure best practices in their domain
- Comprehensive coverage of all development phases
- Built-in quality gates and validation

## Customization

### Project-Specific Configuration
Edit `/context/project-context.md` to include:
- Your project's technology stack
- Custom patterns and conventions
- Performance requirements
- Security requirements
- Deployment configuration

### Adding New Subagents
1. Create new subagent file in `/subagents/`
2. Update go-agent.md to include the new subagent
3. Add relevant commands if needed

### Modifying Standards
Edit `/context/go-standards.md` to:
- Add project-specific patterns
- Update coding conventions
- Include additional best practices

## Migration from standard .opencode configuration

1. **Copy Project Context**: Move project-specific patterns to `project-context.md`
2. **Update Standards**: Customize `go-standards.md` with your conventions
3. **Test Workflow**: Try the simplified workflow with a small feature

## Getting Started

1. **Explore the Configuration**: Read through all files to understand the structure
2. **Customize Project Context**: Update `project-context.md` with your project details
3. **Test the Agent**: Try a simple task to validate the setup
4. **Adopt the Workflow**: Start using the Go agent for new development

This simplified configuration provides all the benefits of specialized AI assistance while maintaining the efficiency and focus needed for productive Go development.