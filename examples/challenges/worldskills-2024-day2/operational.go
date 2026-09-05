//go:build wasip1

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

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
		{name: "Both pipelines sustain 60% successful probes", threshold: 0.60},
		{name: "Both pipelines sustain 70% successful probes", threshold: 0.70},
		{name: "Both pipelines sustain 80% successful probes", threshold: 0.80},
		{name: "Both pipelines sustain 90% successful probes", threshold: 0.90},
	} {
		s.AddCheck(band.name, challenge.NewOperationalCheck(
			5,
			"This band measures the combined rolling success rate of both submitted pipelines",
			evaluator.atLeast(band.threshold),
		))
	}
}

func (e *liveEvaluator) atLeast(threshold float64) trigger {
	return func() (bool, error) {
		e.lock.Lock()
		defer e.lock.Unlock()

		if time.Since(e.lastProbe) >= 15*time.Second {
			e.lastProbe = time.Now()
			configured, err := e.probe()
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
	endpoints, err := submittedEndpoints()
	if err != nil {
		return false, err
	}
	if endpoints.bloodPressureIngest == "" || endpoints.bloodPressureQuery == "" || endpoints.enterpriseAPI == "" {
		return false, nil
	}

	stamp := fmt.Sprintf("cloudjam-%d", time.Now().UnixNano())
	if err := e.post(endpoint(endpoints.bloodPressureIngest, "/post_data"), map[string]string{
		"no": stamp, "systolic": "126", "diastolic": "82",
	}); err != nil {
		return true, fmt.Errorf("blood-pressure ingestion: %w", err)
	}
	if err := e.eventuallyContains(endpoint(endpoints.bloodPressureQuery, "/get_value")+"?no="+url.QueryEscape(stamp), stamp, "126", "82"); err != nil {
		return true, fmt.Errorf("blood-pressure query: %w", err)
	}
	if err := e.post(endpoint(endpoints.enterpriseAPI, "/post_data"), map[string]string{
		"id": stamp, "column": "val4", "value": "ready",
	}); err != nil {
		return true, fmt.Errorf("enterprise ingestion: %w", err)
	}
	if err := e.eventuallyContains(endpoint(endpoints.enterpriseAPI, "/get_value")+"?id="+url.QueryEscape(stamp), stamp, "ready"); err != nil {
		return true, fmt.Errorf("enterprise query: %w", err)
	}
	return true, nil
}

type endpoints struct {
	bloodPressureIngest string
	bloodPressureQuery  string
	enterpriseAPI       string
}

func submittedEndpoints() (endpoints, error) {
	parameters, err := listResources[*ssm.Parameter]()
	if err != nil {
		return endpoints{}, err
	}

	values := map[string]string{}
	for identifier, parameter := range parameters {
		if parameter.Name == nil {
			continue
		}
		if parameter.Value == nil {
			parameter, err = readResource[*ssm.Parameter](identifier)
			if err != nil {
				return endpoints{}, err
			}
		}
		if parameter.Value != nil {
			values[*parameter.Name] = *parameter.Value
		}
	}
	return endpoints{
		bloodPressureIngest: values[bloodPressureIngestParameter],
		bloodPressureQuery:  values[bloodPressureQueryParameter],
		enterpriseAPI:       values[enterpriseAPIParameter],
	}, nil
}

func (e *liveEvaluator) post(target string, value any) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	request := challenge.NewHTTPRequest(http.MethodPost, target, body)
	request.Headers["Content-Type"] = []string{"application/json"}
	request.TimeoutMillis = 5_000
	response, err := e.scenario.SendHTTP(request)
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(response.Body)))
	}
	return nil
}

func (e *liveEvaluator) eventuallyContains(target string, values ...string) error {
	var last error
	for range 5 {
		request := challenge.NewHTTPRequest(http.MethodGet, target, nil)
		request.TimeoutMillis = 5_000
		response, err := e.scenario.SendHTTP(request)
		if err == nil && response.StatusCode >= 200 && response.StatusCode < 300 {
			missing := ""
			for _, value := range values {
				if !bytes.Contains(response.Body, []byte(value)) {
					missing = value
					break
				}
			}
			if missing == "" {
				return nil
			}
			last = fmt.Errorf("response does not contain %q: %s", missing, strings.TrimSpace(string(response.Body)))
		} else if err != nil {
			last = err
		} else {
			last = fmt.Errorf("HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(response.Body)))
		}
		time.Sleep(2 * time.Second)
	}
	return last
}

func endpoint(base, path string) string {
	return strings.TrimRight(base, "/") + path
}
