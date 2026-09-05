package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"codeberg.org/megakuul/cloudjam/examples/challenges/helios-cold-chain/internal/telemetry"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type event struct {
	Detail telemetry.Reading `json:"detail"`
}

type handler struct {
	queueURL string
	sqs      *sqs.Client
}

func main() {
	ctx := context.Background()
	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		panic(err)
	}
	lambda.Start((&handler{
		queueURL: os.Getenv("QUEUE_URL"),
		sqs:      sqs.NewFromConfig(awsConfig),
	}).handle)
}

func (h *handler) handle(ctx context.Context, input event) error {
	if err := input.Detail.Validate(); err != nil {
		return err
	}
	body, err := json.Marshal(input.Detail)
	if err != nil {
		return err
	}
	if _, err := h.sqs.SendMessage(ctx, &sqs.SendMessageInput{
		MessageBody: aws.String(string(body)),
		QueueUrl:    aws.String(h.queueURL),
	}); err != nil {
		return fmt.Errorf("enqueue telemetry: %w", err)
	}
	return nil
}
