package metrics

import (
	"sync/atomic"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
)

// WorkerStats holds per-worker local statistics.
// Each goroutine owns one instance — no locking needed.
type WorkerStats struct {
	Histogram   *hdrhistogram.Histogram
	TotalReqs   int64
	SuccessCnt  int64
	FailureCnt  int64
	TotalBytes  int64
	StatusCodes map[int]int64
}

// NewWorkerStats creates a new WorkerStats with its own histogram.
func NewWorkerStats() *WorkerStats {
	return &WorkerStats{
		// Highest trackable: 60s in microseconds, 3 significant figures
		Histogram:   hdrhistogram.New(1, 60_000_000, 3),
		StatusCodes: make(map[int]int64),
	}
}

// Record records the result of a single HTTP request.
func (ws *WorkerStats) Record(latencyUs int64, bytesRead int64, statusCode int, err error) {
	ws.TotalReqs++
	ws.TotalBytes += bytesRead

	if err != nil {
		ws.FailureCnt++
		return
	}

	ws.SuccessCnt++
	ws.StatusCodes[statusCode]++
	ws.Histogram.RecordValue(latencyUs)
}

// MergeWorkerStats combines multiple per-worker stats into a single Report.
func MergeWorkerStats(stats []*WorkerStats, duration time.Duration, totalReqs *atomic.Int64) *Report {
	merged := hdrhistogram.New(1, 60_000_000, 3)

	var totalBytes, successCnt, failureCnt int64
	statusCodes := make(map[int]int64)

	for _, ws := range stats {
		merged.Merge(ws.Histogram)
		totalBytes += ws.TotalBytes
		successCnt += ws.SuccessCnt
		failureCnt += ws.FailureCnt

		for code, count := range ws.StatusCodes {
			statusCodes[code] += count
		}
	}

	total := totalReqs.Load()
	durationSecs := duration.Seconds()
	if durationSecs <= 0 {
		durationSecs = 1
	}

	var avgLatency float64
	if successCnt > 0 {
		avgLatency = float64(merged.Mean()) / 1000.0
	}

	return &Report{
		TotalRequests: total,
		Success:       successCnt,
		Failures:      failureCnt,
		TotalBytes:    totalBytes,
		Duration:      duration,
		RPS:           float64(total) / durationSecs,
		ThroughputBP:  float64(totalBytes) / durationSecs,
		Latency: LatencyStats{
			P50: float64(merged.ValueAtQuantile(50)) / 1000.0,
			P90: float64(merged.ValueAtQuantile(90)) / 1000.0,
			P99: float64(merged.ValueAtQuantile(99)) / 1000.0,
			Max: float64(merged.Max()) / 1000.0,
			Min: float64(merged.Min()) / 1000.0,
			Avg: avgLatency,
		},
		StatusCodes: statusCodes,
	}
}
