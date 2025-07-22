# Architecture Specification

## Single Binary

The application will be distributed as a single, self-contained executable with no external runtime dependencies. All functionality is compiled directly into the binary.

**Benefits:**
- Simple deployment - just copy and run
- No dependency conflicts
- Predictable behavior across environments

## Dependencies

**Core Principles:**
- Minimize third-party dependencies
- Prefer standard library solutions
- Audit all dependencies for necessity

**Allowed Dependencies:**
- CLI framework: `github.com/spf13/cobra`

## Project Structure

Follow standard Go project layout conventions:

```
river/
├── cmd/
│   └── river/
│       └── main.go          # Application entry point
├── internal/
│   ├── config/              # Configuration handling
│   ├── core/                # Core business logic
│   └── cli/                 # CLI command implementations
├── pkg/                     # Public packages (if any)
├── specs/                   # Technical specifications
├── go.mod
├── go.sum
├── Makefile                 # Build automation
└── README.md
```

**Key Conventions:**
- `cmd/`: Contains `main` packages for executables
- `internal/`: Private application code
- `pkg/`: Public libraries (only if needed)
- Package names are lowercase, single words
- Interfaces defined in consumer packages
- Explicit error handling, no panic in production code