package providers

import "testing"

func TestParseCPUStat(t *testing.T) {
	busy, total, ok := parseCPUStat([]byte("cpu  100 20 30 800 50 0 0 0\ncpu0 1 2 3 4\n"))
	if !ok || busy != 150 || total != 1000 {
		t.Fatalf("busy=%d total=%d ok=%v", busy, total, ok)
	}
}
