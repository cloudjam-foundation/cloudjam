package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"codeberg.org/megakuul/cloudjam/examples/challenges/helios-cold-chain/internal/telemetry"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

type handler struct {
	bucket string
	s3     *s3.Client
}

func main() {
	ctx := context.Background()
	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		panic(err)
	}
	lambda.Start((&handler{
		bucket: os.Getenv("RESULT_BUCKET"),
		s3:     s3.NewFromConfig(awsConfig),
	}).handle)
}

func (h *handler) handle(ctx context.Context, request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	if request.RequestContext.HTTP.Method != http.MethodGet || strings.TrimRight(request.RawPath, "/") != "/result" {
		return response(http.StatusNotFound, map[string]string{"error": "GET /result is required"}), nil
	}
	sequence, err := strconv.ParseInt(request.QueryStringParameters["sequence"], 10, 64)
	if err != nil {
		return response(http.StatusBadRequest, map[string]string{"error": "sequence must be an integer"}), nil
	}
	reading := telemetry.Reading{ShipmentID: request.QueryStringParameters["shipment_id"], Sequence: sequence}
	if reading.ShipmentID == "" || reading.Sequence <= 0 {
		return response(http.StatusBadRequest, map[string]string{"error": "shipment_id and sequence are required"}), nil
	}
	output, err := h.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(h.bucket),
		Key:    aws.String(reading.Key()),
	})
	if err != nil {
		var missing *types.NoSuchKey
		var apiError smithy.APIError
		if errors.As(err, &missing) || errors.As(err, &apiError) && apiError.ErrorCode() == "NoSuchKey" {
			return response(http.StatusNotFound, map[string]string{"error": "result not found"}), nil
		}
		return events.LambdaFunctionURLResponse{}, fmt.Errorf("read result: %w", err)
	}
	defer output.Body.Close()
	body, err := io.ReadAll(io.LimitReader(output.Body, 1<<20))
	if err != nil {
		return events.LambdaFunctionURLResponse{}, err
	}
	return events.LambdaFunctionURLResponse{
		StatusCode: http.StatusOK,
		Headers:    map[string]string{"content-type": "application/json"},
		Body:       string(body),
	}, nil
}

func response(status int, body any) events.LambdaFunctionURLResponse {
	encoded, _ := json.Marshal(body)
	return events.LambdaFunctionURLResponse{
		StatusCode: status,
		Headers:    map[string]string{"content-type": "application/json"},
		Body:       string(encoded),
	}
}
