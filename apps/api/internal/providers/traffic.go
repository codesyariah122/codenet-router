package providers

import "time"

func counterRate(previous, current uint64, elapsed time.Duration) uint64 {
	if elapsed <= 0 || current < previous {
		return 0
	}
	return uint64(float64(current-previous) / elapsed.Seconds())
}
