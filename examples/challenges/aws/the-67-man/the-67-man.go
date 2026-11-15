//go:build wasip1

package main

import (
	"crypto/rand"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"sync/atomic"
	"time"

	"codeberg.org/megakuul/cloudjam/pkg/challenge"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/api"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/policy"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/apigateway"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/apigatewayv2"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/dynamodb"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/lambda"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/sqs"
)

//go:embed brief.md
var brief []byte

//go:embed clues/clue1.md
var clue1 []byte

var permissions = policy.Document{
	Version: policy.Version20121017,
	Statement: []policy.Statement{
		{
			Sid:      "ApiGatewayAccess",
			Effect:   policy.Allow,
			Action:   policy.ActionsFrom(apigateway.ActionsList, apigateway.ActionsRead, apigateway.ActionsWrite),
			Resource: policy.ARNsFrom("arn:aws:apigateway:*:*:*"),
		},
		{
			Sid:      "SQSAccess",
			Effect:   policy.Allow,
			Action:   policy.ActionsFrom(sqs.ActionsTagging, sqs.ActionsRead, sqs.ActionsWrite),
			Resource: policy.ARNsFrom("arn:aws:sqs:*:*:*"),
		},
		{
			Sid:      "LambdaAccess",
			Effect:   policy.Allow,
			Action:   policy.ActionsFrom(lambda.ActionsList, lambda.ActionsTagging, lambda.ActionsRead, lambda.ActionsWrite),
			Resource: policy.ARNsFrom("arn:aws:lambda:*:*:*"),
		},
		{
			Sid:      "DynamodbAccess",
			Effect:   policy.Allow,
			Action:   policy.ActionsFrom(dynamodb.ActionsList, dynamodb.ActionsTagging, dynamodb.ActionsRead, dynamodb.ActionsWrite),
			Resource: policy.ARNsFrom("arn:aws:dynamodb:*:*:*"),
		},
	},
}

var status atomic.Bool = atomic.Bool{}

type armPosition struct {
	Height    int64     `json:"height"`
	Timestamp time.Time `json:"timestamp"`
	Signature string    `json:"signature"`
}

func randomArmPosition() armPosition {
	height, _ := rand.Int(rand.Reader, big.NewInt(100))
	return armPosition{
		Timestamp: time.Now(),
		Height:    height.Int64(),
		Signature: "", // TODO add signature
	}
}

func isValidPosition(six, seven armPosition) bool {
	// TODO check signature
	if six.Height > 50 && seven.Height <= 50 {
		return true
	} else if six.Height <= 50 && seven.Height > 50 {
		return true
	}
	return false
}

type updateSixRequest struct {
	State armPosition `json:"state"`
}

type updateSevenRequest struct {
	State armPosition `json:"state"`
}

type stateResponse struct {
	Six   armPosition `json:"six"`
	Seven armPosition `json:"seven"`
}

func bootstrap(s *challenge.Scenario) error {
	s.AddDescription(string(brief))
	s.AddClue("How to avoid incorrect arm position?", string(clue1), 100)

	id, err := aws.Create(apigatewayv2.Api{
		Name:        new("the-67-gateway"),
		Description: new("67"),
	})
	if err != nil {
		return fmt.Errorf("create api gateway: %w", err)
	}
	gateway, err := aws.Read[apigatewayv2.Api](id)
	if err != nil {
		return fmt.Errorf("fetch api gateway: %w", err)
	}
	endpoint := fmt.Sprintf("https://%s", *gateway.ApiEndpoint)

	s.AddCheck("Healthcheck", challenge.NewOperationalCheck(10, "Healthcheck reports success", func() (bool, error) {
		resp, err := s.SendHTTP(challenge.NewHTTPRequest("GET", fmt.Sprintf("%s/health", endpoint), nil))
		if err != nil {
			return false, err
		}
		slog.Debug(fmt.Sprintf("healthcheck reports status %q with body %q", http.StatusText(resp.StatusCode), string(resp.Body)))
		if resp.StatusCode == http.StatusOK {
			if status.Load() {
				return true, nil
			} else {
				_, err := api.UpdateScore(api.UpdateScoreInput{Increment: -20, Reason: "Incorrect healthcheck report"})
				return false, err
			}
		}
		return false, nil
	}))
	s.AddCheck("67 state", challenge.NewOperationalCheck(50, "the 67 force is in balance", func() (bool, error) {
		resp, err := s.SendHTTP(challenge.NewHTTPRequest("GET", fmt.Sprintf("%s/6", endpoint), nil))
		if err != nil {
			return false, err
		}
		slog.Debug(fmt.Sprintf("67 state yields status %q with body %q", http.StatusText(resp.StatusCode), string(resp.Body)))

		state := &stateResponse{}
		if err = json.Unmarshal(resp.Body, state); err != nil {
			return false, err
		}
		return isValidPosition(state.Six, state.Seven), nil
	}))
	s.AddCheck("67 writes", challenge.NewOperationalCheck(15, "67 write succeeded", func() (bool, error) {
		rawRequest, err := json.Marshal(updateSixRequest{State: randomArmPosition()})
		resp, err := s.SendHTTP(challenge.NewHTTPRequest("POST", fmt.Sprintf("%s/6", endpoint), rawRequest))
		if err != nil {
			status.Store(false)
			return false, err
		}
		slog.Debug(fmt.Sprintf("6 write status %q with body %q", http.StatusText(resp.StatusCode), string(resp.Body)))
		if resp.StatusCode != http.StatusOK {
			status.Store(false)
			return false, nil
		}

		rawRequest, err = json.Marshal(updateSevenRequest{State: randomArmPosition()})
		resp, err = s.SendHTTP(challenge.NewHTTPRequest("POST", fmt.Sprintf("%s/7", endpoint), rawRequest))
		if err != nil {
			status.Store(false)
			return false, err
		}
		slog.Debug(fmt.Sprintf("7 write status %q with body %q", http.StatusText(resp.StatusCode), string(resp.Body)))
		if resp.StatusCode != http.StatusOK {
			status.Store(false)
			return false, nil
		}
		status.Store(true)
		return true, nil
	}))

	// TODO add checks to analyze tags, etc from API gateway
	// TODO add checks for sqs queue
	return nil
}

func main() {
	s := challenge.New("The 67 Man", 10*time.Second, bootstrap)
	s.SetGuardrail(permissions)
	s.SetPermission(permissions)
	s.Start()
}
