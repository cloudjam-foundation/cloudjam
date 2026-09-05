package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

type handler struct {
	stateMachine string
	step         *sfn.Client
}

func main() {
	ctx := context.Background()
	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		panic(err)
	}
	lambda.Start((&handler{
		stateMachine: os.Getenv("STATE_MACHINE_ARN"),
		step:         sfn.NewFromConfig(awsConfig),
	}).handle)
}

func (h *handler) handle(ctx context.Context, input events.SQSEvent) (events.SQSEventResponse, error) {
	response := events.SQSEventResponse{}
	for _, record := range input.Records {
		output, err := h.step.StartSyncExecution(ctx, &sfn.StartSyncExecutionInput{
			Input:           aws.String(record.Body),
			StateMachineArn: aws.String(h.stateMachine),
		})
		if err != nil || output.Status != types.SyncExecutionStatusSucceeded {
			response.BatchItemFailures = append(response.BatchItemFailures, events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
			if err == nil {
				err = fmt.Errorf("execution failed: %s", output.Status)
			}
			fmt.Printf("process %s: %v\n", record.MessageId, err)
		}
	}
	return response, nil
}
