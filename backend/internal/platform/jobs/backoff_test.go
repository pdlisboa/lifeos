package jobs

import (
	"testing"
	"time"
)

func TestNextRunAt(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		attempts int16
		want     time.Duration
	}{
		{0, 1 * time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{10, 1024 * time.Second}, // ~17min, ainda abaixo do teto de 30min
	}
	for _, c := range cases {
		got := nextRunAt(now, c.attempts)
		if gotDelay := got.Sub(now); gotDelay != c.want {
			t.Errorf("attempts=%d: delay=%s, esperava %s", c.attempts, gotDelay, c.want)
		}
	}
}

func TestNextRunAtIsCapped(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	got := nextRunAt(now, 20)
	if delay := got.Sub(now); delay != maxBackoff {
		t.Fatalf("esperava teto de %s, teve %s", maxBackoff, delay)
	}
}

func TestNextRunAtMonotonicUntilCap(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	var prev time.Duration
	for attempts := int16(0); attempts < 15; attempts++ {
		d := nextRunAt(now, attempts).Sub(now)
		if d < prev {
			t.Fatalf("backoff não deveria diminuir: attempts=%d d=%s prev=%s", attempts, d, prev)
		}
		prev = d
	}
}
