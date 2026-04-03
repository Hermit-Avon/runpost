package config

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	NotifyOn string          `json:"notify_on"`
	Capture  CaptureConfig   `json:"capture"`
	Channels []ChannelConfig `json:"channels"`
	Template string          `json:"template"`
}

type CaptureConfig struct {
	MaxStdoutBytes int `json:"max_stdout_bytes"`
	MaxStderrBytes int `json:"max_stderr_bytes"`
}

type ChannelConfig struct {
	Type        string `json:"type"`
	URL         string `json:"url"`
	SecretEnv   string `json:"secret_env"`
	BotTokenEnv string `json:"bot_token_env"`
	ChatID      string `json:"chat_id"`
	Timeout     string `json:"timeout"`
}

func Default() Config {
	return Config{
		NotifyOn: "failure",
		Capture: CaptureConfig{
			MaxStdoutBytes: 65536,
			MaxStderrBytes: 65536,
		},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	if strings.TrimSpace(path) == "" {
		paths, err := defaultConfigFiles()
		if err != nil {
			return cfg, err
		}
		for _, p := range paths {
			if err := loadFileInto(&cfg, p); err != nil {
				return cfg, err
			}
		}
		applyDefaults(&cfg)
		return cfg, nil
	}

	if err := loadFileInto(&cfg, path); err != nil {
		return cfg, err
	}
	applyDefaults(&cfg)
	return cfg, nil
}

func defaultConfigFiles() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	pattern := filepath.Join(home, ".config", "runpost", "*.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	return matches, nil
}

func loadFileInto(cfg *Config, path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".json" {
		if err := json.Unmarshal(b, cfg); err != nil {
			return fmt.Errorf("parse json config %s: %w", path, err)
		}
		return nil
	}

	if err := parseSimpleYAML(string(b), cfg); err != nil {
		return fmt.Errorf("parse yaml config %s: %w", path, err)
	}
	return nil
}

func ChannelTimeout(raw string) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 5 * time.Second
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 5 * time.Second
	}
	return d
}

func applyDefaults(cfg *Config) {
	if cfg.NotifyOn == "" {
		cfg.NotifyOn = "failure"
	}
	if cfg.Capture.MaxStdoutBytes <= 0 {
		cfg.Capture.MaxStdoutBytes = 65536
	}
	if cfg.Capture.MaxStderrBytes <= 0 {
		cfg.Capture.MaxStderrBytes = 65536
	}
}

func parseSimpleYAML(src string, cfg *Config) error {
	s := bufio.NewScanner(strings.NewReader(src))
	section := ""
	var currentChannel *ChannelConfig

	for s.Scan() {
		line := strings.TrimRight(s.Text(), " \t")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		indent := leadingSpaces(line)
		trimmed := strings.TrimSpace(line)

		if indent == 0 {
			currentChannel = nil
			if strings.HasSuffix(trimmed, ":") {
				section = strings.TrimSuffix(trimmed, ":")
				continue
			}
			k, v, ok := splitKV(trimmed)
			if !ok {
				continue
			}
			if k == "notify_on" {
				cfg.NotifyOn = unquote(v)
			}
			continue
		}

		switch section {
		case "capture":
			k, v, ok := splitKV(trimmed)
			if !ok {
				continue
			}
			num, err := strconv.Atoi(unquote(v))
			if err != nil {
				return fmt.Errorf("invalid capture value for %s", k)
			}
			switch k {
			case "max_stdout_bytes":
				cfg.Capture.MaxStdoutBytes = num
			case "max_stderr_bytes":
				cfg.Capture.MaxStderrBytes = num
			}
		case "channels":
			if strings.HasPrefix(trimmed, "-") {
				ch := ChannelConfig{}
				cfg.Channels = append(cfg.Channels, ch)
				currentChannel = &cfg.Channels[len(cfg.Channels)-1]

				rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
				if rest != "" {
					k, v, ok := splitKV(rest)
					if ok {
						assignChannel(currentChannel, k, unquote(v))
					}
				}
				continue
			}

			if currentChannel == nil {
				return errors.New("invalid channels block")
			}
			k, v, ok := splitKV(trimmed)
			if !ok {
				continue
			}
			assignChannel(currentChannel, k, unquote(v))
		}
	}

	if err := s.Err(); err != nil {
		return err
	}
	return nil
}

func assignChannel(ch *ChannelConfig, key, value string) {
	switch key {
	case "type":
		ch.Type = value
	case "url":
		ch.URL = value
	case "secret_env":
		ch.SecretEnv = value
	case "bot_token_env":
		ch.BotTokenEnv = value
	case "chat_id":
		ch.ChatID = value
	case "timeout":
		ch.Timeout = value
	}
}

func splitKV(s string) (string, string, bool) {
	idx := strings.Index(s, ":")
	if idx <= 0 {
		return "", "", false
	}
	k := strings.TrimSpace(s[:idx])
	v := strings.TrimSpace(s[idx+1:])
	return k, v, true
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "\"")
	s = strings.Trim(s, "'")
	return s
}

func leadingSpaces(s string) int {
	count := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			count++
			continue
		}
		break
	}
	return count
}
