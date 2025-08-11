package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ActionlintTestSuite struct {
	suite.Suite
	tempDir string
	session *mcp.ServerSession
}

func (suite *ActionlintTestSuite) SetupSuite() {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "actionlint-test-*")
	require.NoError(suite.T(), err)
	suite.tempDir = tempDir

	// Create a mock session
	suite.session = &mcp.ServerSession{}
}

func (suite *ActionlintTestSuite) TearDownSuite() {
	// Clean up temporary directory
	if suite.tempDir != "" {
		os.RemoveAll(suite.tempDir)
	}
}

func (suite *ActionlintTestSuite) TestLintWorkflow_ValidFile() {
	// Create a valid workflow file
	validWorkflow := `name: Test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: echo "Hello World"`

	filePath := filepath.Join(suite.tempDir, "valid.yml")
	err := os.WriteFile(filePath, []byte(validWorkflow), 0o644)
	require.NoError(suite.T(), err)

	// Test with file path
	params := &mcp.CallToolParamsFor[LintWorkflowParams]{
		Arguments: LintWorkflowParams{
			FilePath: filePath,
		},
	}

	result, err := LintWorkflow(context.Background(), suite.session, params)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), result)

	// Parse the result
	assert.Len(suite.T(), result.Content, 1)
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(suite.T(), ok)

	var lintResult LintResult
	err = json.Unmarshal([]byte(textContent.Text), &lintResult)
	require.NoError(suite.T(), err)

	assert.True(suite.T(), lintResult.Valid)
	assert.Empty(suite.T(), lintResult.Errors)
	assert.Equal(suite.T(), filePath, lintResult.FilePath)
}

func (suite *ActionlintTestSuite) TestLintWorkflow_InvalidFile() {
	// Create an invalid workflow file
	invalidWorkflow := `name: Test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: invalid/action@nonexistent
      - run: echo $UNDEFINED_VAR
      - name: Invalid syntax
        uses: actions/checkout@v4
        with:
          invalid-option: true`

	filePath := filepath.Join(suite.tempDir, "invalid.yml")
	err := os.WriteFile(filePath, []byte(invalidWorkflow), 0o644)
	require.NoError(suite.T(), err)

	params := &mcp.CallToolParamsFor[LintWorkflowParams]{
		Arguments: LintWorkflowParams{
			FilePath: filePath,
		},
	}

	result, err := LintWorkflow(context.Background(), suite.session, params)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), result)

	// Parse the result
	assert.Len(suite.T(), result.Content, 1)
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(suite.T(), ok)

	var lintResult LintResult
	err = json.Unmarshal([]byte(textContent.Text), &lintResult)
	require.NoError(suite.T(), err)

	assert.False(suite.T(), lintResult.Valid)
	assert.NotEmpty(suite.T(), lintResult.Errors)
}

func (suite *ActionlintTestSuite) TestLintWorkflow_ContentInput() {
	// Test with content input instead of file path
	workflow := `name: Test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4`

	params := &mcp.CallToolParamsFor[LintWorkflowParams]{
		Arguments: LintWorkflowParams{
			Content: workflow,
		},
	}

	result, err := LintWorkflow(context.Background(), suite.session, params)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), result)

	// Parse the result
	assert.Len(suite.T(), result.Content, 1)
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(suite.T(), ok)

	var lintResult LintResult
	err = json.Unmarshal([]byte(textContent.Text), &lintResult)
	require.NoError(suite.T(), err)

	assert.True(suite.T(), lintResult.Valid)
	assert.Equal(suite.T(), "inline.yml", lintResult.FilePath)
}

func (suite *ActionlintTestSuite) TestLintWorkflow_MissingInput() {
	// Test with no input
	params := &mcp.CallToolParamsFor[LintWorkflowParams]{
		Arguments: LintWorkflowParams{},
	}

	result, err := LintWorkflow(context.Background(), suite.session, params)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "either file_path or content must be provided")
}

func (suite *ActionlintTestSuite) TestLintWorkflow_NonExistentFile() {
	params := &mcp.CallToolParamsFor[LintWorkflowParams]{
		Arguments: LintWorkflowParams{
			FilePath: "/nonexistent/file.yml",
		},
	}

	result, err := LintWorkflow(context.Background(), suite.session, params)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to read file")
}

