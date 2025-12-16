package http

import "net/http"

type authTransport struct {
	token     string
	transport http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqCopy := req.Clone(req.Context())

	if t.token != "" {
		reqCopy.Header.Set("Authorization", "Bearer "+t.token)
	}

	return t.transport.RoundTrip(reqCopy)
}

func WithAuthToken(token string) HttpOpts {
	return WithTransport(func(rt http.RoundTripper) http.RoundTripper {
		return &authTransport{
			token:     token,
			transport: rt,
		}
	})
}
