package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_EmptyPath_NoDefaultDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.NotifyOn != "failure" {
		t.Fatalf("unexpected notify_on: %s", cfg.NotifyOn)
	}
	if cfg.Capture.MaxStdoutBytes != 65536 {
		t.Fatalf("unexpected capture.max_stdout_bytes: %d", cfg.Capture.MaxStdoutBytes)
	}
	if cfg.Capture.MaxStderrBytes != 65536 {
		t.Fatalf("unexpected capture.max_stderr_bytes: %d", cfg.Capture.MaxStderrBytes)
	}
}

func TestLoad_EmptyPath_LoadsAndMergesDefaultYamlFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cfgDir := filepath.Join(home, ".config", "runpost")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	first := "notify_on: always\ncapture:\n  max_stdout_bytes: 111\nchannels:\n  - type: webhook\n    url: https://a.example\n"
	second := "capture:\n  max_stderr_bytes: 222\nchannels:\n  - type: telegram\n    bot_token_env: BOT\n    chat_id: \"123\"\n"

	if err := os.WriteFile(filepath.Join(cfgDir, "10-base.yaml"), []byte(first), 0o644); err != nil {
		t.Fatalf("WriteFile first failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "20-extra.yaml"), []byte(second), 0o644); err != nil {
		t.Fatalf("WriteFile second failed: %v", err)
	}

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.NotifyOn != "always" {
		t.Fatalf("unexpected notify_on: %s", cfg.NotifyOn)
	}
	if cfg.Capture.MaxStdoutBytes != 111 {
		t.Fatalf("unexpected capture.max_stdout_bytes: %d", cfg.Capture.MaxStdoutBytes)
	}
	if cfg.Capture.MaxStderrBytes != 222 {
		t.Fatalf("unexpected capture.max_stderr_bytes: %d", cfg.Capture.MaxStderrBytes)
	}
	if len(cfg.Channels) != 2 {
		t.Fatalf("unexpected channels count: %d", len(cfg.Channels))
	}
	if cfg.Channels[0].Type != "webhook" || cfg.Channels[1].Type != "telegram" {
		t.Fatalf("unexpected channels order/types: %#v", cfg.Channels)
	}
}

func TestLoad_ExplicitPathStillWorks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "runpost.yaml")
	content := "notify_on: never\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.NotifyOn != "never" {
		t.Fatalf("unexpected notify_on: %s", cfg.NotifyOn)
	}
}
