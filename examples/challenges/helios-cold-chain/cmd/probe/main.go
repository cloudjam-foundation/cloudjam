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
	"strconv"
	"strings"
	"time"

	"codeberg.org/megakuul/cloudjam/examples/challenges/helios-cold-chain/internal/telemetry"
)

func main() {
	ingestURL := flag.String("ingest-url", "", "ingestion Function URL")
	queryURL := flag.String("query-url", "", "query Function URL")
	timeout := flag.Duration("timeout", 2*time.Minute, "maximum result wait")
	flag.Parse()
	if *ingestURL == "" || *queryURL == "" {
		flag.Usage()
		os.Exit(2)
	}

	ctx := context.Background()
	sequence := time.Now().UnixNano()
	readings := []telemetry.Reading{
		{ShipmentID: "probe-safe", Sequence: sequence, TemperatureC: 5, RecordedAt: time.Now().UTC().Format(time.RFC3339)},
		{ShipmentID: "probe-alert", Sequence: sequence, TemperatureC: 12, RecordedAt: time.Now().UTC().Format(time.RFC3339)},
	}
	for _, reading := range readings {
		if err := submit(ctx, *ingestURL, reading); err != nil {
			fail(err)
		}
	}
	for index, reading := range readings {
		expected := "safe"
		if index == 1 {
			expected = "excursion"
		}
		if err := waitForResult(ctx, *queryURL, reading, expected, *timeout); err != nil {
			fail(err)
		}
	}
	fmt.Println("Helios pipeline passed")
}

func submit(ctx context.Context, base string, reading telemetry.Reading) error {
	body, err := json.Marshal(reading)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint(base, "/telemetry"), bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		return fmt.Errorf("ingestion returned %s: %s", response.Status, strings.TrimSpace(string(message)))
	}
	return nil
}

func waitForResult(ctx context.Context, base string, reading telemetry.Reading, expected string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 10 * time.Second}
	for time.Now().Before(deadline) {
		target := endpoint(base, "/result") + "?shipment_id=" + reading.ShipmentID + "&sequence=" + strconv.FormatInt(reading.Sequence, 10)
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
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
			var result telemetry.Result
			if err := json.Unmarshal(body, &result); err != nil {
				return err
			}
			if result.Status == expected && result.ShipmentID == reading.ShipmentID && result.Sequence == reading.Sequence {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("result %s/%d was not available within %s", reading.ShipmentID, reading.Sequence, timeout)
}

func endpoint(base, path string) string {
	return strings.TrimRight(base, "/") + path
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
