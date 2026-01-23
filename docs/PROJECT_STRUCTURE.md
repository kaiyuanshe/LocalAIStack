# LocalAIStack Go Project Structure

## Directory Tree

```
LocalAIStack/
├── cmd/                        # Main application entry points
│   ├── server/                 # Server binary
│   │   └── main.go           # Server entry point
│   └── cli/                   # CLI binary
│       └── main.go           # CLI entry point
│
├── internal/                   # Private application code (not importable by other projects)
│   ├── api/                   # HTTP API server
│   │   └── server.go        # REST API implementation
│   │
│   ├── cli/                   # CLI commands and root command
│   │   ├── root.go          # CLI root command and initialization
│   │   └── commands.go      # Command implementations (module, service, model, system)
│   │
│   ├── config/                 # Configuration management
│   │   └── config.go        # Config structures and loading
│   │
│   ├── control/                # Control layer (core logic)
│   │   └── control.go       # Hardware detection, policy evaluation, state management
│   │
│   ├── module/                 # Module system
│   │   └── types.go         # Module manifest and type definitions
│   │
│   ├── model/                  # Model management
│   │   └── (to be implemented)
│   │
│   └── runtime/                # Runtime execution layer
│       └── (to be implemented)
│
├── pkg/                       # Public libraries (can be imported by external projects)
│   ├── hardware/              # Hardware detection utilities
│   │   └── detector.go       # Hardware detection interface and implementation
│   │
│   ├── logging/               # Logging utilities
│   │   └── logging.go       # Logging setup and configuration
│   │
│   └── utils/                 # Utility functions
│       └── (to be implemented)
│
├── web/                       # Web UI assets
│   ├── static/                # Static files (CSS, JS, images)
│   │   └── (to be implemented)
│   │
│   └── templates/             # HTML templates
│       └── (to be implemented)
│
├── configs/                   # Configuration files
│   ├── config.yaml           # Default server configuration
│   └── policies.yaml         # Hardware capability policies
│
├── docs/                      # Documentation
│   ├── architecture.md        # System architecture
│   ├── modules.md            # Module system specification
│   ├── runtime.md            # Runtime execution model
│   ├── policies.md           # Hardware capability policies
│   ├── features.md           # Feature list (English)
│   ├── features.cn.md        # Feature list (Chinese)
│   ├── aog_research.md      # Intel AOG integration research
│   ├── GO_SETUP.md          # Go development setup guide
│   └── PROJECT_STRUCTURE.md # This file
│
├── scripts/                   # Build and deployment scripts
│   └── (to be implemented)
│
├── build/                     # Build output (gitignored)
│   ├── localaistack-server  # Compiled server binary
│   └── localaistack       # Compiled CLI binary
│
├── go.mod                    # Go module definition
├── go.sum                    # Go dependency checksums
├── Makefile                  # Build automation
├── .gitignore               # Git ignore patterns
├── README.md                # Project README (English)
└── README.cn.md             # Project README (Chinese)
```

## Key Components

### Entry Points (`cmd/`)

- **`cmd/server/main.go`**: Main server binary that initializes all layers and starts the API server
- **`cmd/cli/main.go`**: CLI binary that provides command-line interface for system management

### Internal Packages (`internal/`)

#### `internal/api/`
- HTTP REST API server
- Provides health checks and status endpoints
- Handles API requests from Web UI and external clients

#### `internal/cli/`
- CLI command structure using Cobra
- Root command initialization
- Subcommands: module, service, model, system

#### `internal/config/`
- Configuration structures
- Default configuration values
- Configuration loading logic

#### `internal/control/`
- Core control layer logic
- Hardware detection coordination
- Policy evaluation
- State management
- Module lifecycle orchestration

#### `internal/module/`
- Module manifest definitions
- Module state machine
- Type definitions for modules

### Public Packages (`pkg/`)

