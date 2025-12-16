package http

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

type TransportFunc func(http.RoundTripper) http.RoundTripper

type httpConfig struct {
	connClientTimeout     time.Duration
	requestTimeout        time.Duration
	clientKeepAlive       time.Duration
	tlsHandshakeTimeout   time.Duration
	responseHeaderTimeout time.Duration
	idleConnTimeout       time.Duration
	maxIdleConns          int
	maxIdleConnsPerHost   int
	transports            []TransportFunc
	insecureSkipVerify    bool
}

func defaultHTTPConfig() *httpConfig {
	return &httpConfig{
		connClientTimeout:     30 * time.Second,
		requestTimeout:        30 * time.Second,
		clientKeepAlive:       90 * time.Second,
		tlsHandshakeTimeout:   10 * time.Second,
		responseHeaderTimeout: 10 * time.Second,
		idleConnTimeout:       90 * time.Second,
		maxIdleConns:          100,
		maxIdleConnsPerHost:   10,
		transports:            []TransportFunc{},
		insecureSkipVerify:    false,
	}
}

func newClient(opts ...HttpOpts) *http.Client {
	cfg := defaultHTTPConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return newInternal(cfg)
}

func newInternal(cfg *httpConfig) *http.Client {
	dialer := net.Dialer{
		Timeout:   cfg.connClientTimeout,
		KeepAlive: cfg.clientKeepAlive,
	}

	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		MaxIdleConns:          cfg.maxIdleConns,
		MaxIdleConnsPerHost:   cfg.maxIdleConnsPerHost,
		TLSHandshakeTimeout:   cfg.tlsHandshakeTimeout,
		ResponseHeaderTimeout: cfg.responseHeaderTimeout,
		IdleConnTimeout:       cfg.idleConnTimeout,
	}

	if cfg.insecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	client := &http.Client{
		Timeout:   cfg.requestTimeout,
		Transport: transport,
	}

	if len(cfg.transports) != 0 {
		client = applyTransport(client, cfg.transports...)
	}

	return client
}

func applyTransport(client *http.Client, transports ...TransportFunc) *http.Client {
	transport := client.Transport

	if transport == nil {
		transport = http.DefaultTransport
	}

	for _, transportFunc := range transports {
		transport = transportFunc(transport)
	}

	clone := *client
	clone.Transport = transport

	return &clone
}
