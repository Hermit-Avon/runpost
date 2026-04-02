package executor

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/Hermit-Avon/runpost/internal/capture"
	"github.com/Hermit-Avon/runpost/internal/model"
)

type Options struct {
	MaxStdoutBytes int
	MaxStderrBytes int
}

type Runner struct{}

func (r Runner) Run(ctx context.Context, command []string, opt Options) (model.CommandResult, error) {
	if len(command) == 0 {
		return model.CommandResult{Err: "empty command", ExitCode: 1}, errors.New("empty command")
	}

	start := time.Now()
	stdoutTail := capture.NewTailBuffer(opt.MaxStdoutBytes)
	stderrTail := capture.NewTailBuffer(opt.MaxStderrBytes)

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Stdout = io.MultiWriter(os.Stdout, stdoutTail)
	cmd.Stderr = io.MultiWriter(os.Stderr, stderrTail)

	err := cmd.Run()
	end := time.Now()

	result := model.CommandResult{
		Command:    append([]string{}, command...),
		StartAt:    start,
		EndAt:      end,
		Duration:   end.Sub(start),
		StdoutTail: stdoutTail.String(),
		StderrTail: stderrTail.String(),
		TimedOut:   errors.Is(ctx.Err(), context.DeadlineExceeded),
	}

	if err == nil {
		result.ExitCode = 0
		return result, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
	} else {
		result.ExitCode = 1
	}
	result.Err = err.Error()
	return result, nil
}
