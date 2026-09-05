package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type endpoints struct {
	bloodPressureIngest string
	bloodPressureQuery  string
	enterpriseAPI       string
}

func main() {
	var target endpoints
	flag.StringVar(&target.bloodPressureIngest, "blood-pressure-ingest", "", "blood-pressure ingestion base URL")
	flag.StringVar(&target.bloodPressureQuery, "blood-pressure-query", "", "blood-pressure query base URL")
	flag.StringVar(&target.enterpriseAPI, "enterprise-api", "", "enterprise API base URL")
	flag.Parse()

	if target.bloodPressureIngest == "" || target.bloodPressureQuery == "" || target.enterpriseAPI == "" {
		flag.Usage()
		os.Exit(2)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	if err := probe(client, target); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("both Day 2 pipelines passed")
}

func probe(client *http.Client, target endpoints) error {
	stamp := fmt.Sprintf("probe-%d", time.Now().UnixNano())
	if err := post(client, endpoint(target.bloodPressureIngest, "/post_data"), map[string]string{
		"no": stamp, "systolic": "126", "diastolic": "82",
	}); err != nil {
		return fmt.Errorf("blood-pressure ingestion: %w", err)
	}
	if err := getContains(client, endpoint(target.bloodPressureQuery, "/get_value")+"?no="+url.QueryEscape(stamp), stamp, "126", "82"); err != nil {
		return fmt.Errorf("blood-pressure query: %w", err)
	}
	if err := post(client, endpoint(target.enterpriseAPI, "/post_data"), map[string]string{
		"id": stamp, "column": "val4", "value": "ready",
	}); err != nil {
		return fmt.Errorf("enterprise ingestion: %w", err)
	}
	if err := getContains(client, endpoint(target.enterpriseAPI, "/get_value")+"?id="+url.QueryEscape(stamp), stamp, "ready"); err != nil {
		return fmt.Errorf("enterprise query: %w", err)
	}
	return nil
}

func post(client *http.Client, target string, value any) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	return send(client, request, nil)
}

func getContains(client *http.Client, target string, values ...string) error {
	request, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	return send(client, request, values)
}

func send(client *http.Client, request *http.Request, values []string) error {
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("%s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	for _, value := range values {
		if !bytes.Contains(body, []byte(value)) {
			return fmt.Errorf("response does not contain %q: %s", value, strings.TrimSpace(string(body)))
		}
	}
	return nil
}

func endpoint(base, path string) string {
	return strings.TrimRight(base, "/") + path
}
