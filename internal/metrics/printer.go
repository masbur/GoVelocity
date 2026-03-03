package metrics

import (
	"fmt"
	"os"
	"text/tabwriter"
)

// Print prints the structured benchmark report to stdout
func (r *Report) Print() {
	fmt.Println("\n==============================================")
	fmt.Println("             GoVelocity Results             ")
	fmt.Println("==============================================")

	// Summary info
	fmt.Printf("Total Requests: %d\n", r.TotalRequests)
	fmt.Printf("Duration:       %v\n", r.Duration)
	fmt.Printf("Success:        %d\n", r.Success)
	fmt.Printf("Failures:       %d\n", r.Failures)
	fmt.Printf("Total Bytes:    %s\n", formatBytes(r.TotalBytes))
	fmt.Printf("Avg RPS:        %.2f\n", r.RPS)
	fmt.Printf("Throughput:     %s/s\n\n", formatBytes(int64(r.ThroughputBP)))

	// Latency Table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.Debug)
	fmt.Fprintln(w, "Stat\tLatency (ms)")
	fmt.Fprintln(w, "----\t------------")
	fmt.Fprintf(w, "Avg\t%.2f\n", r.Latency.Avg)
	fmt.Fprintf(w, "Min\t%.2f\n", r.Latency.Min)
	fmt.Fprintf(w, "Max\t%.2f\n", r.Latency.Max)
	fmt.Fprintf(w, "P50\t%.2f\n", r.Latency.P50)
	fmt.Fprintf(w, "P90\t%.2f\n", r.Latency.P90)
	fmt.Fprintf(w, "P99\t%.2f\n", r.Latency.P99)
	w.Flush()

	fmt.Println()

	// Status Codes
	if len(r.StatusCodes) > 0 {
		fmt.Println("Status Codes:")
		for code, count := range r.StatusCodes {
			fmt.Printf("  [%d] %d responses\n", code, count)
		}
	}
	fmt.Println("==============================================")
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
