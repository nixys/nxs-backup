package exec_cmd

import (
	"bytes"
	"os"
	"os/exec"
)

// result contains command exec result
type result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Exec runs command string
func Exec(command string, args ...string) (result, error) {

	var stderr, stdout bytes.Buffer

	cmd := exec.Command(command, args...)

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set environment variables
	cmd.Env = os.Environ()

	err := cmd.Run()

	return result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: cmd.ProcessState.ExitCode(),
	}, err
}
