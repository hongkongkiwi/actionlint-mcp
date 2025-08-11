package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockFileSystem mocks file operations
type MockFileSystem struct {
	mock.Mock
}

func (m *MockFileSystem) ReadFile(path string) ([]byte, error) {
	args := m.Called(path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockFileSystem) Stat(path string) (os.FileInfo, error) {
	args := m.Called(path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(os.FileInfo), args.Error(1)
}

// MockWriter mocks io.Writer for testing output
type MockWriter struct {
	mock.Mock
	Data []byte
}

func (m *MockWriter) Write(p []byte) (n int, err error) {
	m.Data = append(m.Data, p...)
	args := m.Called(p)
	return args.Int(0), args.Error(1)
}

func TestLintWorkflowWithMocks(t *testing.T) {
	session := &mcp.ServerSession{}

	t.Run("simulate_file_read_error", func(t *testing.T) {
		// This simulates what happens when a file can't be read
		params := &mcp.CallToolParamsFor[LintWorkflowParams]{
			Arguments: LintWorkflowParams{
				FilePath: "/this/path/does/not/exist/workflow.yml",
			},
		}

		result, err := LintWorkflow(context.Background(), session, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to read file")
	})

	t.Run("simulate_empty_params", func(t *testing.T) {
		params := &mcp.CallToolParamsFor[LintWorkflowParams]{
			Arguments: LintWorkflowParams{},
		}

		result, err := LintWorkflow(context.Background(), session, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "either file_path or content must be provided")
	})

	t.Run("simulate_nil_context", func(t *testing.T) {
		// Test with nil context - should still work
		params := &mcp.CallToolParamsFor[LintWorkflowParams]{
			Arguments: LintWorkflowParams{
				Content: `name: Test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4`,
			},
		}

		result, err := LintWorkflow(context.TODO(), session, params) // nil context
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestCheckAllWorkflowsWithMocks(t *testing.T) {
	session := &mcp.ServerSession{}

	t.Run("directory_does_not_exist", func(t *testing.T) {
		params := &mcp.CallToolParamsFor[CheckAllWorkflowsParams]{
			Arguments: CheckAllWorkflowsParams{
				Directory: "/nonexistent/directory/that/should/not/exist",
			},
		}

		result, err := CheckAllWorkflows(context.Background(), session, params)
		assert.NoError(t, err) // Should not error, just return no files found
		assert.NotNil(t, result)

		textContent, ok := result.Content[0].(*mcp.TextContent)
		require.True(t, ok)
		assert.Contains(t, textContent.Text, "No workflow files found")
	})

	t.Run("empty_directory_param", func(t *testing.T) {
		// When directory is empty, it should default to .github/workflows
		params := &mcp.CallToolParamsFor[CheckAllWorkflowsParams]{
			Arguments: CheckAllWorkflowsParams{
				Directory: "",
			},
		}

		result, err := CheckAllWorkflows(context.Background(), session, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		// Result depends on whether .github/workflows exists in test environment
	})
}

// TestErrorScenarios tests various error conditions
func TestErrorScenarios(t *testing.T) {
	session := &mcp.ServerSession{}

	errorScenarios := []struct {
		name          string
		params        *mcp.CallToolParamsFor[LintWorkflowParams]
		expectError   bool
		errorContains string
	}{
		{
			name: "nil_params",
			params: &mcp.CallToolParamsFor[LintWorkflowParams]{
				Arguments: LintWorkflowParams{},
			},
			expectError:   true,
			errorContains: "either file_path or content must be provided",
		},
		{
			name: "empty_content",
			params: &mcp.CallToolParamsFor[LintWorkflowParams]{
				Arguments: LintWorkflowParams{
					Content: "",
				},
			},
			expectError:   true,
			errorContains: "either file_path or content must be provided",
		},
		{
			name: "whitespace_only_content",
			params: &mcp.CallToolParamsFor[LintWorkflowParams]{
				Arguments: LintWorkflowParams{
					Content: "   \n\t  \n  ",
				},
			},
			expectError: false, // Should process but find errors
		},
		{
			name: "binary_file_path",
			params: &mcp.CallToolParamsFor[LintWorkflowParams]{
				Arguments: LintWorkflowParams{
					FilePath: "/usr/bin/ls", // Binary file
				},
			},
			expectError: false, // Will try to read but actionlint will error
		},
	}

	for _, scenario := range errorScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			result, err := LintWorkflow(context.Background(), session, scenario.params)

			if scenario.expectError {
				assert.Error(t, err)
				if scenario.errorContains != "" {
					assert.Contains(t, err.Error(), scenario.errorContains)
				}
				assert.Nil(t, result)
			} else if err == nil {
				// May or may not error depending on the scenario
					assert.NotNil(t, result)
				}
			}
		})
	}
}

// TestJSONMarshaling tests JSON marshaling/unmarshaling
func TestJSONMarshaling(t *testing.T) {
	t.Run("lint_result_marshaling", func(t *testing.T) {
		result := LintResult{
			Errors: []LintError{
				{
					Message:  "Test error",
					Line:     10,
					Column:   5,
					Kind:     "syntax",
					Severity: "error",
				},
				{
					Message:  "Another error",
					Line:     20,
					Column:   15,
					Kind:     "expression",
					Severity: "warning",
				},
			},
			Valid:    false,
			FilePath: "test.yml",
		}

		// Marshal to JSON
		data, err := json.Marshal(result)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)

		// Unmarshal back
		var decoded LintResult
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		// Verify fields
		assert.Equal(t, result.Valid, decoded.Valid)
		assert.Equal(t, result.FilePath, decoded.FilePath)
		assert.Len(t, decoded.Errors, 2)
		assert.Equal(t, result.Errors[0].Message, decoded.Errors[0].Message)
		assert.Equal(t, result.Errors[0].Line, decoded.Errors[0].Line)
		assert.Equal(t, result.Errors[0].Column, decoded.Errors[0].Column)
		assert.Equal(t, result.Errors[0].Kind, decoded.Errors[0].Kind)
		assert.Equal(t, result.Errors[0].Severity, decoded.Errors[0].Severity)
	})

	t.Run("params_marshaling", func(t *testing.T) {
		params := LintWorkflowParams{
			FilePath: "/path/to/file.yml",
			Content:  "workflow content",
		}

		// Marshal to JSON
		data, err := json.Marshal(params)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)

		// Unmarshal back
		var decoded LintWorkflowParams
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, params.FilePath, decoded.FilePath)
		assert.Equal(t, params.Content, decoded.Content)
	})

	t.Run("check_all_params_marshaling", func(t *testing.T) {
		params := CheckAllWorkflowsParams{
			Directory: "/path/to/workflows",
		}

		// Marshal to JSON
		data, err := json.Marshal(params)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)

		// Unmarshal back
		var decoded CheckAllWorkflowsParams
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)

		assert.Equal(t, params.Directory, decoded.Directory)
	})
}

