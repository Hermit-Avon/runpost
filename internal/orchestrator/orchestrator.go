package orchestrator

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Hermit-Avon/runpost/internal/config"
	"github.com/Hermit-Avon/runpost/internal/executor"
	"github.com/Hermit-Avon/runpost/internal/formatter"
	"github.com/Hermit-Avon/runpost/internal/model"
	"github.com/Hermit-Avon/runpost/internal/notifier"
	"github.com/Hermit-Avon/runpost/internal/policy"
)

type Options struct {
	Command         []string
	NotifyOn        string
	Timeout         time.Duration
	MaxCaptureBytes int
	DryRun          bool
}

func Run(ctx context.Context, cfg config.Config, opt Options) (int, error) {
	runner := executor.Runner{}

	if opt.MaxCaptureBytes > 0 {
		cfg.Capture.MaxStdoutBytes = opt.MaxCaptureBytes
		cfg.Capture.MaxStderrBytes = opt.MaxCaptureBytes
	}

	runCtx := ctx
	var cancel context.CancelFunc
	if opt.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, opt.Timeout)
		defer cancel()
	}

	result, err := runner.Run(runCtx, opt.Command, executor.Options{
		MaxStdoutBytes: cfg.Capture.MaxStdoutBytes,
		MaxStderrBytes: cfg.Capture.MaxStderrBytes,
	})
	if err != nil {
		return 1, err
	}

	notifyOn := cfg.NotifyOn
	if strings.TrimSpace(opt.NotifyOn) != "" {
		notifyOn = opt.NotifyOn
	}

	if policy.ShouldNotify(notifyOn, result) {
		msg, ferr := formatter.Format(result, cfg.Template)
		if ferr != nil {
			fmt.Fprintf(os.Stderr, "runpost: format notification failed: %v\n", ferr)
		} else {
			if opt.DryRun {
				fmt.Fprintf(os.Stderr, "runpost dry-run (%s)\n%s\n", msg.Title, msg.Body)
			} else {
				sendAll(ctx, cfg, msg)
			}
		}
	}

	return result.ExitCode, nil
}

func sendAll(ctx context.Context, cfg config.Config, msg model.Message) {
	ns := buildNotifiers(cfg)
	if len(ns) == 0 {
		return
	}

	var wg sync.WaitGroup
	for _, n := range ns {
		n := n
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := n.Send(ctx, msg); err != nil {
				fmt.Fprintf(os.Stderr, "runpost: notify[%s] failed: %v\n", n.Name(), err)
			}
		}()
	}
	wg.Wait()
}

func buildNotifiers(cfg config.Config) []notifier.Notifier {
	var ns []notifier.Notifier
	for _, ch := range cfg.Channels {
		switch ch.Type {
		case "webhook":
			n, err := notifier.NewWebhook(notifier.WebhookConfig{
				URL:       ch.URL,
				SecretEnv: ch.SecretEnv,
				Timeout:   config.ChannelTimeout(ch.Timeout),
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "runpost: skip invalid webhook notifier: %v\n", err)
				continue
			}
			ns = append(ns, n)
		case "telegram":
			n, err := notifier.NewTelegram(notifier.TelegramConfig{
				BotTokenEnv: ch.BotTokenEnv,
				ChatID:      ch.ChatID,
				Timeout:     config.ChannelTimeout(ch.Timeout),
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "runpost: skip invalid telegram notifier: %v\n", err)
				continue
			}
			ns = append(ns, n)
		default:
			if ch.Type != "" {
				fmt.Fprintf(os.Stderr, "runpost: unsupported channel type %q\n", ch.Type)
			}
		}
	}
	return ns
}