#### `pkg/hardware/`
- Hardware detection interface
- Native hardware detector implementation
- Hardware profile structures (CPU, GPU, Memory, Storage)

#### `pkg/logging/`
- Logging setup using zerolog
- Log level configuration
- Output format configuration

### Configuration (`configs/`)

#### `config.yaml`
- Server configuration (host, port, timeouts)
- Logging configuration (level, format, output)
- Control layer configuration (data directory, policies)
- Storage configuration (model, cache, download directories)
- Runtime configuration (Docker, native execution modes)

#### `policies.yaml`
- Hardware capability tier definitions
- Allowed/denied features per tier
- Runtime availability per tier

## Build System

### Makefile Targets

```bash
make build          # Build all binaries
make build-server   # Build server only
make build-cli      # Build CLI only
make build-all      # Build for all platforms
make test           # Run tests
make test-coverage  # Run tests with coverage
make clean          # Clean build artifacts
make fmt            # Format code
make vet            # Run go vet
make lint           # Run golangci-lint
make deps           # Download dependencies
make tidy           # Tidy go.mod
make run-server     # Run server
make run-cli        # Run CLI
```

### Cross-Platform Build

```bash
# Linux amd64
GOOS=linux GOARCH=amd64 make build

# Linux arm64
GOOS=linux GOARCH=arm64 make build

# macOS amd64
GOOS=darwin GOARCH=amd64 make build

# macOS arm64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 make build

# Windows
GOOS=windows GOARCH=amd64 make build
```

## Module Organization

### Internal vs Public

- **`internal/`**: Private code that cannot be imported by other Go modules. Used for application-specific logic.
- **`pkg/`**: Public code that can be imported by other projects. Used for reusable libraries.

### Command Pattern

Each command in `cmd/` is a self-contained binary:
- `localaistack-server`: Runs the control layer and API server
- `localaistack`: CLI for system management

## Configuration Layering

Configuration is loaded in order of precedence (highest to lowest):
1. Command-line flags
2. Environment variables (`LOCALAISTACK_*`)
3. Configuration file (`--config`, `$LOCALAISTACK_CONFIG`, or default paths)
4. Default values in code

Default search paths include:
- `$HOME/.localaistack/config.yaml`
- `./configs/config.yaml`
- `./config.yaml`
- `/etc/localaistack/config.yaml`

## Dependency Management

Dependencies are managed via `go.mod`:
```bash
go mod download    # Download dependencies
go mod tidy        # Clean up dependencies
go get -u ./...    # Update all dependencies
```

## Testing

Tests should be placed alongside the code they test:
```
internal/api/
  ├── server.go
  └── server_test.go

pkg/hardware/
  ├── detector.go
  └── detector_test.go
```

Run tests:
```bash
make test
```

## Future Extensions

### Planned Packages

- `internal/runtime/docker.go` - Docker runtime integration
- `internal/runtime/native.go` - Native process execution
- `internal/model/registry.go` - Model registry and management
- `internal/control/policy.go` - Policy evaluation engine
- `internal/control/state.go` - State persistence and reconciliation
- `pkg/semver/` - Semantic versioning utilities
- `pkg/download/` - Download manager for models
- `pkg/cache/` - Caching utilities

### Web UI

The `web/` directory will contain:
- Static assets (CSS, JavaScript, images)
- HTML templates for server-side rendering
- Build scripts for frontend assets

## Development Workflow

1. Create feature branch
2. Make changes
3. Run tests: `make test`
4. Format code: `make fmt`
5. Check code quality: `make vet` and `make lint`
6. Build: `make build`
7. Test locally: `make run-server` or `make run-cli`
8. Commit and push
9. Create pull request

## Notes

- Go version: 1.23+
- All binaries are statically linked for easy distribution
- Configuration uses YAML format for readability
- Logging uses zerolog for structured, zero-allocation logging
- API uses standard library `net/http` for simplicity

---

**Document Version**: 1.0
**Last Updated**: 2026-01-23
