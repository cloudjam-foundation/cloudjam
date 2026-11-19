//go:build wasip1

package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
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
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/apigatewayv2"
)

//go:embed brief.md
var brief []byte

//go:embed clues/clue1.md
var clue1 []byte

var privkey, _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

var status atomic.Bool = atomic.Bool{}

type armPosition struct {
	Height    int64     `json:"height"`
	Timestamp time.Time `json:"timestamp"`
	Signature string    `json:"signature"`
}

func randomArmPosition() armPosition {
	randHeight, _ := rand.Int(rand.Reader, big.NewInt(100))
	timestamp, height := time.Now(), randHeight.Int64()
	signature, err := ecdsa.SignASN1(rand.Reader, privkey, fmt.Append(nil, timestamp.Unix(), height))
	if err != nil {
		panic(err)
	}
	return armPosition{
		Timestamp: timestamp,
		Height:    height,
		Signature: string(signature),
	}
}

func validateArmPosition(six, seven armPosition) error {
	if !ecdsa.VerifyASN1(&privkey.PublicKey, fmt.Append(nil, six.Timestamp.Unix(), six.Height), []byte(six.Signature)) {
		return fmt.Errorf("incorrect signature on six arm")
	} else if !ecdsa.VerifyASN1(&privkey.PublicKey, fmt.Append(nil, seven.Timestamp.Unix(), seven.Height), []byte(seven.Signature)) {
		return fmt.Errorf("incorrect signature on seven arm")
	}
	if six.Height > 50 && seven.Height <= 50 {
		return nil
	} else if six.Height <= 50 && seven.Height > 50 {
		return nil
	}
	return fmt.Errorf("cursed arm position (not slay bro)")
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
		if err := validateArmPosition(state.Six, state.Seven); err != nil {
			slog.Debug(fmt.Sprintf("67 state yields incorrect position: %v", err))
			return false, err
		}
		threshold := time.Now().Add(-5 * time.Minute)
		if state.Six.Timestamp.Before(threshold) || state.Seven.Timestamp.Before(threshold) {
			slog.Debug(fmt.Sprintf("67 state yields legacy position: %v", err))
			return false, fmt.Errorf("67 state is to old: %v", err)
		}
		return true, nil
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
