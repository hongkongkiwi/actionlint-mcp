package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test table for various workflow scenarios
var workflowTestCases = []struct {
	name        string
	workflow    string
	expectValid bool
	errorCount  int
	errorTypes  []string
}{
	{
		name: "valid_simple_workflow",
		workflow: `name: Test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4`,
		expectValid: true,
		errorCount:  0,
	},
	{
		name: "valid_complex_workflow",
		workflow: `name: CI/CD Pipeline
on:
  push:
    branches: [ main, develop ]
  pull_request:
    types: [ opened, synchronize, reopened ]
  schedule:
    - cron: '0 0 * * 0'

env:
  NODE_VERSION: '18'

jobs:
  test:
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest, macos-latest]
        node: [16, 18, 20]
    steps:
      - uses: actions/checkout@v4
      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: ${{ matrix.node }}
      - run: npm ci
      - run: npm test
  
  deploy:
    needs: test
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: echo "Deploying..."`,
		expectValid: true,
		errorCount:  0,
	},
	{
		name: "invalid_yaml_syntax",
		workflow: `name: Invalid
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          invalid syntax here
      - run: echo "test"`,
		expectValid: false,
		errorCount:  1,
	},
	{
		name: "invalid_missing_required_fields",
		workflow: `name: Missing Required
on: push
jobs:
  test:
    steps:
      - uses: actions/checkout@v4`,
		expectValid: false,
		errorCount:  1,
	},
	{
		name: "invalid_unknown_runner",
		workflow: `name: Unknown Runner
on: push
jobs:
  test:
    runs-on: invalid-runner
    steps:
      - run: echo "test"`,
		expectValid: false,
		errorCount:  1,
	},
	{
		name: "invalid_circular_dependency",
		workflow: `name: Circular Deps
on: push
jobs:
  job1:
    needs: job2
    runs-on: ubuntu-latest
    steps:
      - run: echo "job1"
  job2:
    needs: job1
    runs-on: ubuntu-latest
    steps:
      - run: echo "job2"`,
		expectValid: false,
		errorCount:  1,
	},
	{
		name: "valid_with_environment",
		workflow: `name: With Environment
on:
  push:
    branches: [ main ]
jobs:
  deploy:
    runs-on: ubuntu-latest
    environment:
      name: production
      url: https://example.com
    steps:
      - uses: actions/checkout@v4
      - run: echo "Deploying to production"`,
		expectValid: true,
		errorCount:  0,
	},
	{
		name: "valid_with_permissions",
		workflow: `name: With Permissions
on: push
permissions:
  contents: read
  issues: write
jobs:
  test:
    runs-on: ubuntu-latest
    permissions:
      checks: write
    steps:
      - uses: actions/checkout@v4`,
		expectValid: true,
		errorCount:  0,
	},
	{
		name: "invalid_syntax_in_expression",
		workflow: `name: Invalid Expression
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"
        if: ${{ invalid expression }}`,
		expectValid: false,
		errorCount:  1,
	},
	{
		name: "valid_with_outputs",
		workflow: `name: With Outputs
on: push
jobs:
  setup:
    runs-on: ubuntu-latest
    outputs:
      version: ${{ steps.get_version.outputs.version }}
    steps:
      - id: get_version
        run: echo "version=1.0.0" >> $GITHUB_OUTPUT
  build:
    needs: setup
    runs-on: ubuntu-latest
    steps:
      - run: echo "Building version ${{ needs.setup.outputs.version }}"`,
		expectValid: true,
		errorCount:  0,
	},
	{
		name: "valid_with_concurrency",
		workflow: `name: With Concurrency
on: push
concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4`,
		expectValid: true,
		errorCount:  0,
	},
	{
		name:        "empty_workflow",
		workflow:    ``,
		expectValid: false,
		errorCount:  1,
	},
	{
		name: "workflow_with_tabs",
		workflow: `name: With Tabs
on: push
jobs:
	test:
		runs-on: ubuntu-latest
		steps:
			- run: echo "test"`,
		expectValid: false,
		errorCount:  1,
	},
}

