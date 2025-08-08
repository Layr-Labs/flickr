package docker

import (
	"context"
	"fmt"
	"os/exec"
)

type RunOptions struct {
	Name     string
	Detached bool
	Env      map[string]string
	Cmd      []string // Optional command to run in container
}

type Docker interface {
	Pull(ctx context.Context, ref string) error
	Run(ctx context.Context, ref string, opts RunOptions) error
}

type Runner struct{}

func New() *Runner {
	return &Runner{}
}

func (r *Runner) Pull(ctx context.Context, ref string) error {
	cmd := exec.CommandContext(ctx, "docker", "pull", ref)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker pull failed: %v\n%s", err, string(out))
	}
	return nil
}

func (r *Runner) Run(ctx context.Context, ref string, opts RunOptions) error {
	args := []string{"run"}
	if opts.Name != "" {
		args = append(args, "--name", opts.Name)
	}
	if opts.Detached {
		args = append(args, "-d")
	}
	for k, v := range opts.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}
	args = append(args, ref)
	
	// Add optional command
	if len(opts.Cmd) > 0 {
		args = append(args, opts.Cmd...)
	}
	
	cmd := exec.CommandContext(ctx, "docker", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker run failed: %v\n%s", err, string(out))
	}
	return nil
}