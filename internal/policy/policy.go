package policy

import "github.com/Hermit-Avon/runpost/internal/model"

const (
	NotifyAlways  = "always"
	NotifyFailure = "failure"
	NotifyTimeout = "timeout"
	NotifyNever   = "never"
)

func ShouldNotify(mode string, result model.CommandResult) bool {
	switch mode {
	case NotifyAlways:
		return true
	case NotifyFailure:
		return result.ExitCode != 0
	case NotifyTimeout:
		return result.TimedOut
	case NotifyNever:
		return false
	default:
		return result.ExitCode != 0
	}
}
