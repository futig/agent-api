package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"go.uber.org/zap"
)

type Connector struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

type ConnectorConfig struct {
	BaseURL string
	Logger  *zap.Logger
}

func NewConnector(config *ConnectorConfig, options ...HttpOpts) *Connector {
	return &Connector{
		baseURL:    config.BaseURL,
		httpClient: newClient(options...),
		logger:     config.Logger,
	}
}

type RequestOpt func(*requestConfig)

type requestConfig struct {
	headers     map[string]string
	overrideURL string
}

func WithHeader(key, value string) RequestOpt {
	return func(c *requestConfig) {
		if c.headers == nil {
			c.headers = make(map[string]string)
		}
		c.headers[key] = value
	}
}

func WithURL(url string) RequestOpt {
	return func(c *requestConfig) {
		c.overrideURL = url
	}
}

func (c *Connector) DoRequest(ctx context.Context, method, endpoint string, reqBody, respBody any, opts ...RequestOpt) error {
	// Apply request options
	cfg := &requestConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Use override URL if provided, otherwise use baseURL + endpoint
	var url string
	if cfg.overrideURL != "" {
		url = cfg.overrideURL
	} else {
		url = c.baseURL + endpoint
	}

	var bodyReader io.Reader
	var rawBody []byte
	if reqBody != nil {
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		rawBody = jsonData
		bodyReader = bytes.NewReader(jsonData)
		// Attach payload to context for logging transport
		ctx = context.WithValue(ctx, payloadContextKey{}, rawBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// Set default headers
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	// Add custom headers
	for key, value := range cfg.headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &NetworkError{Err: err}
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	// Decode response if needed
	if respBody != nil && len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, respBody); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// doSingleMultipartRequest performs a single multipart request
func (c *Connector) DoMultipartRequest(ctx context.Context, method, endpoint string, prepareBody func(*multipart.Writer) error, respBody any, opts ...RequestOpt) error {
	// Apply request options
	cfg := &requestConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Use override URL if provided, otherwise use baseURL + endpoint
	var url string
	if cfg.overrideURL != "" {
		url = cfg.overrideURL
	} else {
		url = c.baseURL + endpoint
	}

	// Prepare multipart body
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := prepareBody(writer); err != nil {
		return fmt.Errorf("prepare multipart body: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")

	// Add custom headers
	for key, value := range cfg.headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &NetworkError{Err: err}
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	if respBody != nil && len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, respBody); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// HTTPError represents an HTTP error response
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// NetworkError represents a network-level error (connection, timeout, etc.)
type NetworkError struct {
	Err error
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("network error: %v", e.Err)
}

func (e *NetworkError) Unwrap() error {
	return e.Err
}
