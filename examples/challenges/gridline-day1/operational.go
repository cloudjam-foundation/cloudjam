//go:build wasip1

package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"codeberg.org/megakuul/cloudjam/examples/challenges/gridline-day1/internal/scoring"
	"codeberg.org/megakuul/cloudjam/pkg/challenge"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/api"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/ssm"
)

const endpointParameter = "/cloudjam/gridline/day1/endpoint-url"

func bootstrap(s *challenge.Scenario) error {
	for _, check := range scoring.Checks {
		if _, err := api.RegisterScore(api.RegisterScoreInput{
			Name: check.Name, Type: api.ScoreTypeOperational, Maximum: check.Points,
		}); err != nil {
			return err
		}
	}
	s.AddEvent("GridLine live evaluation", challenge.Event{
		Trigger: func() (bool, error) { return true, nil },
		Event: func(ctx context.Context, s *challenge.Scenario) error {
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()
			earned := make([]float64, len(scoring.Checks))
			for round := 1; round <= 360; round++ {
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
				results := scoring.Run(ctx, func(request api.SendHTTPInput) (api.SendHTTPOutput, error) {
					if configurationError != nil {
						return api.SendHTTPOutput{}, fmt.Errorf("read endpoint: %w", configurationError)
					}
					if base == "" {
						return api.SendHTTPOutput{}, fmt.Errorf("waiting for %s in eu-central-1; no request sent", endpointParameter)
					}
					request.URL = base + request.URL
					return s.SendHTTP(request)
				})
				for i, result := range results {
					if result.OK {
						earned[i] += result.Check.Points
					}
					if _, err := api.SubmitScore(api.SubmitScoreInput{
						Name: result.Check.Name, Type: api.ScoreTypeOperational,
						Score: earned[i], Maximum: float64(round) * result.Check.Points,
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
