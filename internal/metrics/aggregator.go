package metrics

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/schollz/progressbar/v3"
)

// Report contains the final statistics of the benchmark
type Report struct {
	TotalRequests int64
	Success       int64
	Failures      int64
	TotalBytes    int64
	Duration      time.Duration
	RPS           float64
	ThroughputBP  float64
	Latency       LatencyStats
	StatusCodes   map[int]int64
}

// LatencyStats holds percentile data for latencies (in milliseconds)
type LatencyStats struct {
	P50 float64
	P90 float64
	P99 float64
	Max float64
	Min float64
	Avg float64
}

// RunProgressBar starts a progress bar in a separate goroutine.
// It reads from an atomic counter and stops when the done channel is closed.
func RunProgressBar(duration time.Duration, counter *atomic.Int64, done <-chan struct{}) {
	totalSeconds := int64(duration.Seconds())
	bar := progressbar.Default(totalSeconds, "Benchmarking")

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	elapsed := int64(0)
	for {
		select {
		case <-done:
			bar.Set64(totalSeconds)
			bar.Finish()
			fmt.Println()
			return
		case <-ticker.C:
			elapsed++
			bar.Set64(elapsed)
		}
	}
}
