package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"codeberg.org/megakuul/cloudjam/examples/challenges/borealis-field-observatory/internal/observation"
)

func main() {
	endpoint := flag.String("endpoint", "", "App Runner service URL")
	timeout := flag.Duration("timeout", 2*time.Minute, "maximum lake query wait")
	flag.Parse()
	if *endpoint == "" {
		flag.Usage()
		os.Exit(2)
	}

	reading := observation.Reading{
		ObservationID: fmt.Sprintf("probe-%d", time.Now().UnixNano()),
		Habitat:       "fern-house",
		HumidityPct:   82,
		CO2PPM:        1500,
		RecordedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	if err := submit(context.Background(), *endpoint, reading); err != nil {
		fail(err)
	}
	if err := waitForObservation(context.Background(), *endpoint, reading.ObservationID, *timeout); err != nil {
		fail(err)
	}
	fmt.Println("Borealis pipeline passed")
}

func submit(ctx context.Context, base string, reading observation.Reading) error {
	body, err := json.Marshal(reading)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url(base, "/observations"), bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 15 * time.Second}).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		return fmt.Errorf("submit returned %s: %s", response.Status, strings.TrimSpace(string(message)))
	}
	return nil
}

func waitForObservation(ctx context.Context, base, id string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 20 * time.Second}
	for time.Now().Before(deadline) {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, url(base, "/observations/"+id), nil)
		if err != nil {
			return err
		}
		response, err := client.Do(request)
		if err != nil {
			return err
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		response.Body.Close()
		if readErr != nil {
			return readErr
		}
		if response.StatusCode == http.StatusOK {
			var reading observation.Reading
			if err := json.Unmarshal(body, &reading); err != nil {
				return err
			}
			if reading.ObservationID == id && reading.Condition == "attention" {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("observation %s was not queryable within %s", id, timeout)
}

func url(base, path string) string {
	return strings.TrimRight(base, "/") + path
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
