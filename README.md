# Actionlint MCP Server

[![Go Version](https://img.shields.io/github/go-mod/go-version/hongkongkiwi/actionlint-mcp)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/hongkongkiwi/actionlint-mcp)](https://goreportcard.com/report/github.com/hongkongkiwi/actionlint-mcp)

An MCP (Model Context Protocol) server that exposes [actionlint](https://github.com/rhysd/actionlint) functionality for linting GitHub Actions workflow files. This allows AI assistants like Claude to validate and check GitHub Actions workflows for errors, best practices, and security issues.

## Features

- **`lint_workflow`**: Lint a single GitHub Actions workflow file or content
- **`check_all_workflows`**: Check all workflow files in a directory
- Real-time validation of workflow syntax
- Detection of common mistakes and security issues
- Shell script validation (with shellcheck)
- Python code validation (with pyflakes)

## Installation

### Prerequisites

- Go 1.21+ installed (only for building from source)
- Optional: `shellcheck` for shell script validation
- Optional: `pyflakes` for Python code validation

### Quick Install (Recommended)

#### Using the install script (macOS/Linux)

```bash
curl -sSfL https://raw.githubusercontent.com/hongkongkiwi/actionlint-mcp/main/install.sh | sh -s -- -b /usr/local/bin
```

#### Download pre-built binaries

Download the latest release for your platform from the [releases page](https://github.com/hongkongkiwi/actionlint-mcp/releases).

Available for:
- **Linux**: amd64, arm64, arm/v7, arm/v6, 386, ppc64le, s390x, riscv64
- **macOS**: amd64 (Intel), arm64 (Apple Silicon)
- **Windows**: amd64, 386
- **FreeBSD**: amd64, arm64, arm/v7, 386
- **OpenBSD**: amd64, arm64, 386
- **NetBSD**: amd64, arm64, arm/v7, 386

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

### Docker

```bash
docker run -it ghcr.io/hongkongkiwi/actionlint-mcp:latest
```

## Binary Verification

All release binaries are signed using [Sigstore Cosign](https://github.com/sigstore/cosign) for supply chain security.

### Verify checksums and signatures

1. Download the binary, checksums, and signature files from the release:
```bash
VERSION=v1.0.0  # Replace with actual version
wget https://github.com/hongkongkiwi/actionlint-mcp/releases/download/${VERSION}/actionlint-mcp_Linux_x86_64.tar.gz
wget https://github.com/hongkongkiwi/actionlint-mcp/releases/download/${VERSION}/checksums.txt
wget https://github.com/hongkongkiwi/actionlint-mcp/releases/download/${VERSION}/checksums.txt.sig
wget https://github.com/hongkongkiwi/actionlint-mcp/releases/download/${VERSION}/checksums.txt.pem
```

2. Verify the checksum:
```bash
sha256sum -c checksums.txt 2>&1 | grep OK
```

3. Verify the signature with cosign:
```bash
cosign verify-blob \
  --certificate checksums.txt.pem \
  --signature checksums.txt.sig \
  checksums.txt
```

The signature is created using keyless signing with GitHub OIDC, ensuring the binary was built by the official GitHub Actions workflow.

## Editor Integration

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

## Usage

Once configured, the AI assistant can use these tools:

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

## Tools

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

## Environment Variables

- `SHELLCHECK_COMMAND`: Path to shellcheck binary for shell script validation
- `PYFLAKES_COMMAND`: Path to pyflakes binary for Python code validation

## Development

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

Install pre-commit hooks:

```bash
pip install pre-commit
pre-commit install
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [actionlint](https://github.com/rhysd/actionlint) - The powerful linter for GitHub Actions
- [MCP SDK for Go](https://github.com/modelcontextprotocol/go-sdk) - Model Context Protocol SDK

## Support

If you encounter any issues or have questions, please [open an issue](https://github.com/hongkongkiwi/actionlint-mcp/issues) on GitHub.