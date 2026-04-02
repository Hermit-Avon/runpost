package capture

import "sync"

type TailBuffer struct {
	mu   sync.Mutex
	max  int
	data []byte
}

func NewTailBuffer(max int) *TailBuffer {
	if max < 0 {
		max = 0
	}
	return &TailBuffer{max: max}
}

func (t *TailBuffer) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.max == 0 {
		return len(p), nil
	}

	if len(p) >= t.max {
		t.data = append(t.data[:0], p[len(p)-t.max:]...)
		return len(p), nil
	}

	total := len(t.data) + len(p)
	if total > t.max {
		drop := total - t.max
		t.data = append([]byte{}, t.data[drop:]...)
	}
	t.data = append(t.data, p...)

	return len(p), nil
}

func (t *TailBuffer) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return string(t.data)
}
