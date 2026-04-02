package model

import "time"

type CommandResult struct {
	Command    []string
	StartAt    time.Time
	EndAt      time.Time
	Duration   time.Duration
	ExitCode   int
	TimedOut   bool
	StdoutTail string
	StderrTail string
	Err        string
}

type Message struct {
	Title      string
	Body       string
	Level      string
	Tags       map[string]string
	OccurredAt time.Time
}
