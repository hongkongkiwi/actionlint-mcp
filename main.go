package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rhysd/actionlint"
)

// Build variables set by ldflags
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

type LintWorkflowParams struct {
	FilePath string `json:"file_path,omitempty" jsonschema:"description=Path to the workflow file to lint"`
	Content  string `json:"content,omitempty" jsonschema:"description=Content of the workflow file to lint (if file_path is not provided)"`
}

type CheckAllWorkflowsParams struct {
	Directory string `json:"directory,omitempty" jsonschema:"description=Directory to search for workflow files (defaults to .github/workflows)"`
}

type LintResult struct {
	Errors   []LintError `json:"errors"`
	Valid    bool        `json:"valid"`
	FilePath string      `json:"file_path,omitempty"`
}

type LintError struct {
	Message  string `json:"message"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Kind     string `json:"kind"`
	Severity string `json:"severity"`
}

func LintWorkflow(_ context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[LintWorkflowParams]) (*mcp.CallToolResultFor[any], error) {
	var filePath string
	var content []byte
	var err error

	switch {
	case params.Arguments.FilePath != "":
		filePath = params.Arguments.FilePath
		content, err = os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}
	case params.Arguments.Content != "":
		filePath = "inline.yml"
		content = []byte(params.Arguments.Content)
	default:
		return nil, fmt.Errorf("either file_path or content must be provided")
	}

	// Create linter with default options
	const configFilePath = ".github/actionlint.yaml"
	configFile := configFilePath
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		configFile = ""
	}

	opts := &actionlint.LinterOptions{
		Shellcheck:     os.Getenv("SHELLCHECK_COMMAND"),
		Pyflakes:       os.Getenv("PYFLAKES_COMMAND"),
		ConfigFile:     configFile,
		IgnorePatterns: []string{},
	}

	linter, err := actionlint.NewLinter(io.Discard, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create linter: %w", err)
	}

	// Run the linter
	errs, err := linter.Lint(filePath, content, nil)
	if err != nil {
		return nil, fmt.Errorf("linting failed: %w", err)
	}

	// Convert errors to our format
	result := LintResult{
		Errors:   make([]LintError, 0, len(errs)),
		Valid:    len(errs) == 0,
		FilePath: filePath,
	}

	for _, e := range errs {
		lintErr := LintError{
			Message: e.Message,
			Kind:    e.Kind,
		}

		// Get position info
		lintErr.Line = e.Line
		lintErr.Column = e.Column

		// Determine severity based on error kind
		switch e.Kind {
		case "syntax-check", "type-check":
			lintErr.Severity = "error"
		case "shellcheck", "pyflakes":
			lintErr.Severity = "warning"
		default:
			lintErr.Severity = "info"
		}

		result.Errors = append(result.Errors, lintErr)
	}

	// Convert result to JSON string for display
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(resultJSON),
			},
		},
	}, nil
}

func CheckAllWorkflows(_ context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[CheckAllWorkflowsParams]) (*mcp.CallToolResultFor[any], error) {
	directory := ".github/workflows"
	if params.Arguments.Directory != "" {
		directory = params.Arguments.Directory
	}

	// Find all workflow files
	pattern := filepath.Join(directory, "*.yml")
	files1, _ := filepath.Glob(pattern)
	pattern = filepath.Join(directory, "*.yaml")
	files2, _ := filepath.Glob(pattern)

	files := files1
	files = append(files, files2...)

	if len(files) == 0 {
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("No workflow files found in %s", directory),
				},
			},
		}, nil
	}

	// Lint all files
	allResults := make(map[string]LintResult)

	for _, file := range files {
		// Call LintWorkflow for each file
		lintParams := &mcp.CallToolParamsFor[LintWorkflowParams]{
			Arguments: LintWorkflowParams{
				FilePath: file,
			},
		}

		result, err := LintWorkflow(context.Background(), nil, lintParams)
		if err != nil {
			allResults[file] = LintResult{
				Errors: []LintError{{
					Message:  fmt.Sprintf("Failed to lint: %v", err),
					Severity: "error",
				}},
				Valid:    false,
				FilePath: file,
			}
			continue
		}

		// Parse the result back from JSON
		var lintResult LintResult
		if len(result.Content) > 0 {
			if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
				if err := json.Unmarshal([]byte(textContent.Text), &lintResult); err == nil {
					allResults[file] = lintResult
				}
			}
		}
	}

	// Format the results
	summary := map[string]interface{}{
		"total_files":       len(files),
		"files_with_errors": 0,
		"total_errors":      0,
		"results":           allResults,
	}

	for _, result := range allResults {
		if !result.Valid {
			summary["files_with_errors"] = summary["files_with_errors"].(int) + 1
			summary["total_errors"] = summary["total_errors"].(int) + len(result.Errors)
		}
	}

	resultJSON, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(resultJSON),
			},
		},
	}, nil
}

func main() {
	// Parse command line flags
	versionFlag := flag.Bool("version", false, "Print version information")
	flag.Parse()

	// Handle version flag
	if *versionFlag {
		fmt.Printf("actionlint-mcp %s\n", version)
		fmt.Printf("  Commit: %s\n", commit)
		fmt.Printf("  Built:  %s\n", date)
		fmt.Printf("  Built by: %s\n", builtBy)
		os.Exit(0)
	}

	// Create the server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "actionlint-mcp",
		Version: version,
	}, nil)

	// Register the lint_workflow tool
	lintSchema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"file_path": {
				Type:        "string",
				Description: "Path to the workflow file to lint",
			},
			"content": {
				Type:        "string",
				Description: "Content of the workflow file to lint (if file_path is not provided)",
			},
		},
		OneOf: []*jsonschema.Schema{
			{Required: []string{"file_path"}},
			{Required: []string{"content"}},
		},
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "lint_workflow",
		Description: "Lint a GitHub Actions workflow file using actionlint",
		InputSchema: lintSchema,
	}, LintWorkflow)

	// Register the check_all_workflows tool
	checkSchema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"directory": {
				Type:        "string",
				Description: "Directory to search for workflow files (defaults to .github/workflows)",
			},
		},
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "check_all_workflows",
		Description: "Check all GitHub Actions workflow files in a directory",
		InputSchema: checkSchema,
	}, CheckAllWorkflows)

	// Run the server
	if err := server.Run(context.Background(), mcp.NewStdioTransport()); err != nil {
		log.Fatal(err)
	}
}
