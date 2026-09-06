package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"codeberg.org/megakuul/cloudjam/examples/challenges/gridline-day1/internal/scoring"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/api"
)

func main() {
	endpoint := flag.String("endpoint", "", "CloudFront base URL")
	rounds := flag.Int("rounds", 1, "number of one-minute scoring rounds")
	flag.Parse()
	base, err := url.Parse(*endpoint)
	if err != nil || base.Host == "" || base.Scheme != "https" && base.Scheme != "http" || *rounds < 1 {
		fmt.Fprintln(os.Stderr, "provide -endpoint with an HTTP(S) base URL and -rounds greater than zero")
		os.Exit(2)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	client := &http.Client{Timeout: 5 * time.Second}
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	encoder := json.NewEncoder(os.Stdout)
	failed := false
	for round := 0; round < *rounds; round++ {
		results := scoring.Run(ctx, func(input api.SendHTTPInput) (api.SendHTTPOutput, error) {
			request, err := http.NewRequestWithContext(ctx, input.Method, strings.TrimRight(*endpoint, "/")+input.URL, bytes.NewReader(input.Body))
			if err != nil {
				return api.SendHTTPOutput{}, err
			}
			request.Header = http.Header(input.Headers)
			response, err := client.Do(request)
			if err != nil {
				return api.SendHTTPOutput{}, err
			}
			defer response.Body.Close()
			body, err := io.ReadAll(io.LimitReader(response.Body, api.MaxHTTPBodySize+1))
			if len(body) > api.MaxHTTPBodySize {
				return api.SendHTTPOutput{}, fmt.Errorf("response exceeds %d bytes", api.MaxHTTPBodySize)
			}
			return api.SendHTTPOutput{StatusCode: response.StatusCode, Body: body}, err
		})
		for _, result := range results {
			if err := encoder.Encode(result); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			failed = failed || !result.OK
		}
		if round+1 < *rounds {
			select {
			case <-ctx.Done():
				os.Exit(1)
			case <-ticker.C:
			}
		}
	}
	if failed {
		os.Exit(1)
	}
}
