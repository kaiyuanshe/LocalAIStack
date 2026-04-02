# LocalAIStack Go Development Setup

## Prerequisites

- Go 1.23 or higher
- Git
- Make (for build automation)
- Docker (optional, for container-based execution)

## Installation

### Clone the Repository

```bash
git clone https://github.com/zhuangbiaowei/LocalAIStack.git
cd stack
```

### Install Dependencies

```bash
make deps
# or
go mod download
```

## Development

### Build Binaries

Build for current platform:
```bash
make build
```

Build server binary only:
```bash
make build-server
```

Build CLI binary only:
```bash
make build-cli
```

Build for all platforms (Linux amd64/arm64):
```bash
make build-all
```

Binaries will be created in the `build/` directory:
- `build/localaistack-server` - Server binary
- `build/localaistack` - CLI binary

### Run Tests

```bash
make test
```

Run tests with coverage report:
```bash
make test-coverage
```

### Code Quality

Format code:
```bash
make fmt
```

Run go vet:
```bash
make vet
```

Run linter (requires golangci-lint):
```bash
make lint
```

Install golangci-lint:
```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

### Run Development Server

```bash
make run-server
```

Or directly:
```bash
./build/localaistack-server
```

### Run CLI

```bash
make run-cli
# or
./build/localaistack --help
```

## Project Structure

```
.
├── cmd/                    # Main applications
│   ├── server/            # Server binary entry point
│   └── cli/               # CLI binary entry point
├── internal/               # Private application code
│   ├── api/               # HTTP API server
│   ├── cli/               # CLI commands and root command
│   ├── config/            # Configuration management
│   ├── control/           # Control layer (hardware, policies, state)
│   ├── module/            # Module system
│   ├── model/             # Model management
│   └── runtime/           # Runtime execution layer
├── pkg/                   # Public libraries
│   ├── hardware/          # Hardware detection
│   ├── logging/           # Logging utilities
│   └── utils/             # Utility functions
├── web/                   # Web UI assets
│   ├── static/            # Static files
│   └── templates/         # HTML templates
├── configs/               # Configuration files
│   ├── config.yaml        # Default configuration
│   └── policies.yaml      # Hardware capability policies
├── docs/                  # Documentation
├── scripts/               # Build and deployment scripts
├── build/                 # Build output (gitignored)
├── Makefile              # Build automation
├── go.mod               # Go module definition
└── .gitignore          # Git ignore rules
```

## Configuration

Default configuration is in `configs/config.yaml`.

You can override configuration by:
1. Command-line flags
2. Environment variables
3. Custom config file with `--config` flag

### CLI Configuration

```bash
# Use custom config file
localaistack --config /path/to/config.yaml module install ollama

# Verbose output
localaistack --verbose service start ollama
```

### Server Configuration

Server uses the same configuration file. Edit `configs/config.yaml` to customize:
- Server host and port
- Logging level and format
- Data directories
- Runtime settings

## Key Components

### Control Layer (`internal/control/`)
- Hardware detection and profiling
- Capability policy evaluation
- State management and reconciliation

### Module System (`internal/module/`)
- Module manifest parsing
- Dependency resolution
- Lifecycle management

### Runtime Layer (`internal/runtime/`)
- Container-based execution (Docker)
- Native execution management
- Process lifecycle

### API Layer (`internal/api/`)
- RESTful HTTP API
- Health checks
- Status endpoints

## Cross-Platform Compilation

### Linux

```bash
# amd64
GOOS=linux GOARCH=amd64 make build

# arm64
GOOS=linux GOARCH=arm64 make build
```

### macOS

```bash
# amd64
GOOS=darwin GOARCH=amd64 make build

# arm64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 make build
```

### Windows

```bash
GOOS=windows GOARCH=amd64 make build
```

On Windows, `make build` for the current platform now emits `build/las.exe` and `build/las-server.exe`.

## Installation

### System-wide Installation

```bash
make install
```

This installs binaries to `/usr/local/bin/`.

### Uninstall

```bash
make uninstall
```

## Development Workflow

1. **Create a feature branch**
   ```bash
   git checkout -b feature/my-feature
   ```

2. **Make changes and test**
   ```bash
   make test
   make fmt
   make vet
   ```

3. **Commit changes**
   ```bash
   git add .
   git commit -m "Add my feature"
   ```

4. **Build and verify**
   ```bash
   make build
   make run-server
   ```

5. **Push and create PR**
   ```bash
   git push origin feature/my-feature
   ```

## Troubleshooting

### Build Errors

If you encounter build errors:
```bash
# Clean and rebuild
make clean
make tidy
make build
```

### Dependency Issues

```bash
# Refresh dependencies
make tidy

# Update all dependencies
go get -u ./...
```

### Permission Denied

If you get permission denied errors running binaries:
```bash
# Make binaries executable
chmod +x build/localaistack-server
chmod +x build/localaistack
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and linting
5. Submit a pull request

Please follow the existing code style and add tests for new features.

## License

Apache 2.0 (or MIT, TBD)

## Support

For issues and questions:
- GitHub Issues: https://github.com/zhuangbiaowei/LocalAIStack/issues
- Documentation: See `docs/` directory

---

**Version**: 0.1.0-dev
**Last Updated**: 2026-01-23