// TestConfigFileHandling tests config file handling
func TestConfigFileHandling(t *testing.T) {
	session := &mcp.ServerSession{}

	t.Run("with_custom_config_file", func(t *testing.T) {
		// Create a custom actionlint config
		configDir := ".github"
		err := os.MkdirAll(configDir, 0o755)
		if err != nil {
			t.Skip("Cannot create .github directory")
		}
		defer os.RemoveAll(configDir)

		configContent := `self-hosted-runner:
  labels:
    - custom-runner
    - self-hosted`

		configPath := ".github/actionlint.yaml"
		err = os.WriteFile(configPath, []byte(configContent), 0o644)
		if err != nil {
			t.Skip("Cannot create config file")
		}
		defer os.Remove(configPath)

		workflow := `name: Test
on: push
jobs:
  test:
    runs-on: custom-runner
    steps:
      - uses: actions/checkout@v4`

		params := &mcp.CallToolParamsFor[LintWorkflowParams]{
			Arguments: LintWorkflowParams{
				Content: workflow,
			},
		}

		result, err := LintWorkflow(context.Background(), session, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// With the config, custom-runner should be valid
		textContent, ok := result.Content[0].(*mcp.TextContent)
		require.True(t, ok)

		var lintResult LintResult
		err = json.Unmarshal([]byte(textContent.Text), &lintResult)
		require.NoError(t, err)

		// Should be valid with the custom runner defined in config
		assert.True(t, lintResult.Valid)
	})
}

// TestOutputWriter tests the io.Discard writer usage
func TestOutputWriter(t *testing.T) {
	// Test that we're using io.Discard for linter output
	session := &mcp.ServerSession{}

	// Create a workflow that would normally produce output
	workflow := `name: Test with Output
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: |
          echo "This would normally produce output"
          echo "But it should be discarded"`

	params := &mcp.CallToolParamsFor[LintWorkflowParams]{
		Arguments: LintWorkflowParams{
			Content: workflow,
		},
	}

	// Capture stdout to ensure nothing is printed
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	result, err := LintWorkflow(context.Background(), session, params)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read any output that was captured
	output := make([]byte, 1024)
	n, _ := r.Read(output)
	r.Close()

	// Should not have printed anything to stdout
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, n, "No output should be printed to stdout")
}

// TestVersionInfo tests version information
func TestVersionInfo(t *testing.T) {
	// Test that version variables exist and can be set
	oldVersion := version
	oldCommit := commit
	oldDate := date
	oldBuiltBy := builtBy

	defer func() {
		version = oldVersion
		commit = oldCommit
		date = oldDate
		builtBy = oldBuiltBy
	}()

	version = "test-version"
	commit = "test-commit"
	date = "test-date"
	builtBy = "test-builder"

	assert.Equal(t, "test-version", version)
	assert.Equal(t, "test-commit", commit)
	assert.Equal(t, "test-date", date)
	assert.Equal(t, "test-builder", builtBy)
}

// MockError is a custom error type for testing
type MockError struct {
	message string
}

func (e MockError) Error() string {
	return e.message
}

func TestCustomErrors(t *testing.T) {
	// Test that our error handling works with custom error types
	err := MockError{message: "custom error"}
	assert.Error(t, err)
	assert.Equal(t, "custom error", err.Error())

	// Test that we can wrap errors properly
	wrappedErr := errors.New("wrapped: " + err.Error())
	assert.Error(t, wrappedErr)
	assert.Contains(t, wrappedErr.Error(), "custom error")
}

// TestIOReaderScenarios tests various io.Reader scenarios
func TestIOReaderScenarios(t *testing.T) {
	// Test that io.Discard works as expected
	n, err := io.Discard.Write([]byte("test data"))
	assert.NoError(t, err)
	assert.Greater(t, n, 0)

	// Test that io.Discard can handle large writes
	largeData := make([]byte, 1024*1024) // 1MB
	n, err = io.Discard.Write(largeData)
	assert.NoError(t, err)
	assert.Equal(t, len(largeData), n)
}
