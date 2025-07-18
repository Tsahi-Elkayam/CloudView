# CloudView

A plugin-based, cloud-agnostic CLI tool for managing and monitoring resources across multiple cloud providers.

## Overview

CloudView provides a unified interface for:
- **Resource Discovery**: Find and inventory resources across cloud providers
- **Cost Management**: Track and analyze cloud spending
- **Security Monitoring**: Identify security findings and compliance issues
- **Alert Management**: Monitor and manage alerts from multiple clouds
- **Multi-Cloud Operations**: Perform operations across providers with a single tool

## Features

- 🔌 **Plugin-Based Architecture**: Extensible design for adding new cloud providers
- ☁️ **Multi-Cloud Support**: Unified interface for AWS, GCP, Azure (planned)
- 📊 **Rich Output Formats**: Table, JSON, and Excel export options
- 🔍 **Advanced Filtering**: Filter resources by type, region, tags, and more
- 💰 **Cost Analytics**: Detailed cost analysis and forecasting
- 🔒 **Security Insights**: Security findings and compliance reporting
- ⚡ **High Performance**: Parallel execution and intelligent caching
- 🛠️ **Easy Configuration**: YAML configuration with environment variable support

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/Tsahi-Elkayam/cloudview.git
cd cloudview

# Build the binary
make build

# Install to your PATH
make install
```

### Basic Usage

```bash
# Show help
cloudview --help

# Show version
cloudview version

# List AWS resources (requires AWS configuration)
cloudview inventory --provider aws

# List specific resource types
cloudview inventory --provider aws --type ec2
cloudview inventory --provider aws --type s3

# Filter by region
cloudview inventory --provider aws --region us-east-1

# Filter by multiple criteria
cloudview inventory --provider aws --type ec2 --region us-east-1,us-west-2 --tag Environment=production

# Output in different formats
cloudview inventory --provider aws --output json
cloudview inventory --provider aws --output yaml
```

## AWS Configuration

CloudView supports multiple AWS authentication methods:

### Method 1: AWS Profile (Recommended)
```yaml
# ~/.cloudview.yaml
providers:
  aws:
    enabled: true
    profile: "your-profile-name"
    region: "us-east-1"
    regions:
      - "us-east-1"
      - "us-west-2"
```

### Method 2: Environment Variables
```bash
export AWS_PROFILE=your-profile
export AWS_REGION=us-east-1
export CLOUDVIEW_AWS_ENABLED=true
```

### Method 3: Access Keys (Not recommended for production)
```yaml
# ~/.cloudview.yaml
providers:
  aws:
    enabled: true
    access_key_id: "your-access-key"
    secret_access_key: "your-secret-key"
    region: "us-east-1"
```

## Development Status

CloudView is currently under active development. Here's the implementation roadmap:

### ✅ Milestone 1: Foundation (Complete)
- [x] Core CLI structure with Cobra
- [x] Plugin architecture and registry
- [x] Basic data models and interfaces
- [x] Logging and configuration framework
- [x] Build and development tooling

### ✅ Milestone 2: AWS Foundation (Complete)
- [x] AWS provider implementation
- [x] EC2 and S3 resource discovery
- [x] Basic inventory command
- [x] AWS authentication (profiles, access keys, IAM roles)
- [x] Multi-region support
- [x] Resource filtering and querying

### 🚧 Milestone 3: Core Business Logic (Next)
- [ ] Multi-provider execution
- [ ] Advanced filtering and aggregation
- [ ] Table, JSON, YAML output formats
- [ ] Performance optimizations

### 📅 Future Milestones
- **Milestone 4**: Advanced AWS features (Cost, Security, Lambda)
- **Milestone 5**: Enhanced output and caching
- **Milestone 6**: Comprehensive testing and documentation
- **Milestone 7**: Multi-provider preparation

## Architecture

CloudView follows a plugin-based architecture that enables:

```
┌─────────────────────────────────────┐
│           CLI Interface             │
├─────────────────────────────────────┤
│         Command Router              │
├─────────────────────────────────────┤
│        Plugin Registry              │
├─────────────────────────────────────┤
│   Cloud Provider Plugins            │
│   ┌─────┬─────┬─────────┐           │
│   │ AWS │ GCP │  Azure  │           │
│   └─────┴─────┴─────────┘           │
├─────────────────────────────────────┤
│      Core Business Logic            │
└─────────────────────────────────────┘
```

Key benefits:
- **Zero Core Changes**: Adding new providers requires no changes to core logic
- **Plugin Independence**: Each provider is self-contained
- **Parallel Execution**: Multiple providers can be queried simultaneously
- **Error Isolation**: Failures in one provider don't affect others

## Project Structure

```
cloudview/
├── cmd/                    # CLI commands and main entry point
├── pkg/                    # Public packages
│   ├── providers/          # Cloud provider plugins
│   ├── models/             # Data models
│   ├── core/               # Core business logic
│   ├── config/             # Configuration management
│   ├── cache/              # Caching layer
│   ├── output/             # Output formatters
│   └── utils/              # Utility functions
├── internal/               # Internal packages
├── test/                   # Test files and fixtures
├── configs/                # Configuration files
├── docs/                   # Documentation
└── scripts/                # Build and deployment scripts
```

## Development

### Prerequisites

- Go 1.21 or later
- Make (for build automation)
- Git

### Building

```bash
# Download dependencies
make deps

