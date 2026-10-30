package providers

import (
	"testing"
	"time"
)

func TestCounterRate(t *testing.T) {
	if got := counterRate(100, 300, 2*time.Second); got != 100 {
		t.Fatalf("rate=%d", got)
	}
	if got := counterRate(300, 100, time.Second); got != 0 {
		t.Fatalf("counter reset rate=%d", got)
	}
	if got := counterRate(100, 200, 0); got != 0 {
		t.Fatalf("zero interval rate=%d", got)
	}
}
