package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Hermit-Avon/runpost/internal/config"
	"github.com/Hermit-Avon/runpost/internal/orchestrator"
)

type CLIOptions struct {
	ConfigPath      string
	NotifyOn        string
	Timeout         time.Duration
	MaxCaptureBytes int
	DryRun          bool
}

func Main(args []string, stderr io.Writer) int {
	opt, cmd, err := parseArgs(args, stderr)
	if err != nil {
		if err.Error() != "usage" {
			fmt.Fprintf(stderr, "runpost: %v\n", err)
		}
		return 1
	}

	cfg, err := config.Load(opt.ConfigPath)
	if err != nil {
		fmt.Fprintf(stderr, "runpost: load config failed: %v\n", err)
		return 1
	}

	if len(cmd) == 0 {
		printUsage(stderr)
		return 1
	}

	exitCode, err := orchestrator.Run(context.Background(), cfg, orchestrator.Options{
		Command:         cmd,
		NotifyOn:        opt.NotifyOn,
		Timeout:         opt.Timeout,
		MaxCaptureBytes: opt.MaxCaptureBytes,
		DryRun:          opt.DryRun,
	})
	if err != nil {
		fmt.Fprintf(stderr, "runpost: %v\n", err)
		return 1
	}
	return exitCode
}

func parseArgs(args []string, stderr io.Writer) (CLIOptions, []string, error) {
	fs := flag.NewFlagSet("runpost", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var opt CLIOptions
	fs.StringVar(&opt.ConfigPath, "config", "", "config file path")
	fs.StringVar(&opt.NotifyOn, "notify-on", "", "notify policy: always|failure|timeout|never")
	fs.DurationVar(&opt.Timeout, "timeout", 0, "command timeout, e.g. 30s")
	fs.IntVar(&opt.MaxCaptureBytes, "max-capture-bytes", 0, "capture tail bytes for stdout/stderr")
	fs.BoolVar(&opt.DryRun, "dry-run", false, "print notification message but do not send")

	idx := indexOfDoubleDash(args)
	if idx >= 0 {
		if err := fs.Parse(args[:idx]); err != nil {
			return opt, nil, err
		}
		return opt, args[idx+1:], nil
	}

	if err := fs.Parse(args); err != nil {
		return opt, nil, err
	}

	rest := fs.Args()
	if len(rest) == 0 {
		printUsage(stderr)
		return opt, nil, fmt.Errorf("usage")
	}
	return opt, rest, nil
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: runpost [flags] -- <command> [args...]")
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --config <path>")
	fmt.Fprintln(w, "  --notify-on <always|failure|timeout|never>")
	fmt.Fprintln(w, "  --timeout <duration>")
	fmt.Fprintln(w, "  --max-capture-bytes <n>")
	fmt.Fprintln(w, "  --dry-run")
}

func indexOfDoubleDash(args []string) int {
	for i, v := range args {
		if strings.TrimSpace(v) == "--" {
			return i
		}
	}
	return -1
}