# Format and vet code
make fmt vet

# Run tests
make test

# Build binary
make build

# Build for all platforms
make build-all
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run benchmarks
make benchmark
```

### Development Tools

```bash
# Run in development mode
make dev

# Generate mocks (requires mockgen)
make mocks

# Lint code (requires golangci-lint)
make lint

# Security scan (requires gosec)
make security
```

## Configuration

CloudView uses YAML configuration files. Create a `.cloudview.yaml` file in your home directory:

```yaml
# Currently minimal - will expand with AWS support
providers:
  aws:
    enabled: false  # Will be true in Milestone 2
    regions: []
    config: {}

cache:
  enabled: true
  ttl: 300s
  storage: memory

output:
  format: table
  colors: true

logging:
  level: info
  format: text
```

## Environment Variables

- `CLOUDVIEW_LOG_LEVEL`: Set log level (trace, debug, info, warn, error)
- `CLOUDVIEW_LOG_FORMAT`: Set log format (text, json)
- `CLOUDVIEW_LOG_COLOR`: Enable/disable colored logs (true, false)

## Contributing

We welcome contributions! Here's how to get started:

1. **Fork the repository**
2. **Create a feature branch**: `git checkout -b feature/amazing-feature`
3. **Make your changes**: Follow the existing code style
4. **Add tests**: Ensure your changes are tested
5. **Run the test suite**: `make test`
6. **Commit your changes**: `git commit -m 'Add amazing feature'`
7. **Push to the branch**: `git push origin feature/amazing-feature`
8. **Open a Pull Request**

### Development Guidelines

- Follow Go best practices and idioms
- Add tests for new functionality
- Update documentation as needed
- Use conventional commit messages
- Ensure CI passes before submitting PR

## Roadmap

### Short Term (Next 3 Months)
- Complete AWS provider implementation
- Add basic resource inventory functionality
- Implement table and JSON output formats
- Add comprehensive test coverage

### Medium Term (3-6 Months)
- Add cost management features
- Implement security scanning
- Add caching and performance optimizations
- Excel export functionality

### Long Term (6+ Months)
- GCP provider support
- Azure provider support
- Advanced filtering and querying
- Plugin marketplace for community providers

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- 📚 **Documentation**: [docs/](docs/)
- 🐛 **Bug Reports**: [GitHub Issues](https://github.com/Tsahi-Elkayam/cloudview/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/Tsahi-Elkayam/cloudview/discussions)

## Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) for CLI framework
- Logging powered by [Logrus](https://github.com/sirupsen/logrus)
- Configuration management with [Viper](https://github.com/spf13/viper)

---

**Note**: CloudView is currently in active development. Features and APIs may change as we progress through the development milestones. See the [development status](#development-status) section for current progress.