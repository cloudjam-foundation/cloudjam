package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

func main() {
	ingestURL := flag.String("ingest-url", "", "ingestion API base URL")
	resultPrefix := flag.String("result-prefix", "/worldskills/day1/results", "Parameter Store result prefix")
	region := flag.String("region", "us-east-1", "AWS region")
	timeout := flag.Duration("timeout", 3*time.Minute, "maximum result wait")
	flag.Parse()
	if *ingestURL == "" {
		flag.Usage()
		os.Exit(2)
	}

	ctx := context.Background()
	awsConfig, err := config.LoadDefaultConfig(ctx, config.WithRegion(*region))
	if err != nil {
		fail(err)
	}
	id := fmt.Sprintf("probe-%d", time.Now().UnixNano())
	message := fmt.Sprintf("A1B2C3-%d", time.Now().UnixNano())
	if err := post(ctx, endpoint(*ingestURL, "/post_data"), id, message); err != nil {
		fail(err)
	}
	if err := waitForResult(ctx, ssm.NewFromConfig(awsConfig), strings.TrimRight(*resultPrefix, "/")+"/"+id, message, *timeout); err != nil {
		fail(err)
	}
	fmt.Println("Day 1 pipeline passed")
}

func post(ctx context.Context, target, id, message string) error {
	body, err := json.Marshal(map[string]string{"id": id, "message": message})
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("data ingestion: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("data ingestion: %s: %s", response.Status, strings.TrimSpace(string(responseBody)))
	}
	return nil
}

func waitForResult(ctx context.Context, client *ssm.Client, name, expected string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		output, err := client.GetParameter(ctx, &ssm.GetParameterInput{Name: aws.String(name)})
		var notFound *types.ParameterNotFound
		if err == nil && output.Parameter != nil && output.Parameter.Value != nil && *output.Parameter.Value == expected {
			return nil
		}
		if err != nil && !errors.As(err, &notFound) {
			return err
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("result %s was not written within %s", name, timeout)
}

func endpoint(base, path string) string {
	return strings.TrimRight(base, "/") + path
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