func (suite *ActionlintTestSuite) TestCheckAllWorkflows_ValidWorkflows() {
	// Create a workflows directory
	workflowsDir := filepath.Join(suite.tempDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0o755)
	require.NoError(suite.T(), err)

	// Create multiple workflow files
	workflows := map[string]string{
		"ci.yml": `name: CI
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4`,
		"deploy.yaml": `name: Deploy
on:
  push:
    branches: [main]
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4`,
	}

	for name, content := range workflows {
		filePath := filepath.Join(workflowsDir, name)
		writeErr := os.WriteFile(filePath, []byte(content), 0o644)
		require.NoError(suite.T(), writeErr)
	}

	params := &mcp.CallToolParamsFor[CheckAllWorkflowsParams]{
		Arguments: CheckAllWorkflowsParams{
			Directory: workflowsDir,
		},
	}

	result, err := CheckAllWorkflows(context.Background(), suite.session, params)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), result)

	// Parse the result
	assert.Len(suite.T(), result.Content, 1)
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(suite.T(), ok)

	var summary map[string]interface{}
	err = json.Unmarshal([]byte(textContent.Text), &summary)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), float64(2), summary["total_files"])
	assert.Equal(suite.T(), float64(0), summary["files_with_errors"])
	assert.Equal(suite.T(), float64(0), summary["total_errors"])
}

func (suite *ActionlintTestSuite) TestCheckAllWorkflows_EmptyDirectory() {
	emptyDir := filepath.Join(suite.tempDir, "empty")
	err := os.MkdirAll(emptyDir, 0o755)
	require.NoError(suite.T(), err)

	params := &mcp.CallToolParamsFor[CheckAllWorkflowsParams]{
		Arguments: CheckAllWorkflowsParams{
			Directory: emptyDir,
		},
	}

	result, err := CheckAllWorkflows(context.Background(), suite.session, params)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), result)

	assert.Len(suite.T(), result.Content, 1)
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(suite.T(), ok)
	assert.Contains(suite.T(), textContent.Text, "No workflow files found")
}

func (suite *ActionlintTestSuite) TestCheckAllWorkflows_WithErrors() {
	// Create a workflows directory
	workflowsDir := filepath.Join(suite.tempDir, "workflows-with-errors")
	err := os.MkdirAll(workflowsDir, 0o755)
	require.NoError(suite.T(), err)

	// Create a workflow with syntax errors
	invalidWorkflow := `name: Invalid
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          invalid syntax here
      - run: |
          echo "Missing quote`

	filePath := filepath.Join(workflowsDir, "invalid.yml")
	err = os.WriteFile(filePath, []byte(invalidWorkflow), 0o644)
	require.NoError(suite.T(), err)

	params := &mcp.CallToolParamsFor[CheckAllWorkflowsParams]{
		Arguments: CheckAllWorkflowsParams{
			Directory: workflowsDir,
		},
	}

	result, err := CheckAllWorkflows(context.Background(), suite.session, params)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), result)

	// Parse the result
	assert.Len(suite.T(), result.Content, 1)
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(suite.T(), ok)

	var summary map[string]interface{}
	err = json.Unmarshal([]byte(textContent.Text), &summary)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), float64(1), summary["total_files"])
	// Files with errors should be > 0
	filesWithErrors, ok := summary["files_with_errors"].(float64)
	require.True(suite.T(), ok)
	assert.Greater(suite.T(), filesWithErrors, float64(0))
}

func TestActionlintTestSuite(t *testing.T) {
	suite.Run(t, new(ActionlintTestSuite))
}

// Benchmark tests
func BenchmarkLintWorkflow(b *testing.B) {
	workflow := `name: Benchmark Test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: echo "Hello World"`

	session := &mcp.ServerSession{}
	params := &mcp.CallToolParamsFor[LintWorkflowParams]{
		Arguments: LintWorkflowParams{
			Content: workflow,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = LintWorkflow(context.Background(), session, params)
	}
}

func BenchmarkCheckAllWorkflows(b *testing.B) {
	// Create temp directory with workflows
	tempDir, err := os.MkdirTemp("", "bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	err = os.MkdirAll(workflowsDir, 0o755)
	if err != nil {
		b.Fatal(err)
	}

	// Create test workflow files
	for i := 0; i < 5; i++ {
		workflow := `name: Test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4`
		filePath := filepath.Join(workflowsDir, "test%d.yml")
		err := os.WriteFile(filePath, []byte(workflow), 0o644)
		if err != nil {
			b.Fatal(err)
		}
	}

	session := &mcp.ServerSession{}
	params := &mcp.CallToolParamsFor[CheckAllWorkflowsParams]{
		Arguments: CheckAllWorkflowsParams{
			Directory: workflowsDir,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = CheckAllWorkflows(context.Background(), session, params)
	}
}
