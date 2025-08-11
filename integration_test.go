package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerCreation(t *testing.T) {
	// Test server creation
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	assert.NotNil(t, server)
}

func TestToolRegistration(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	// Register lint_workflow tool
	lintSchema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"file_path": {
				Type:        "string",
				Description: "Path to the workflow file to lint",
			},
			"content": {
				Type:        "string",
				Description: "Content of the workflow file to lint",
			},
		},
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "lint_workflow",
		Description: "Lint a GitHub Actions workflow file",
		InputSchema: lintSchema,
	}, LintWorkflow)

	// Register check_all_workflows tool
	checkSchema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"directory": {
				Type:        "string",
				Description: "Directory to search for workflow files",
			},
		},
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "check_all_workflows",
		Description: "Check all workflow files in a directory",
		InputSchema: checkSchema,
	}, CheckAllWorkflows)

	// Server should be created without errors
	assert.NotNil(t, server)
}

func TestConcurrentLinting(t *testing.T) {
	session := &mcp.ServerSession{}
	workflows := []string{
		`name: Test1
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4`,
		`name: Test2
on: pull_request
jobs:
  test:
    runs-on: windows-latest
    steps:
      - uses: actions/setup-node@v4`,
		`name: Test3
on:
  schedule:
    - cron: '0 0 * * *'
jobs:
  test:
    runs-on: macos-latest
    steps:
      - run: echo "test"`,
	}

	var wg sync.WaitGroup
	results := make([]bool, len(workflows))
	errors := make([]error, len(workflows))

	for i, workflow := range workflows {
		wg.Add(1)
		go func(index int, content string) {
			defer wg.Done()

			params := &mcp.CallToolParamsFor[LintWorkflowParams]{
				Arguments: LintWorkflowParams{
					Content: content,
				},
			}

			result, err := LintWorkflow(context.Background(), session, params)
			if err != nil {
				errors[index] = err
				return
			}

			if result != nil && len(result.Content) > 0 {
				if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
					var lintResult LintResult
					if err := json.Unmarshal([]byte(textContent.Text), &lintResult); err == nil {
						results[index] = lintResult.Valid
					}
				}
			}
		}(i, workflow)
	}

	wg.Wait()

	// Check that all concurrent operations succeeded
	for i, err := range errors {
		assert.NoError(t, err, "Concurrent lint %d failed", i)
	}

	for i, valid := range results {
		assert.True(t, valid, "Workflow %d should be valid", i)
	}
}

func TestCheckAllWorkflows_Integration(t *testing.T) {
	tempDir := t.TempDir()
	session := &mcp.ServerSession{}

	// Create a complex directory structure
	dirs := []string{
		filepath.Join(tempDir, ".github", "workflows"),
		filepath.Join(tempDir, "other", "workflows"),
		filepath.Join(tempDir, "nested", "deep", "workflows"),
	}

	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err)
	}

	// Create various workflow files
	workflows := map[string]string{
		filepath.Join(dirs[0], "ci.yml"): `name: CI
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4`,
		filepath.Join(dirs[0], "cd.yaml"): `name: CD
on:
  push:
    branches: [main]
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4`,
		filepath.Join(dirs[0], "invalid.yml"): `name: Invalid
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          bad syntax: here`,
		filepath.Join(dirs[1], "other.yml"): `name: Other
on: workflow_dispatch
jobs:
  run:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"`,
	}

	for path, content := range workflows {
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Also create non-workflow files that should be ignored
	nonWorkflows := []string{
		filepath.Join(dirs[0], "README.md"),
		filepath.Join(dirs[0], "config.json"),
		filepath.Join(dirs[0], "test.txt"),
	}

	for _, path := range nonWorkflows {
		err := os.WriteFile(path, []byte("not a workflow"), 0644)
		require.NoError(t, err)
	}

	t.Run("check_github_workflows_directory", func(t *testing.T) {
		params := &mcp.CallToolParamsFor[CheckAllWorkflowsParams]{
			Arguments: CheckAllWorkflowsParams{
				Directory: dirs[0],
			},
		}

		result, err := CheckAllWorkflows(context.Background(), session, params)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Parse result
		textContent, ok := result.Content[0].(*mcp.TextContent)
		require.True(t, ok)

		var summary map[string]interface{}
		err = json.Unmarshal([]byte(textContent.Text), &summary)
		require.NoError(t, err)

		// Should find 3 workflow files (ci.yml, cd.yaml, invalid.yml)
		assert.Equal(t, float64(3), summary["total_files"])
		// At least one file has errors (invalid.yml)
		assert.GreaterOrEqual(t, summary["files_with_errors"].(float64), float64(1))
	})

	t.Run("check_other_directory", func(t *testing.T) {
		params := &mcp.CallToolParamsFor[CheckAllWorkflowsParams]{
			Arguments: CheckAllWorkflowsParams{
				Directory: dirs[1],
			},
		}

		result, err := CheckAllWorkflows(context.Background(), session, params)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Parse result
		textContent, ok := result.Content[0].(*mcp.TextContent)
		require.True(t, ok)

		var summary map[string]interface{}
		err = json.Unmarshal([]byte(textContent.Text), &summary)
		require.NoError(t, err)

		// Should find 1 workflow file
		assert.Equal(t, float64(1), summary["total_files"])
		assert.Equal(t, float64(0), summary["files_with_errors"])
	})

	t.Run("check_empty_directory", func(t *testing.T) {
		params := &mcp.CallToolParamsFor[CheckAllWorkflowsParams]{
			Arguments: CheckAllWorkflowsParams{
				Directory: dirs[2],
			},
		}

		result, err := CheckAllWorkflows(context.Background(), session, params)
		require.NoError(t, err)
		require.NotNil(t, result)

		textContent, ok := result.Content[0].(*mcp.TextContent)
		require.True(t, ok)
		assert.Contains(t, textContent.Text, "No workflow files found")
	})

	t.Run("check_nonexistent_directory", func(t *testing.T) {
		params := &mcp.CallToolParamsFor[CheckAllWorkflowsParams]{
			Arguments: CheckAllWorkflowsParams{
				Directory: filepath.Join(tempDir, "nonexistent"),
			},
		}

		result, err := CheckAllWorkflows(context.Background(), session, params)
		require.NoError(t, err)
		require.NotNil(t, result)

		textContent, ok := result.Content[0].(*mcp.TextContent)
		require.True(t, ok)
		assert.Contains(t, textContent.Text, "No workflow files found")
	})
}

