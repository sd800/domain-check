package registry

import (
	"context"
	"os/exec"
)

// Use the macOS system executable, never a program supplied through PATH.
// DNS keeps dig's normal retry/time limits under the lookup's overall context.
func runSystemDig(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "/usr/bin/dig", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	data, readErr := readLimited(stdout)
	if readErr != nil {
		_ = cmd.Process.Kill()
	}
	waitErr := cmd.Wait()
	if readErr != nil {
		return nil, readErr
	}
	return data, waitErr
}
