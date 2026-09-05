//go:build wasip1

package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"codeberg.org/megakuul/cloudjam/examples/challenges/borealis-field-observatory/internal/observation"
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
		{name: "Container-to-lake path sustains 60% successful probes", threshold: 0.60},
		{name: "Container-to-lake path sustains 70% successful probes", threshold: 0.70},
		{name: "Container-to-lake path sustains 80% successful probes", threshold: 0.80},
		{name: "Container-to-lake path sustains 90% successful probes", threshold: 0.90},
	} {
		s.AddCheck(band.name, challenge.NewOperationalCheck(
			5,
			"This band measures the rolling success rate from App Runner through IoT, Firehose, S3, Glue, and Athena",
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
	base, err := submittedEndpoint()
	if err != nil {
		return false, err
	}
	if base == "" {
		return false, nil
	}
	reading := observation.Reading{
		ObservationID: fmt.Sprintf("cloudjam-%d", time.Now().UnixNano()),
		Habitat:       "fern-house",
		HumidityPct:   82,
		CO2PPM:        1500,
		RecordedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	body, err := json.Marshal(reading)
	if err != nil {
		return true, err
	}
	request := challenge.NewHTTPRequest(http.MethodPost, endpoint(base, "/observations"), body)
	request.Headers["Content-Type"] = []string{"application/json"}
	request.TimeoutMillis = 15_000
	response, err := e.scenario.SendHTTP(request)
	if err != nil {
		return true, fmt.Errorf("observation submission: %w", err)
	}
	if response.StatusCode != http.StatusAccepted {
		return true, fmt.Errorf("observation submission: HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(response.Body)))
	}

	resultURL := endpoint(base, "/observations/"+reading.ObservationID)
	for range 60 {
		request := challenge.NewHTTPRequest(http.MethodGet, resultURL, nil)
		request.TimeoutMillis = 20_000
		response, err := e.scenario.SendHTTP(request)
		if err != nil {
			return true, fmt.Errorf("observation query: %w", err)
		}
		if response.StatusCode == http.StatusOK {
			var result observation.Reading
			if err := json.Unmarshal(response.Body, &result); err != nil {
				return true, fmt.Errorf("observation query: %w", err)
			}
			if result.ObservationID == reading.ObservationID && result.Condition == "attention" {
				return true, nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return true, fmt.Errorf("observation %s was not queryable within two minutes", reading.ObservationID)
}

func submittedEndpoint() (string, error) {
	parameters, err := listResources[*ssm.Parameter]()
	if err != nil {
		return "", err
	}
	parameter, exists := parameters[endpointParameter]
	if !exists {
		return "", nil
	}
	if parameter.Value == nil {
		parameter, err = readResource[*ssm.Parameter](endpointParameter)
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
