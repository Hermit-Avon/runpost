package policy

import (
	"testing"

	"github.com/Hermit-Avon/runpost/internal/model"
)

func TestShouldNotify(t *testing.T) {
	failure := model.CommandResult{ExitCode: 1}
	success := model.CommandResult{ExitCode: 0}
	timedOut := model.CommandResult{ExitCode: 124, TimedOut: true}

	if !ShouldNotify(NotifyAlways, success) {
		t.Fatal("always should notify")
	}
	if !ShouldNotify(NotifyFailure, failure) {
		t.Fatal("failure should notify on non-zero")
	}
	if ShouldNotify(NotifyFailure, success) {
		t.Fatal("failure should not notify on zero")
	}
	if !ShouldNotify(NotifyTimeout, timedOut) {
		t.Fatal("timeout should notify when timed out")
	}
	if ShouldNotify(NotifyTimeout, failure) {
		t.Fatal("timeout should not notify without timeout")
	}
	if ShouldNotify(NotifyNever, failure) {
		t.Fatal("never should not notify")
	}
}
