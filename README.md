# GoVelocity

GoVelocity is a high-performance HTTP benchmarking CLI tool written in Go. It is designed to be fast, lightweight, and capable of simulating hundreds of thousands of requests per second concurrently.

## Features

- **High Concurrency**: Utilizes a Worker Pool pattern to manage thousands of simultaneous connections efficiently.
- **Dual HTTP Clients**: Provides the option to use the standard `net/http` client or the highly optimized `fasthttp` client for maximum throughput.
- **Accurate Metrics**: Records latency percentiles (P50, P90, P99, Max, Min, Avg) accurately without memory allocation overhead using `hdrhistogram`.
- **Real-time Progress**: Displays a live progress bar during benchmarking.
- **Customization**: Supports custom HTTP headers (`-H`) and URL query parameters (`-q`).
- **Race-Free Architecture**: Operates on a per-worker stats architecture, eliminating channel contention and data races for peak performance.

## Installation

Ensure you have Go installed (version 1.25+ recommended).

```bash
git clone https://github.com/masbur/govelocity.git
cd govelocity
go build -o govelocity
```

You can then move the `govelocity` binary to your system's PATH.

## Usage

Run `govelocity --help` to see all available options.

```bash
A fast and lightweight HTTP benchmarking CLI tool built in Go.

Usage:
  govelocity [flags]

Flags:
      --client string     HTTP client engine ('net/http' or 'fasthttp') (default "net/http")
  -c, --connections int   Number of concurrent connections (default 10)
  -d, --duration int      Duration of the test in seconds (default 10)
  -H, --header strings    Custom HTTP headers (e.g. -H "Accept: text/html" -H "Authorization: Bearer token")
  -h, --help              help for govelocity
  -m, --method string     HTTP method to use (default "GET")
  -q, --query strings     Custom query parameters (e.g. -q "foo=bar" -q "baz=qux")
  -u, --url string        Target URL (e.g., http://localhost:8080)
```

### Examples

**Basic Benchmark:**
Run a 10-second test with 100 concurrent connections against an API.

```bash
./govelocity -u http://localhost:8080/api/users -c 100 -d 10
```

**Use FastHTTP Client:**
Use the `fasthttp` engine for potentially higher throughput.

```bash
./govelocity -u http://localhost:8080/api/users -c 200 -d 30 --client fasthttp
```

**Custom Headers and Queries:**
Pass custom authorization headers and query parameters.

```bash
./govelocity -u https://api.example.com/data -c 50 -d 15 -H "Authorization: Bearer my-secret-token" -H "Accept: application/json" -q "sort=desc" -q "limit=50"
```

## Performance

GoVelocity is designed to minize overhead. In local loopback tests (100 connections, 5 seconds), it demonstrated the ability to process:

- **`net/http`**: ~196,000+ Requests Per Second
- **`fasthttp`**: ~325,000+ Requests Per Second

_(Note: Actual performance will vary greatly depending on network conditions and the target server's capabilities)._

## Architecture

- **Clean CLI**: Built using `spf13/cobra`.
- **Per-Worker Stats**: To avoid channel contention and locking overhead, each worker goroutine maintains its own local metrics and histogram. These are merged into a final report only after the benchmark completes.
- **Zero-Alloc Hot Path**: The engines (especially `fasthttp`) are tuned to minimize memory allocation in the tight request loop, relying on pre-allocated request objects via `Clone()` patterns.

## Development

A simple dummy HTTP server is included for testing purposes:

```bash
# Start the test server in the background
go run testserver/server.go &

# Run a benchmark against it
./govelocity -u http://localhost:8080 -c 50 -d 5

# Kill the test server when done
pkill -f "testserver/server.go"
```
