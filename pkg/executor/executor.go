package executor

import (
	"bytes"
	"fmt"
	"os/exec"
)

// ExecutionResult contains the result of a script execution.
type ExecutionResult struct {
	ScriptName string
	Stdout     string
	Stderr     string
	Error      error
	Duration   float64
}

// Execute runs a shell script and returns the output.
func Execute(scriptName string, scriptContent string) *ExecutionResult {
	result := &ExecutionResult{
		ScriptName: scriptName,
	}

	cmd := exec.Command("bash", "-c", scriptContent)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()
	result.Error = err

	return result
}

// FormatResult returns a formatted string representation of the execution result.
func FormatResult(result *ExecutionResult) string {
	output := fmt.Sprintf("\n========== Script: %s ==========\n", result.ScriptName)
	output += fmt.Sprintf("Stdout:\n%s\n", result.Stdout)
	if result.Stderr != "" {
		output += fmt.Sprintf("Stderr:\n%s\n", result.Stderr)
	}
	if result.Error != nil {
		output += fmt.Sprintf("Error: %v\n", result.Error)
	}
	output += "====================================\n"
	return output
}
