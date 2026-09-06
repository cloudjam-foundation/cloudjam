//go:build wasip1

package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"codeberg.org/megakuul/cloudjam/examples/challenges/vitalforge-day2/internal/scoring"
	"codeberg.org/megakuul/cloudjam/pkg/challenge"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/api"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/ssm"
)

const endpointParameter = "/cloudjam/vitalforge/day2/endpoint-url"

func bootstrap(s *challenge.Scenario) error {
	started := time.Now()
	design, err := addDesignChecks(s, started)
	if err != nil {
		return err
	}
	s.AddEvent("VitalForge live evaluation", challenge.Event{
		Trigger: func() (bool, error) { return true, nil },
		Event: func(ctx context.Context, s *challenge.Scenario) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Until(started.Add(50 * time.Minute))):
			}
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()
			for round := 1; round <= 310; round++ {
				base := ""
				parameters, err := aws.List[*ssm.Parameter]()
				if err == nil {
					if _, exists := parameters[endpointParameter]; exists {
						var parameter *ssm.Parameter
						parameter, err = aws.Read[*ssm.Parameter](endpointParameter)
						if err == nil && parameter.Value != nil {
							base = strings.TrimRight(strings.TrimSpace(*parameter.Value), "/")
						}
					}
				}
				configurationError := err
				design.lock.Lock()
				design.evidence.Endpoint = base
				design.lock.Unlock()
				results := scoring.Run(ctx, func(request api.SendHTTPInput) (api.SendHTTPOutput, error) {
					if configurationError != nil {
						return api.SendHTTPOutput{}, fmt.Errorf("read endpoint: %w", configurationError)
					}
					if base == "" {
						return api.SendHTTPOutput{}, fmt.Errorf("waiting for %s in the game region; no request sent", endpointParameter)
					}
					request.URL = base + request.URL
					return s.SendHTTP(request)
				})
				design.lock.Lock()
				for _, result := range results {
					design.evidence.Checks[result.Check.ID] = result
				}
				design.lock.Unlock()
				for _, result := range results {
					score := 0.0
					if result.OK {
						score = result.Check.Points
					}
					if _, err := api.SubmitScore(api.SubmitScoreInput{
						Name: result.Check.Name, Type: api.ScoreTypeOperational,
						Score: score, Maximum: result.Check.Points, Accumulate: true,
						Reason: fmt.Sprintf("Round %d | %s", round, result.Detail),
					}); err != nil {
						slog.Error(fmt.Sprintf("submit %s: %v", result.Check.ID, err))
					}
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-ticker.C:
				}
			}
			return nil
		},
	})
	return nil
}
