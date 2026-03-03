package engine

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

type httpHeader struct {
	key   string
	value string
}

func parseHeaders(rawHeaders []string) []httpHeader {
	var headers []httpHeader
	for _, h := range rawHeaders {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			headers = append(headers, httpHeader{
				key:   strings.TrimSpace(parts[0]),
				value: strings.TrimSpace(parts[1]),
			})
		}
	}
	return headers
}

// Client is an interface that abstracts standard net/http and fasthttp.
// Each worker goroutine must own its own Client via Clone().
type Client interface {
	// Init prepares the client with target method and URL.
	Init(method, url string) error
	// Do executes a single HTTP request and returns (statusCode, bytesRead, error).
	Do() (int, int64, error)
	// Clone creates an independent copy of this Client, safe for a separate goroutine.
	Clone() Client
}

// ---------------------------------------------------------------------------
// net/http Client
// ---------------------------------------------------------------------------

// NetHTTPClient implements Client using standard net/http.
type NetHTTPClient struct {
	httpClient *http.Client
	headers    []httpHeader
	method     string
	url        string
}

// NewNetHTTPClient initializes an optimized standard HTTP client.
func NewNetHTTPClient(concurrency int, rawHeaders []string) *NetHTTPClient {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:   true,
		MaxIdleConns:        concurrency,
		MaxIdleConnsPerHost: concurrency,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		DisableCompression:  true, // Avoid gzip CPU overhead during benchmarking
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	return &NetHTTPClient{
		httpClient: client,
		headers:    parseHeaders(rawHeaders),
	}
}

// Init sets the target method and URL.
func (c *NetHTTPClient) Init(method, urlStr string) error {
	c.method = method
	c.url = urlStr
	return nil
}

// Clone returns an independent copy sharing the same http.Client (which is goroutine-safe).
func (c *NetHTTPClient) Clone() Client {
	return &NetHTTPClient{
		httpClient: c.httpClient, // http.Client is safe for concurrent use
		headers:    c.headers,    // read-only slice, safe to share
		method:     c.method,
		url:        c.url,
	}
}

// Do executes a single HTTP request. Builds the request inline to avoid Clone() overhead.
func (c *NetHTTPClient) Do() (int, int64, error) {
	req, err := http.NewRequestWithContext(context.Background(), c.method, c.url, nil)
	if err != nil {
		return 0, 0, err
	}

	// Set headers directly — no map copy, just iterating a small slice.
	for i := range c.headers {
		req.Header[c.headers[i].key] = []string{c.headers[i].value}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, 0, err
	}

	// Fully drain body to allow TCP connection reuse (keep-alive).
	written, _ := io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	return resp.StatusCode, written, nil
}

// ---------------------------------------------------------------------------
// fasthttp Client
// ---------------------------------------------------------------------------

// FastHTTPClient implements Client using highly optimized fasthttp.
type FastHTTPClient struct {
	client  *fasthttp.Client
	headers []httpHeader
	method  string
	url     string
	// Per-worker pre-allocated request — only accessed by the owning goroutine.
	cachedReq *fasthttp.Request
}

// NewFastHTTPClient initializes a fasthttp client.
func NewFastHTTPClient(concurrency int, rawHeaders []string) *FastHTTPClient {
	client := &fasthttp.Client{
		MaxConnsPerHost:               concurrency,
		ReadTimeout:                   30 * time.Second,
		WriteTimeout:                  30 * time.Second,
		TLSConfig:                     &tls.Config{InsecureSkipVerify: true},
		DisableHeaderNamesNormalizing: true, // Skip header normalization for speed
	}
	return &FastHTTPClient{
		client:  client,
		headers: parseHeaders(rawHeaders),
	}
}

// Init sets the target method and URL.
func (c *FastHTTPClient) Init(method, urlStr string) error {
	c.method = method
	c.url = urlStr
	return nil
}

// Clone returns an independent copy with its own pre-allocated request.
func (c *FastHTTPClient) Clone() Client {
	fc := &FastHTTPClient{
		client:  c.client, // fasthttp.Client is safe for concurrent use
		headers: c.headers,
		method:  c.method,
		url:     c.url,
	}
	// Pre-allocate request for this worker
	fc.cachedReq = fasthttp.AcquireRequest()
	fc.cachedReq.SetRequestURI(c.url)
	fc.cachedReq.Header.SetMethod(c.method)
	for _, h := range c.headers {
		fc.cachedReq.Header.Set(h.key, h.value)
	}
	return fc
}

// Do executes a single HTTP request using fasthttp — zero-alloc hot path.
func (c *FastHTTPClient) Do() (int, int64, error) {
	resp := fasthttp.AcquireResponse()

	err := c.client.Do(c.cachedReq, resp)
	if err != nil {
		fasthttp.ReleaseResponse(resp)
		return 0, 0, err
	}

	status := resp.StatusCode()
	bodyLen := int64(resp.Header.ContentLength())
	if bodyLen < 0 {
		bodyLen = int64(len(resp.Body()))
	}

	fasthttp.ReleaseResponse(resp)
	return status, bodyLen, nil
}
