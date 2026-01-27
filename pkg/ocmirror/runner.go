package ocmirror

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

// Result captures the output of an oc-mirror command execution.
type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
}

// Runner executes oc-mirror commands.
type Runner struct {
	binaryPath string
	env        []string
}

// NewRunner creates a runner pointing to the oc-mirror binary.
// If binaryPath is empty, it defaults to "oc-mirror" (expects it in PATH).
func NewRunner(binaryPath string) *Runner {
	if binaryPath == "" {
		binaryPath = "oc-mirror"
	}
	return &Runner{
		binaryPath: binaryPath,
	}
}

// WithEnv adds environment variables to the runner.
func (r *Runner) WithEnv(env []string) *Runner {
	r.env = env
	return r
}

// Run executes oc-mirror with the given arguments.
func (r *Runner) Run(ctx context.Context, args ...string) (*Result, error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx, r.binaryPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if len(r.env) > 0 {
		cmd.Env = append(cmd.Environ(), r.env...)
	}

	err := cmd.Run()

	result := &Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: time.Since(start),
	}

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		return result, fmt.Errorf("failed to execute oc-mirror: %w", err)
	}

	return result, nil
}

// MirrorToDisk runs oc-mirror to mirror images to a local directory.
func (r *Runner) MirrorToDisk(ctx context.Context, configPath, destDir string, extraArgs ...string) (*Result, error) {
	args := []string{
		"--config", configPath,
		fmt.Sprintf("file://%s", destDir),
		"--v2",
	}
	args = append(args, extraArgs...)
	return r.Run(ctx, args...)
}

// DiskToMirror runs oc-mirror to push images from a local archive to a registry.
func (r *Runner) DiskToMirror(ctx context.Context, configPath, sourceDir, destRegistry string, extraArgs ...string) (*Result, error) {
	args := []string{
		"--config", configPath,
		"--from", fmt.Sprintf("file://%s", sourceDir),
		fmt.Sprintf("docker://%s", destRegistry),
		"--v2",
	}
	args = append(args, extraArgs...)
	return r.Run(ctx, args...)
}

// MirrorToMirror runs oc-mirror to mirror images from one registry to another different registry.
func (r *Runner) MirrorToMirror(ctx context.Context, confPath, workspace, destRegistry string, extraArgs ...string) (*Result, error) {
	args := []string{
		"--config", confPath,
		"--workspace", fmt.Sprintf("file://%s", workspace),
		fmt.Sprintf("docker://%s", destRegistry),
		"--v2",
	}
	args = append(args, extraArgs...)
	return r.Run(ctx, args...)
}
