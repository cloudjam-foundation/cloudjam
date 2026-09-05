package challenge

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"codeberg.org/megakuul/cloudjam/pkg/challenge/api"
)

const defaultHTTPTimeout = 10 * time.Second
const maxHTTPTimeout = 30 * time.Second

func (c *Challenge) sendHTTP(ctx context.Context, input *api.SendHTTPInput) (*api.SendHTTPOutput, error) {
	if len(input.Body) > api.MaxHTTPBodySize {
		return nil, fmt.Errorf("http request bodies larger than %d bytes are not supported", api.MaxHTTPBodySize)
	}
	endpoint, err := url.Parse(input.URL)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}
	if endpoint.Scheme != "http" && endpoint.Scheme != "https" {
		return nil, fmt.Errorf("unsupported url scheme %q", endpoint.Scheme)
	}
	if endpoint.Host == "" {
		return nil, fmt.Errorf("url has no host")
	}

	timeout := defaultHTTPTimeout
	if input.TimeoutMillis > 0 {
		timeout = time.Duration(min(input.TimeoutMillis, maxHTTPTimeout.Milliseconds())) * time.Millisecond
	}
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, input.Method, endpoint.String(), bytes.NewReader(input.Body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	request.Header = http.Header(input.Headers).Clone()

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, api.MaxHTTPBodySize+1))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if len(body) > api.MaxHTTPBodySize {
		return nil, fmt.Errorf("http response bodies larger than %d bytes are not supported", api.MaxHTTPBodySize)
	}
	return &api.SendHTTPOutput{
		StatusCode: response.StatusCode,
		Headers:    map[string][]string(response.Header.Clone()),
		Body:       body,
	}, nil
}
