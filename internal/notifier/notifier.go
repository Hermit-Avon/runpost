package notifier

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Hermit-Avon/runpost/internal/model"
)

type Notifier interface {
	Name() string
	Send(ctx context.Context, msg model.Message) error
}

type WebhookConfig struct {
	URL       string
	SecretEnv string
	Timeout   time.Duration
}

type TelegramConfig struct {
	BotTokenEnv string
	ChatID      string
	Timeout     time.Duration
}

type webhookNotifier struct {
	cfg    WebhookConfig
	client *http.Client
}

type telegramNotifier struct {
	cfg    TelegramConfig
	client *http.Client
}

func NewWebhook(cfg WebhookConfig) (Notifier, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, errors.New("webhook url is empty")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	return &webhookNotifier{cfg: cfg, client: &http.Client{Timeout: cfg.Timeout}}, nil
}

func NewTelegram(cfg TelegramConfig) (Notifier, error) {
	if strings.TrimSpace(cfg.BotTokenEnv) == "" {
		return nil, errors.New("telegram bot token env is empty")
	}
	if strings.TrimSpace(cfg.ChatID) == "" {
		return nil, errors.New("telegram chat id is empty")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	return &telegramNotifier{cfg: cfg, client: &http.Client{Timeout: cfg.Timeout}}, nil
}

func (n *webhookNotifier) Name() string { return "webhook" }

func (n *webhookNotifier) Send(ctx context.Context, msg model.Message) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return withRetry(3, func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.cfg.URL, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		if n.cfg.SecretEnv != "" {
			secret := os.Getenv(n.cfg.SecretEnv)
			if secret != "" {
				req.Header.Set("X-Runpost-Signature", signPayload(payload, secret))
			}
		}

		resp, err := n.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)

		if resp.StatusCode >= 300 {
			return fmt.Errorf("webhook status: %d", resp.StatusCode)
		}
		return nil
	})
}

func (n *telegramNotifier) Name() string { return "telegram" }

func (n *telegramNotifier) Send(ctx context.Context, msg model.Message) error {
	token := os.Getenv(n.cfg.BotTokenEnv)
	if token == "" {
		return fmt.Errorf("telegram env %s is empty", n.cfg.BotTokenEnv)
	}

	u := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	body := map[string]any{
		"chat_id":    n.cfg.ChatID,
		"text":       fmt.Sprintf("%s\n\n%s", msg.Title, msg.Body),
		"parse_mode": "Markdown",
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	return withRetry(3, func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := n.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)
		if resp.StatusCode >= 300 {
			return fmt.Errorf("telegram status: %d", resp.StatusCode)
		}
		return nil
	})
}

func withRetry(max int, fn func() error) error {
	if max < 1 {
		max = 1
	}
	var lastErr error
	wait := 200 * time.Millisecond
	for i := 0; i < max; i++ {
		if err := fn(); err != nil {
			lastErr = err
			time.Sleep(wait)
			wait *= 2
			continue
		}
		return nil
	}
	return lastErr
}

func signPayload(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	_, _ = h.Write(payload)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}