func TestLintWorkflow_TableDriven(t *testing.T) {
	session := &mcp.ServerSession{}

	for _, tc := range workflowTestCases {
		t.Run(tc.name, func(t *testing.T) {
			params := &mcp.CallToolParamsFor[LintWorkflowParams]{
				Arguments: LintWorkflowParams{
					Content: tc.workflow,
				},
			}

			result, err := LintWorkflow(context.Background(), session, params)

			if tc.workflow == "" {
				// Empty workflow should return an error
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			// Parse result
			assert.Len(t, result.Content, 1)
			textContent, ok := result.Content[0].(*mcp.TextContent)
			require.True(t, ok)

			var lintResult LintResult
			err = json.Unmarshal([]byte(textContent.Text), &lintResult)
			require.NoError(t, err)

			// Check validity
			assert.Equal(t, tc.expectValid, lintResult.Valid, "Workflow validity mismatch for %s", tc.name)

			// Check error count if specified
			if tc.errorCount >= 0 {
				assert.GreaterOrEqual(t, len(lintResult.Errors), tc.errorCount,
					"Expected at least %d errors for %s, got %d", tc.errorCount, tc.name, len(lintResult.Errors))
			}

			// If invalid, ensure there are errors
			if !tc.expectValid {
				assert.NotEmpty(t, lintResult.Errors, "Expected errors for invalid workflow %s", tc.name)
			}
		})
	}
}

func TestLintWorkflow_FileOperations(t *testing.T) {
	tempDir := t.TempDir()
	session := &mcp.ServerSession{}

	t.Run("read_file_with_BOM", func(t *testing.T) {
		// Create file with UTF-8 BOM
		workflow := "\xef\xbb\xbf" + `name: Test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4`

		filePath := filepath.Join(tempDir, "bom.yml")
		err := os.WriteFile(filePath, []byte(workflow), 0644)
		require.NoError(t, err)

		params := &mcp.CallToolParamsFor[LintWorkflowParams]{
			Arguments: LintWorkflowParams{
				FilePath: filePath,
			},
		}

		result, err := LintWorkflow(context.Background(), session, params)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("read_file_with_different_extensions", func(t *testing.T) {
		workflow := `name: Test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4`

		extensions := []string{".yml", ".yaml", ".YML", ".YAML"}
		for _, ext := range extensions {
			filePath := filepath.Join(tempDir, "workflow"+ext)
			err := os.WriteFile(filePath, []byte(workflow), 0644)
			require.NoError(t, err)

			params := &mcp.CallToolParamsFor[LintWorkflowParams]{
				Arguments: LintWorkflowParams{
					FilePath: filePath,
				},
			}

			result, err := LintWorkflow(context.Background(), session, params)
			assert.NoError(t, err, "Failed to lint file with extension %s", ext)
			assert.NotNil(t, result)
		}
	})

	t.Run("handle_permission_denied", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "readonly.yml")
		err := os.WriteFile(filePath, []byte("name: Test"), 0644)
		require.NoError(t, err)

		// Make file unreadable
		err = os.Chmod(filePath, 0000)
		require.NoError(t, err)
		defer os.Chmod(filePath, 0644) // Restore permissions for cleanup

		params := &mcp.CallToolParamsFor[LintWorkflowParams]{
			Arguments: LintWorkflowParams{
				FilePath: filePath,
			},
		}

		result, err := LintWorkflow(context.Background(), session, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to read file")
	})

	t.Run("handle_symlink", func(t *testing.T) {
		workflow := `name: Test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4`

		originalPath := filepath.Join(tempDir, "original.yml")
		symlinkPath := filepath.Join(tempDir, "symlink.yml")

		err := os.WriteFile(originalPath, []byte(workflow), 0644)
		require.NoError(t, err)

		err = os.Symlink(originalPath, symlinkPath)
		require.NoError(t, err)

		params := &mcp.CallToolParamsFor[LintWorkflowParams]{
			Arguments: LintWorkflowParams{
				FilePath: symlinkPath,
			},
		}

		result, err := LintWorkflow(context.Background(), session, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestLintWorkflow_LargeFiles(t *testing.T) {
	session := &mcp.ServerSession{}

	t.Run("large_valid_workflow", func(t *testing.T) {
		// Generate a large but valid workflow
		var builder strings.Builder
		builder.WriteString(`name: Large Workflow
on: push
jobs:`)

		for i := 0; i < 100; i++ {
			builder.WriteString(fmt.Sprintf(`
  job_%d:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: echo "Job %d"`, i, i))
		}

		params := &mcp.CallToolParamsFor[LintWorkflowParams]{
			Arguments: LintWorkflowParams{
				Content: builder.String(),
			},
		}

		result, err := LintWorkflow(context.Background(), session, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("deeply_nested_workflow", func(t *testing.T) {
		workflow := `name: Deeply Nested
on:
  push:
    branches:
      - main
    paths:
      - 'src/**'
      - 'tests/**'
jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        include:
          - os: ubuntu-latest
            node: 16
            env:
              TEST_ENV: development
              NESTED:
                DEEP:
                  VALUE: test
    steps:
      - uses: actions/checkout@v4
        with:
          submodules: recursive
          fetch-depth: 0
          token: ${{ secrets.GITHUB_TOKEN }}`

		params := &mcp.CallToolParamsFor[LintWorkflowParams]{
			Arguments: LintWorkflowParams{
				Content: workflow,
			},
		}

		result, err := LintWorkflow(context.Background(), session, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestLintWorkflow_ErrorDetails(t *testing.T) {
	session := &mcp.ServerSession{}

	testCases := []struct {
		name          string
		workflow      string
		expectedInMsg []string
		checkLine     bool
		minLine       int
	}{
		{
			name: "syntax_error_with_line_number",
			workflow: `name: Syntax Error
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          invalid: [syntax error]`,
			expectedInMsg: []string{"invalid"},
			checkLine:     true,
			minLine:       8,
		},
		{
			name: "undefined_needs",
			workflow: `name: Undefined Needs
on: push
jobs:
  job1:
    needs: nonexistent
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"`,
			expectedInMsg: []string{"nonexistent"},
			checkLine:     true,
			minLine:       4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := &mcp.CallToolParamsFor[LintWorkflowParams]{
				Arguments: LintWorkflowParams{
					Content: tc.workflow,
				},
			}

			result, err := LintWorkflow(context.Background(), session, params)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Parse result
			textContent, ok := result.Content[0].(*mcp.TextContent)
			require.True(t, ok)

			var lintResult LintResult
			err = json.Unmarshal([]byte(textContent.Text), &lintResult)
			require.NoError(t, err)

			assert.False(t, lintResult.Valid)
			assert.NotEmpty(t, lintResult.Errors)

			// Check error messages
			for _, expected := range tc.expectedInMsg {
				found := false
				for _, err := range lintResult.Errors {
					if strings.Contains(strings.ToLower(err.Message), strings.ToLower(expected)) {
						found = true

						// Check line number if required
						if tc.checkLine && tc.minLine > 0 {
							assert.GreaterOrEqual(t, err.Line, tc.minLine,
								"Error line number should be >= %d, got %d", tc.minLine, err.Line)
						}
						break
					}
				}
				assert.True(t, found, "Expected error message to contain '%s'", expected)
			}
		})
	}
}

func TestLintWorkflow_SpecialCharacters(t *testing.T) {
	session := &mcp.ServerSession{}

	testCases := []struct {
		name     string
		workflow string
	}{
		{
			name: "unicode_in_strings",
			workflow: `name: Unicode Test 测试 🚀
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "Hello 世界 🌍"`,
		},
		{
			name: "special_chars_in_env",
			workflow: `name: Special Chars
on: push
env:
  SPECIAL: "!@#$%^&*()"
  QUOTES: "He said 'hello'"
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "$SPECIAL"`,
		},
		{
			name: "multiline_strings",
			workflow: `name: Multiline
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: |
          echo "Line 1"
          echo "Line 2"
          echo "Line 3"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := &mcp.CallToolParamsFor[LintWorkflowParams]{
				Arguments: LintWorkflowParams{
					Content: tc.workflow,
				},
			}

			result, err := LintWorkflow(context.Background(), session, params)
			assert.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}
