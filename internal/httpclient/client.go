package httpclient

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

func New(proxyURL string) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if proxyURL != "" {
		u, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("invalid HTTP_PROXY: %w", err)
		}
		transport.Proxy = http.ProxyURL(u)
	}
	return &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}, nil
}
