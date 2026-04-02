package formatter

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/Hermit-Avon/runpost/internal/model"
)

const defaultTemplate = `Command: {{.CommandLine}}
ExitCode: {{.ExitCode}}
Duration: {{.Duration}}
TimedOut: {{.TimedOut}}
StartAt: {{.StartAt}}
EndAt: {{.EndAt}}

stdout tail:
{{.StdoutTail}}

stderr tail:
{{.StderrTail}}
`

type Data struct {
	CommandLine string
	ExitCode    int
	Duration    string
	TimedOut    bool
	StartAt     string
	EndAt       string
	StdoutTail  string
	StderrTail  string
}

func Format(result model.CommandResult, tpl string) (model.Message, error) {
	if strings.TrimSpace(tpl) == "" {
		tpl = defaultTemplate
	}

	data := Data{
		CommandLine: strings.Join(result.Command, " "),
		ExitCode:    result.ExitCode,
		Duration:    result.Duration.String(),
		TimedOut:    result.TimedOut,
		StartAt:     result.StartAt.Format("2006-01-02 15:04:05"),
		EndAt:       result.EndAt.Format("2006-01-02 15:04:05"),
		StdoutTail:  emptyFallback(result.StdoutTail),
		StderrTail:  emptyFallback(result.StderrTail),
	}

	t, err := template.New("message").Parse(tpl)
	if err != nil {
		return model.Message{}, err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return model.Message{}, err
	}

	level := "info"
	title := fmt.Sprintf("runpost: success (%s)", data.CommandLine)
	if result.ExitCode != 0 || result.TimedOut {
		level = "error"
		title = fmt.Sprintf("runpost: failed (%s)", data.CommandLine)
	}

	return model.Message{
		Title:      title,
		Body:       buf.String(),
		Level:      level,
		OccurredAt: result.EndAt,
		Tags: map[string]string{
			"exit_code": fmt.Sprintf("%d", result.ExitCode),
			"timed_out": fmt.Sprintf("%t", result.TimedOut),
		},
	}, nil
}

func emptyFallback(s string) string {
	if strings.TrimSpace(s) == "" {
		return "<empty>"
	}
	return s
}
