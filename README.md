# Actionlint MCP Server

[![Go Version](https://img.shields.io/github/go-mod/go-version/hongkongkiwi/actionlint-mcp)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/hongkongkiwi/actionlint-mcp)](https://goreportcard.com/report/github.com/hongkongkiwi/actionlint-mcp)
[![CI Status](https://github.com/hongkongkiwi/actionlint-mcp/workflows/PR%20Checks/badge.svg)](https://github.com/hongkongkiwi/actionlint-mcp/actions)
[![Release](https://img.shields.io/github/release/hongkongkiwi/actionlint-mcp.svg)](https://github.com/hongkongkiwi/actionlint-mcp/releases/latest)
[![Docker Pulls](https://img.shields.io/docker/pulls/hongkongkiwi/actionlint-mcp)](https://hub.docker.com/r/hongkongkiwi/actionlint-mcp)
[![codecov](https://codecov.io/gh/hongkongkiwi/actionlint-mcp/branch/main/graph/badge.svg)](https://codecov.io/gh/hongkongkiwi/actionlint-mcp)

An MCP (Model Context Protocol) server that exposes [actionlint](https://github.com/rhysd/actionlint) functionality for linting GitHub Actions workflow files. This allows AI assistants like Claude to validate and check GitHub Actions workflows for errors, best practices, and security issues.

## 🚀 Features

- **`lint_workflow`**: Lint a single GitHub Actions workflow file or content
- **`check_all_workflows`**: Check all workflow files in a directory
- **Real-time validation** of workflow syntax and semantics
- **Security scanning** for common vulnerabilities and misconfigurations
- **Best practices enforcement** for GitHub Actions workflows
- **Shell script validation** with shellcheck integration
- **Python code validation** with pyflakes integration
- **Expression syntax checking** for GitHub Actions expressions
- **Runner availability validation** for self-hosted runners
- **Action version checking** for outdated or insecure actions
- **Matrix job validation** for complex workflow matrices
- **Reusable workflow support** with input/output validation

## 📦 Installation

### Prerequisites

- Go 1.21+ installed (only for building from source)
- Optional: `shellcheck` for shell script validation
- Optional: `pyflakes` for Python code validation

### 🎯 Quick Install (Recommended)

#### Using the install script (macOS/Linux)

```bash
# Install to /usr/local/bin (requires sudo)
curl -sSfL https://raw.githubusercontent.com/hongkongkiwi/actionlint-mcp/main/install.sh | sudo sh

# Install to custom directory
curl -sSfL https://raw.githubusercontent.com/hongkongkiwi/actionlint-mcp/main/install.sh | sh -s -- -b ~/.local/bin

# Install specific version
curl -sSfL https://raw.githubusercontent.com/hongkongkiwi/actionlint-mcp/main/install.sh | sh -s -- -b /usr/local/bin v1.0.0
```

#### Download pre-built binaries

Download the latest release for your platform from the [releases page](https://github.com/hongkongkiwi/actionlint-mcp/releases).

Available platforms:
- **Linux**: amd64, arm64
- **macOS**: amd64 (Intel), arm64 (Apple Silicon)
- **Windows**: amd64

### Install from source

```bash
# Clone the repository
git clone https://github.com/hongkongkiwi/actionlint-mcp.git
cd actionlint-mcp

# Install dependencies
go mod download

# Build the binary
make build

# Or install to /usr/local/bin
make install
```

### Install with go install

```bash
go install github.com/hongkongkiwi/actionlint-mcp@latest
```

### 🐳 Docker

```bash
# Run with Docker Hub image
docker run -it --rm hongkongkiwi/actionlint-mcp:latest

# Run with GitHub Container Registry
docker run -it --rm ghcr.io/hongkongkiwi/actionlint-mcp:latest

# Run with volume mount for local workflows
docker run -it --rm -v $(pwd):/workspace hongkongkiwi/actionlint-mcp:latest

# Run specific version
docker run -it --rm hongkongkiwi/actionlint-mcp:v1.0.0
```

## 🔐 Security & Verification

### Binary Verification

All release binaries include SHA256 checksums for verification. The checksums are included in the release notes.

```bash
# Download binary and verify checksum
VERSION=v1.0.0
PLATFORM=linux-amd64
wget https://github.com/hongkongkiwi/actionlint-mcp/releases/download/${VERSION}/actionlint-mcp-${VERSION}-${PLATFORM}.tar.gz
wget https://github.com/hongkongkiwi/actionlint-mcp/releases/download/${VERSION}/checksums.txt

# Verify checksum
sha256sum -c checksums.txt 2>&1 | grep OK
```

### Security Scanning

This project uses multiple security scanning tools in CI/CD:
- **Semgrep**: Static analysis for security vulnerabilities
- **Trivy**: Container and filesystem vulnerability scanning
- **Dependabot**: Automated dependency updates
- **CodeQL**: GitHub's semantic code analysis

## 🔧 Configuration & Integration

### Claude Desktop

Add to your Claude Desktop configuration:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
**Linux**: `~/.config/claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "actionlint": {
      "command": "/usr/local/bin/actionlint-mcp",
      "env": {
        "SHELLCHECK_COMMAND": "shellcheck",
        "PYFLAKES_COMMAND": "pyflakes"
      }
    }
  }
}
```

### VS Code with Continue

Add to your Continue configuration (`~/.continue/config.json`):

```json
{
  "models": [
    {
      "model": "claude-3-5-sonnet",
      "provider": "anthropic",
      "mcpServers": {
        "actionlint": {
          "command": "/usr/local/bin/actionlint-mcp",
          "env": {
            "SHELLCHECK_COMMAND": "shellcheck",
            "PYFLAKES_COMMAND": "pyflakes"
          }
        }
      }
    }
  ]
}
```

### Neovim with avante.nvim

Add to your avante.nvim configuration:

```lua
require('avante').setup({
  mcp_servers = {
    actionlint = {
      command = "/usr/local/bin/actionlint-mcp",
      env = {
        SHELLCHECK_COMMAND = "shellcheck",
        PYFLAKES_COMMAND = "pyflakes"
      }
    }
  }
})
```

### Zed

Add to your Zed configuration (`~/.config/zed/settings.json`):

```json
{
  "language_models": {
    "mcp_servers": {
      "actionlint": {
        "command": "/usr/local/bin/actionlint-mcp",
        "env": {
          "SHELLCHECK_COMMAND": "shellcheck",
          "PYFLAKES_COMMAND": "pyflakes"
        }
      }
    }
  }
}
```

## 💡 Usage Examples

Once configured, your AI assistant can help you with:

### Lint a specific workflow file

Ask the assistant to:
- "Check my GitHub Actions workflow for errors"
- "Validate .github/workflows/ci.yml"
- "Find issues in my deployment workflow"

### Check all workflows in a project

Ask the assistant to:
- "Check all my GitHub Actions workflows"
- "Validate all workflows in the project"
- "Find issues in any workflow files"

## 🛠️ MCP Tools API

### `lint_workflow`

Lints a single GitHub Actions workflow file.

**Parameters:**
- `file_path` (string): Path to the workflow file to lint
- `content` (string): Content of the workflow file (if file_path not provided)

**Returns:**
```json
{
  "errors": [
    {
      "message": "undefined variable \"UNDEFINED_VAR\"",
      "line": 23,
      "column": 14,
      "kind": "expression",
      "severity": "error"
    }
  ],
  "valid": false,
  "file_path": ".github/workflows/ci.yml"
}
```

### `check_all_workflows`

Checks all GitHub Actions workflow files in a directory.

**Parameters:**
- `directory` (string, optional): Directory to search (defaults to `.github/workflows`)

**Returns:**
```json
{
  "total_files": 3,
  "files_with_errors": 1,
  "total_errors": 2,
  "results": {
    ".github/workflows/ci.yml": {
      "errors": [...],
      "valid": false,
      "file_path": ".github/workflows/ci.yml"
    }
  }
}
```

## ⚙️ Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SHELLCHECK_COMMAND` | Path to shellcheck binary for shell script validation | `shellcheck` |
| `PYFLAKES_COMMAND` | Path to pyflakes binary for Python code validation | `pyflakes` |
| `LOG_LEVEL` | Logging verbosity (debug, info, warn, error) | `info` |
| `MCP_TIMEOUT` | Timeout for MCP operations in seconds | `30` |

## 🧪 Development

### Running tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific test
go test -v -run TestLintWorkflow
```

### Linting

```bash
# Run Go linters
make lint

# Format code
make fmt
```

### Pre-commit hooks

Install pre-commit hooks for automatic code quality checks:

```bash
pip install pre-commit
pre-commit install

# Run hooks manually
pre-commit run --all-files
```

### Building for multiple platforms

```bash
# Build for all platforms
make build-all

# Build for specific platform
GOOS=linux GOARCH=amd64 make build
GOOS=darwin GOARCH=arm64 make build
GOOS=windows GOARCH=amd64 make build
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for your changes
4. Ensure all tests pass (`make test`)
5. Run linters (`make lint`)
6. Commit your changes (`git commit -m 'Add some amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

### Development Guidelines

- Follow Go best practices and idioms
- Write unit tests for new functionality
- Update documentation as needed
- Ensure CI/CD pipeline passes
- Add integration tests for MCP protocol changes

## 🚀 Roadmap

- [ ] Support for GitHub Enterprise Server
- [ ] Custom rule configuration
- [ ] VSCode extension with inline linting
- [ ] Web UI for standalone usage
- [ ] Integration with GitHub Apps
- [ ] Support for composite actions linting
- [ ] Workflow cost estimation
- [ ] Performance profiling for workflows

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [actionlint](https://github.com/rhysd/actionlint) - The powerful linter for GitHub Actions
- [MCP SDK for Go](https://github.com/modelcontextprotocol/go-sdk) - Model Context Protocol SDK
- [shellcheck](https://github.com/koalaman/shellcheck) - Shell script static analysis tool
- [pyflakes](https://github.com/PyCQA/pyflakes) - Python code checker

## 💬 Support

If you encounter any issues or have questions:

- [Open an issue](https://github.com/hongkongkiwi/actionlint-mcp/issues) on GitHub
- Check the [discussions](https://github.com/hongkongkiwi/actionlint-mcp/discussions) for Q&A
- Review the [wiki](https://github.com/hongkongkiwi/actionlint-mcp/wiki) for detailed documentation

## ⭐ Star History

[![Star History Chart](https://api.star-history.com/svg?repos=hongkongkiwi/actionlint-mcp&type=Date)](https://star-history.com/#hongkongkiwi/actionlint-mcp&Date)