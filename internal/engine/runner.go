package engine

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/masbur/govelocity/internal/metrics"
)

// Runner orchestrates the benchmarking.
type Runner struct {
	URL         string
	Method      string
	Concurrency int
	Duration    time.Duration
	ClientOpt   string
	Headers     []string
}

// Run starts the benchmark and returns a Report.
func (r *Runner) Run() *metrics.Report {
	// Initialize the prototype Client
	var prototype Client
	if r.ClientOpt == "fasthttp" {
		prototype = NewFastHTTPClient(r.Concurrency, r.Headers)
	} else {
		prototype = NewNetHTTPClient(r.Concurrency, r.Headers)
	}

	if err := prototype.Init(r.Method, r.URL); err != nil {
		fmt.Printf("Error initializing client: %v\n", err)
		os.Exit(1)
	}

	// Shared atomic counter for progress bar
	var totalReqs atomic.Int64

	// Per-worker stats — each worker owns one, no locking needed
	workerStats := make([]*metrics.WorkerStats, r.Concurrency)

	// Progress bar — runs in its own goroutine, reads atomic counter
	progressDone := make(chan struct{})
	go metrics.RunProgressBar(r.Duration, &totalReqs, progressDone)

	// Deadline for the test
	deadline := time.Now().Add(r.Duration)

	// Start Worker Pool
	var wg sync.WaitGroup
	wg.Add(r.Concurrency)

	for i := 0; i < r.Concurrency; i++ {
		ws := metrics.NewWorkerStats()
		workerStats[i] = ws

		// Each worker gets its own Client clone — no shared mutable state
		workerClient := prototype.Clone()

		go func() {
			defer wg.Done()

			for time.Now().Before(deadline) {
				start := time.Now()
				status, bytesRead, err := workerClient.Do()
				latencyUs := time.Since(start).Microseconds()

				ws.Record(latencyUs, bytesRead, status, err)
				totalReqs.Add(1)
			}
		}()
	}

	// Wait for all workers to finish
	wg.Wait()

	// Stop progress bar
	close(progressDone)

	// Merge all per-worker stats into a single Report
	return metrics.MergeWorkerStats(workerStats, r.Duration, &totalReqs)
}
