package http

import "time"

type HttpOpts func(*httpConfig)

func WithConnClientTimeout(timeout time.Duration) HttpOpts {
	return func(c *httpConfig) {
		c.connClientTimeout = timeout
	}
}

func WithRequestTimeout(timeout time.Duration) HttpOpts {
	return func(c *httpConfig) {
		c.requestTimeout = timeout
	}
}

func WithClientKeepAlive(keepAlive time.Duration) HttpOpts {
	return func(c *httpConfig) {
		c.clientKeepAlive = keepAlive
	}
}

func WithTLSHandshakeTimeout(timeout time.Duration) HttpOpts {
	return func(c *httpConfig) {
		c.tlsHandshakeTimeout = timeout
	}
}

func WithResponseHeaderTimeout(timeout time.Duration) HttpOpts {
	return func(c *httpConfig) {
		c.responseHeaderTimeout = timeout
	}
}

func WithIdleConnTimeout(timeout time.Duration) HttpOpts {
	return func(c *httpConfig) {
		c.idleConnTimeout = timeout
	}
}

func WithMaxIdleConns(maxConns int) HttpOpts {
	return func(c *httpConfig) {
		c.maxIdleConns = maxConns
	}
}

func WithMaxIdleConnsPerHost(maxConns int) HttpOpts {
	return func(c *httpConfig) {
		c.maxIdleConnsPerHost = maxConns
	}
}

func WithTransport(transport TransportFunc) HttpOpts {
	return func(c *httpConfig) {
		c.transports = append(c.transports, transport)
	}
}

func WithInsecureSkipVerify(skip bool) HttpOpts {
	return func(c *httpConfig) {
		c.insecureSkipVerify = skip
	}
}
