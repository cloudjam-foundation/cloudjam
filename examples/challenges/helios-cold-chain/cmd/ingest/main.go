package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"codeberg.org/megakuul/cloudjam/examples/challenges/helios-cold-chain/internal/telemetry"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventtypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

type handler struct {
	events *eventbridge.Client
	bus    string
}

func main() {
	ctx := context.Background()
	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		panic(err)
	}
	lambda.Start((&handler{
		events: eventbridge.NewFromConfig(awsConfig),
		bus:    os.Getenv("EVENT_BUS_NAME"),
	}).handle)
}

func (h *handler) handle(ctx context.Context, request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	if request.RequestContext.HTTP.Method != http.MethodPost || strings.TrimRight(request.RawPath, "/") != "/telemetry" {
		return response(http.StatusNotFound, map[string]string{"error": "POST /telemetry is required"}), nil
	}
	var reading telemetry.Reading
	if err := json.Unmarshal([]byte(request.Body), &reading); err != nil {
		return response(http.StatusBadRequest, map[string]string{"error": "request body must be valid JSON"}), nil
	}
	if err := reading.Validate(); err != nil {
		return response(http.StatusBadRequest, map[string]string{"error": err.Error()}), nil
	}
	detail, err := json.Marshal(reading)
	if err != nil {
		return events.LambdaFunctionURLResponse{}, err
	}
	output, err := h.events.PutEvents(ctx, &eventbridge.PutEventsInput{Entries: []eventtypes.PutEventsRequestEntry{{
		Detail:       aws.String(string(detail)),
		DetailType:   aws.String("TemperatureReading"),
		EventBusName: aws.String(h.bus),
		Source:       aws.String("helios.coldchain"),
	}}})
	if err != nil {
		return events.LambdaFunctionURLResponse{}, fmt.Errorf("publish telemetry: %w", err)
	}
	if output.FailedEntryCount != 0 {
		return events.LambdaFunctionURLResponse{}, fmt.Errorf("publish telemetry: event was rejected")
	}
	return response(http.StatusAccepted, map[string]any{
		"accepted":    true,
		"shipment_id": reading.ShipmentID,
		"sequence":    reading.Sequence,
	}), nil
}

func response(status int, body any) events.LambdaFunctionURLResponse {
	encoded, _ := json.Marshal(body)
	return events.LambdaFunctionURLResponse{
		StatusCode: status,
		Headers:    map[string]string{"content-type": "application/json"},
		Body:       string(encoded),
	}
}
