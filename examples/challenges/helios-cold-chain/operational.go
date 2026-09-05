//go:build wasip1

package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"codeberg.org/megakuul/cloudjam/examples/challenges/helios-cold-chain/internal/telemetry"
	"codeberg.org/megakuul/cloudjam/pkg/challenge"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/ssm"
)

const (
	probeWindow        = 20
	minimumProbeRounds = 3
)

type liveEvaluator struct {
	scenario *challenge.Scenario

	lock      sync.Mutex
	lastProbe time.Time
	results   []bool
}

func addOperationalChecks(s *challenge.Scenario) {
	evaluator := &liveEvaluator{scenario: s}
	for _, band := range []struct {
		name      string
		threshold float64
	}{
		{name: "Complete pipeline sustains 60% successful probes", threshold: 0.60},
		{name: "Complete pipeline sustains 70% successful probes", threshold: 0.70},
		{name: "Complete pipeline sustains 80% successful probes", threshold: 0.80},
		{name: "Complete pipeline sustains 90% successful probes", threshold: 0.90},
	} {
		s.AddCheck(band.name, challenge.NewOperationalCheck(
			5,
			"This band measures the rolling success rate from telemetry ingestion through the result query",
			evaluator.atLeast(band.threshold),
		))
	}
}

func (e *liveEvaluator) atLeast(threshold float64) trigger {
	return func() (bool, error) {
		e.lock.Lock()
		defer e.lock.Unlock()

		if time.Since(e.lastProbe) >= 15*time.Second {
			configured, err := e.probe()
			e.lastProbe = time.Now()
			if !configured {
				return false, err
			}
			e.record(err == nil)
		}
		if len(e.results) < minimumProbeRounds {
			return false, nil
		}

		successes := 0
		for _, passed := range e.results {
			if passed {
				successes++
			}
		}
		return float64(successes)/float64(len(e.results)) >= threshold, nil
	}
}

func (e *liveEvaluator) record(result bool) {
	e.results = append(e.results, result)
	if len(e.results) > probeWindow {
		e.results = e.results[len(e.results)-probeWindow:]
	}
	message := "Operational probe batch failed"
	if result {
		message = "Operational probe batch succeeded"
	}
	if err := e.scenario.ReportProgress(message); err != nil {
		slog.Error(fmt.Sprintf("report operational probe: %v", err))
	}
}

func (e *liveEvaluator) probe() (bool, error) {
	ingest, query, err := submittedEndpoints()
	if err != nil {
		return false, err
	}
	if ingest == "" || query == "" {
		return false, nil
	}

	reading := telemetry.Reading{
		ShipmentID:   fmt.Sprintf("cloudjam-%d", time.Now().UnixNano()),
		Sequence:     time.Now().UnixNano(),
		TemperatureC: 12,
		RecordedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	body, err := json.Marshal(reading)
	if err != nil {
		return true, err
	}
	request := challenge.NewHTTPRequest(http.MethodPost, endpoint(ingest, "/telemetry"), body)
	request.Headers["Content-Type"] = []string{"application/json"}
	request.TimeoutMillis = 5_000
	response, err := e.scenario.SendHTTP(request)
	if err != nil {
		return true, fmt.Errorf("telemetry ingestion: %w", err)
	}
	if response.StatusCode != http.StatusAccepted {
		return true, fmt.Errorf("telemetry ingestion: HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(response.Body)))
	}

	values := url.Values{
		"sequence":    []string{strconv.FormatInt(reading.Sequence, 10)},
		"shipment_id": []string{reading.ShipmentID},
	}
	resultURL := endpoint(query, "/result") + "?" + values.Encode()
	for range 60 {
		request := challenge.NewHTTPRequest(http.MethodGet, resultURL, nil)
		request.TimeoutMillis = 5_000
		response, err := e.scenario.SendHTTP(request)
		if err != nil {
			return true, fmt.Errorf("result query: %w", err)
		}
		if response.StatusCode == http.StatusOK {
			var result telemetry.Result
			if err := json.Unmarshal(response.Body, &result); err != nil {
				return true, fmt.Errorf("result query: %w", err)
			}
			if result.ShipmentID == reading.ShipmentID && result.Sequence == reading.Sequence && result.Status == "excursion" {
				return true, nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return true, fmt.Errorf("result %s/%d was not available within two minutes", reading.ShipmentID, reading.Sequence)
}

func submittedEndpoints() (string, string, error) {
	ingest, err := parameterValue(ingestEndpointParameter)
	if err != nil {
		return "", "", err
	}
	query, err := parameterValue(queryEndpointParameter)
	return ingest, query, err
}

func parameterValue(name string) (string, error) {
	parameters, err := listResources[*ssm.Parameter]()
	if err != nil {
		return "", err
	}
	parameter, exists := parameters[name]
	if !exists {
		return "", nil
	}
	if parameter.Value == nil {
		parameter, err = readResource[*ssm.Parameter](name)
		if err != nil {
			return "", err
		}
	}
	if parameter.Value != nil {
		return *parameter.Value, nil
	}
	return "", nil
}
func endpoint(base, path string) string {
	return strings.TrimRight(base, "/") + path
}