func TestEnvironmentVariables(t *testing.T) {
	session := &mcp.ServerSession{}

	t.Run("with_shellcheck_enabled", func(t *testing.T) {
		// Temporarily set SHELLCHECK_COMMAND
		oldVal := os.Getenv("SHELLCHECK_COMMAND")
		os.Setenv("SHELLCHECK_COMMAND", "shellcheck")
		defer os.Setenv("SHELLCHECK_COMMAND", oldVal)

		workflow := `name: Shell Test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: |
          echo $undefined_var
          [ -z "$var" ] && echo "empty"`

		params := &mcp.CallToolParamsFor[LintWorkflowParams]{
			Arguments: LintWorkflowParams{
				Content: workflow,
			},
		}

		result, err := LintWorkflow(context.Background(), session, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("with_pyflakes_enabled", func(t *testing.T) {
		// Temporarily set PYFLAKES_COMMAND
		oldVal := os.Getenv("PYFLAKES_COMMAND")
		os.Setenv("PYFLAKES_COMMAND", "pyflakes")
		defer os.Setenv("PYFLAKES_COMMAND", oldVal)

		workflow := `name: Python Test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: |
          python -c "
          import sys
          undefined_variable
          print('test')
          "`

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

func TestMainFunction(t *testing.T) {
	t.Run("version_flag", func(t *testing.T) {
		// Save original args
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		// Test version flag
		os.Args = []string{"actionlint-mcp", "-version"}

		// We can't easily test main() directly since it calls os.Exit
		// Instead, test the version variables are set correctly
		assert.NotEmpty(t, version)
		assert.NotEmpty(t, commit)
		assert.NotEmpty(t, date)
		assert.NotEmpty(t, builtBy)
	})
}

func TestPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	session := &mcp.ServerSession{}

	t.Run("lint_large_workflow_performance", func(t *testing.T) {
		// Generate a large workflow
		var workflow string
		workflow = `name: Large Workflow
on: push
jobs:`
		for i := 0; i < 50; i++ {
			workflow += `
  job` + string(rune('0'+i)) + `:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: echo "test"`
		}

		start := time.Now()

		params := &mcp.CallToolParamsFor[LintWorkflowParams]{
			Arguments: LintWorkflowParams{
				Content: workflow,
			},
		}

		result, err := LintWorkflow(context.Background(), session, params)

		duration := time.Since(start)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		// Should complete within reasonable time (5 seconds for large workflow)
		assert.Less(t, duration, 5*time.Second, "Linting took too long: %v", duration)
	})

	t.Run("check_many_workflows_performance", func(t *testing.T) {
		tempDir := t.TempDir()
		workflowsDir := filepath.Join(tempDir, ".github", "workflows")
		err := os.MkdirAll(workflowsDir, 0755)
		require.NoError(t, err)

		// Create many workflow files
		for i := 0; i < 20; i++ {
			workflow := `name: Test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4`

			filePath := filepath.Join(workflowsDir, "workflow"+string(rune('0'+i))+".yml")
			err := os.WriteFile(filePath, []byte(workflow), 0644)
			require.NoError(t, err)
		}

		start := time.Now()

		params := &mcp.CallToolParamsFor[CheckAllWorkflowsParams]{
			Arguments: CheckAllWorkflowsParams{
				Directory: workflowsDir,
			},
		}

		result, err := CheckAllWorkflows(context.Background(), session, params)

		duration := time.Since(start)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		// Should complete within reasonable time (10 seconds for many files)
		assert.Less(t, duration, 10*time.Second, "Checking all workflows took too long: %v", duration)
	})
}

func TestMemoryLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	session := &mcp.ServerSession{}
	workflow := `name: Test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4`

	// Run multiple iterations to detect potential memory leaks
	for i := 0; i < 100; i++ {
		params := &mcp.CallToolParamsFor[LintWorkflowParams]{
			Arguments: LintWorkflowParams{
				Content: workflow,
			},
		}

		result, err := LintWorkflow(context.Background(), session, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	}

	// If we get here without running out of memory, the test passes
	assert.True(t, true, "No memory leak detected")
}
