package cmd

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/masbur/govelocity/internal/engine"
	"github.com/spf13/cobra"
)

var (
	targetURL   string
	concurrency int
	duration    int
	method      string
	clientOpt   string
	headers     []string
	queries     []string
)

var rootCmd = &cobra.Command{
	Use:   "govelocity",
	Short: "GoVelocity is a high-performance HTTP benchmarking tool",
	Long:  `A fast and lightweight HTTP benchmarking CLI tool built in Go`,
	Run: func(cmd *cobra.Command, args []string) {
		// Validate URL
		if targetURL == "" {
			fmt.Println("Error: target URL is required")
			cmd.Usage()
			os.Exit(1)
		}

		parsedURL, err := url.ParseRequestURI(targetURL)
		if err != nil {
			fmt.Printf("Error: invalid URL format: %v\n", err)
			os.Exit(1)
		}

		if len(queries) > 0 {
			q := parsedURL.Query()
			for _, query := range queries {
				parts := strings.SplitN(query, "=", 2)
				if len(parts) == 2 {
					q.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
				} else {
					q.Add(strings.TrimSpace(parts[0]), "")
				}
			}
			parsedURL.RawQuery = q.Encode()
			targetURL = parsedURL.String()
		}

		// Validate concurrency
		if concurrency < 1 {
			fmt.Println("Error: concurrency must be at least 1")
			os.Exit(1)
		}

		// Validate duration
		if duration < 1 {
			fmt.Println("Error: duration must be at least 1 second")
			os.Exit(1)
		}

		// Validate client flag
		if clientOpt != "net/http" && clientOpt != "fasthttp" {
			fmt.Println("Error: --client must be 'net/http' or 'fasthttp'")
			os.Exit(1)
		}

		fmt.Printf("Running %ds test @ %s\n", duration, targetURL)
		fmt.Printf("%d concurrent connections using %s\n", concurrency, clientOpt)

		if len(headers) > 0 {
			fmt.Printf("Using custom headers: %v\n", headers)
		}

		runner := engine.Runner{
			URL:         targetURL,
			Method:      method,
			Concurrency: concurrency,
			Duration:    time.Duration(duration) * time.Second,
			ClientOpt:   clientOpt,
			Headers:     headers,
		}

		report := runner.Run()
		report.Print()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&targetURL, "url", "u", "", "Target URL (e.g., http://localhost:8080)")
	rootCmd.Flags().IntVarP(&concurrency, "connections", "c", 10, "Number of concurrent connections")
	rootCmd.Flags().IntVarP(&duration, "duration", "d", 10, "Duration of the test in seconds")
	rootCmd.Flags().StringVarP(&method, "method", "m", "GET", "HTTP method to use")
	rootCmd.Flags().StringVar(&clientOpt, "client", "net/http", "HTTP client engine ('net/http' or 'fasthttp')")
	rootCmd.Flags().StringSliceVarP(&headers, "header", "H", []string{}, "Custom HTTP headers (e.g. -H \"Accept: text/html\" -H \"Authorization: Bearer token\")")
	rootCmd.Flags().StringSliceVarP(&queries, "query", "q", []string{}, "Custom query parameters (e.g. -q \"foo=bar\" -q \"baz=qux\")")

	rootCmd.MarkFlagRequired("url")
}
