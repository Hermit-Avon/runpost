package capture

import "testing"

func TestTailBuffer(t *testing.T) {
	b := NewTailBuffer(5)
	_, _ = b.Write([]byte("hello"))
	_, _ = b.Write([]byte("world"))

	if got := b.String(); got != "world" {
		t.Fatalf("expected world, got %q", got)
	}
}
