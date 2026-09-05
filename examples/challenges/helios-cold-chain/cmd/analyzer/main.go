package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"codeberg.org/megakuul/cloudjam/examples/challenges/helios-cold-chain/internal/telemetry"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type limits struct {
	MinimumC float64 `json:"minimum_c"`
	MaximumC float64 `json:"maximum_c"`
}

type handler struct {
	bucket string
	key    string
	secret string
	topic  string

	s3      *s3.Client
	secrets *secretsmanager.Client
	sns     *sns.Client
}

func main() {
	ctx := context.Background()
	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		panic(err)
	}
	lambda.Start((&handler{
		bucket:  os.Getenv("RESULT_BUCKET"),
		key:     os.Getenv("KMS_KEY_ID"),
		secret:  os.Getenv("LIMITS_SECRET"),
		topic:   os.Getenv("ALERT_TOPIC_ARN"),
		s3:      s3.NewFromConfig(awsConfig),
		secrets: secretsmanager.NewFromConfig(awsConfig),
		sns:     sns.NewFromConfig(awsConfig),
	}).handle)
}

func (h *handler) handle(ctx context.Context, reading telemetry.Reading) (telemetry.Result, error) {
	if err := reading.Validate(); err != nil {
		return telemetry.Result{}, err
	}
	thresholds, err := h.limits(ctx)
	if err != nil {
		return telemetry.Result{}, err
	}
	status := "safe"
	if reading.TemperatureC < thresholds.MinimumC || reading.TemperatureC > thresholds.MaximumC {
		status = "excursion"
	}
	result := telemetry.Result{
		Reading:     reading,
		Status:      status,
		ProcessedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	body, err := json.Marshal(result)
	if err != nil {
		return telemetry.Result{}, err
	}
	if _, err := h.s3.PutObject(ctx, &s3.PutObjectInput{
		Body:                 bytes.NewReader(body),
		Bucket:               aws.String(h.bucket),
		ContentType:          aws.String("application/json"),
		Key:                  aws.String(reading.Key()),
		ServerSideEncryption: "aws:kms",
		SSEKMSKeyId:          aws.String(h.key),
	}); err != nil {
		return telemetry.Result{}, fmt.Errorf("store result: %w", err)
	}
	if status == "excursion" {
		if _, err := h.sns.Publish(ctx, &sns.PublishInput{
			Message:  aws.String(string(body)),
			Subject:  aws.String("Cold-chain temperature excursion"),
			TopicArn: aws.String(h.topic),
		}); err != nil {
			return telemetry.Result{}, fmt.Errorf("publish alert: %w", err)
		}
	}
	return result, nil
}

func (h *handler) limits(ctx context.Context) (limits, error) {
	output, err := h.secrets.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: aws.String(h.secret)})
	if err != nil {
		return limits{}, fmt.Errorf("read limits: %w", err)
	}
	var thresholds limits
	if output.SecretString == nil {
		return limits{}, fmt.Errorf("read limits: secret has no string value")
	}
	if err := json.Unmarshal([]byte(*output.SecretString), &thresholds); err != nil {
		return limits{}, fmt.Errorf("read limits: %w", err)
	}
	if thresholds.MinimumC >= thresholds.MaximumC {
		return limits{}, fmt.Errorf("read limits: minimum_c must be below maximum_c")
	}
	return thresholds, nil
}
