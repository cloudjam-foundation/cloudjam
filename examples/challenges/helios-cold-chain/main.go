//go:build wasip1

// Command helios-cold-chain runs an advanced serverless cold-chain challenge.
package main

import (
	_ "embed"
	"time"

	"codeberg.org/megakuul/cloudjam/pkg/challenge"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/policy"
)

const (
	ingestEndpointParameter = "/cloudjam/helios/ingest-url"
	queryEndpointParameter  = "/cloudjam/helios/query-url"
)

var (
	//go:embed assets/helios-reference.jpg
	referenceDiagram []byte

	//go:embed generated/helios-ingest.zip
	ingestApplication []byte

	//go:embed generated/helios-relay.zip
	relayApplication []byte

	//go:embed generated/helios-orchestrator.zip
	orchestratorApplication []byte

	//go:embed generated/helios-analyzer.zip
	analyzerApplication []byte

	//go:embed generated/helios-query.zip
	queryApplication []byte

	//go:embed generated/helios-probe.zip
	heliosProbe []byte
)

func main() {
	scenario := challenge.New("Helios Cold-Chain Relay", 20*time.Second, bootstrap).
		AddDescription("# Helios Cold-Chain Relay\n\n"+
			"Helios Medical Logistics transports temperature-sensitive biologics between hospitals. Its existing telemetry gateway cannot absorb regional traffic bursts and silently loses readings during downstream incidents. Build a secure, event-driven replacement that accepts readings immediately, processes them durably, detects temperature excursions, and makes every result available to clinical operations.").
		AddDescription("The safe transport range is **2–8 °C**. The platform must remain serverless, tolerate retries without losing a reading, and use **us-east-1** throughout. Apply the tag `Challenge=helios-cold-chain` to the principal resources.").
		AddDescription("## Initial state\n\n"+
			"Six IAM roles are preconfigured: `helios-ingest-role`, `helios-relay-role`, `helios-orchestrator-role`, `helios-analyzer-role`, `helios-query-role`, and `helios-states-role`. Use the matching role for each supplied application and the workflow. Competitors cannot create replacement roles.").
		AddDescription("## Stage 1 — Receive telemetry\n\n"+
			"Deploy **helios-ingest.zip** as an arm64 Lambda function using the `provided.al2023` runtime. Set `EVENT_BUS_NAME` and expose the function through a public Lambda Function URL. Create the custom EventBridge bus **helios-cold-chain**. An enabled rule on this bus must match source `helios.coldchain` and detail type `TemperatureReading`, then invoke **helios-relay.zip**. Configure retries and a dead-letter queue for the rule target.").
		AddDescription("## Stage 2 — Buffer and orchestrate\n\n"+
			"Deploy **helios-relay.zip**, set `QUEUE_URL`, and buffer its output in the encrypted standard queue **helios-telemetry**. Configure long polling, a visibility timeout of at least 60 seconds, and the encrypted dead-letter queue **helios-telemetry-dlq** with at least seven days of retention. Use a redrive policy to move repeatedly failing messages.\n\n"+
			"Deploy **helios-orchestrator.zip**, set `STATE_MACHINE_ARN`, and connect it to helios-telemetry with an enabled Lambda event source mapping. Use a batch size between 2 and 10 and enable `ReportBatchItemFailures`.").
		AddDescription("## Stage 3 — Analyze and alert\n\n"+
			"Store `{\"minimum_c\":2,\"maximum_c\":8}` in the encrypted Secrets Manager secret **helios/temperature-limits**. Create an **Express** Step Functions workflow that invokes **helios-analyzer.zip** and returns its result. Enable workflow logging and X-Ray tracing.\n\n"+
			"Set the analyzer variables `RESULT_BUCKET`, `KMS_KEY_ID`, `LIMITS_SECRET`, and `ALERT_TOPIC_ARN`. Store results in a KMS-encrypted, versioned S3 bucket whose name starts with **helios-results-**. Publish excursions to the encrypted SNS topic **helios-excursions**. The supplied binary writes `processed/<shipment_id>/<sequence>.json`.").
		AddDescription("## Stage 4 — Query results\n\n"+
			"Deploy **helios-query.zip**, set `RESULT_BUCKET`, and expose a public Function URL. A successful request returns the analyzer result stored in S3. Keep all five functions on arm64, at or below 512 MB, tagged, and configured with active X-Ray tracing.").
		AddDescription("## Service contract\n\n"+
			"Ingestion accepts **POST /telemetry** with:\n\n"+
			"```json\n{\"shipment_id\":\"BIO-1042\",\"sequence\":1,\"temperature_c\":5.4,\"recorded_at\":\"2026-09-03T12:00:00Z\"}\n```\n\n"+
			"It returns HTTP 202. Query accepts **GET /result?shipment_id=BIO-1042&sequence=1** and returns HTTP 200 after processing. The result contains the original fields, `processed_at`, and status `safe` or `excursion`.").
		AddDescription("## Operations and cost\n\n"+
			"Set finite retention on the Lambda log groups. Add a CloudWatch dashboard and an alarm for pipeline failures. Create a short-retention EventBridge archive for replay. Block public S3 access, enforce bucket-owner ownership, enable lifecycle management, and configure S3 Intelligent-Tiering. Use the AWS managed KMS aliases for SQS, SNS, S3, and Secrets Manager; no customer-managed key is required.").
		AddDescription("## Live evaluation\n\n"+
			"When both endpoints are ready, store their base URLs as String parameters `"+ingestEndpointParameter+"` and `"+queryEndpointParameter+"`. The evaluator continuously submits unique readings and queries their results. Operational points follow the rolling end-to-end success rate: 5 points each at 60%, 70%, 80%, and 90%. **helios-probe.zip** performs the same check from any Linux workstation.").
		AddDescription("## Scoring\n\n"+
			"Points cover operational excellence, security, reliability, performance efficiency, cost optimization, and the complete event-driven architecture. Maximum: **320 design + 20 operational**.").
		AddDiagram("Helios reference architecture", referenceDiagram).
		AddAsset("helios-ingest.zip", ingestApplication).
		AddAsset("helios-relay.zip", relayApplication).
		AddAsset("helios-orchestrator.zip", orchestratorApplication).
		AddAsset("helios-analyzer.zip", analyzerApplication).
		AddAsset("helios-query.zip", queryApplication).
		AddAsset("helios-probe.zip", heliosProbe).
		AddClue("where should i begin",
			"Deploy helios-ingest.zip and create the custom event bus first. Verify that POST /telemetry creates a matching EventBridge event before adding the durable processing path.", -5).
		AddClue("how should eventbridge reach the queue",
			"Use helios-relay.zip as the EventBridge target. Its QUEUE_URL variable points to helios-telemetry; this keeps EventBridge independent of the queue encryption key policy.", -5).
		AddClue("how should the workflow invoke the analyzer",
			"Create an Express state machine with a Lambda Invoke task targeting helios-analyzer. The orchestrator uses StartSyncExecution, so a Standard workflow will not work.", -10).
		AddClue("how do i test the complete system",
			"Create both /cloudjam/helios endpoint parameters, then run helios-probe from the supplied archive. It submits one safe reading and one excursion and waits for both query results.", -5).
		SetPermission(accessPolicy()).
		SetGuardrail(accessPolicy())

	addDesignChecks(scenario)
	addOperationalChecks(scenario)
	scenario.Start()
}

func accessPolicy() policy.Document {
	return policy.Document{
		Version: policy.Version20121017,
		Statement: []policy.Statement{
			{
				Effect: policy.Allow,
				Action: policy.Actions{
					"cloudwatch:*", "events:*", "iam:GetRole", "iam:ListRoles", "iam:PassRole",
					"kms:DescribeKey", "kms:ListAliases", "lambda:*", "logs:*", "s3:*",
					"secretsmanager:*", "sns:*", "sqs:*", "ssm:*", "states:*",
					"sts:GetCallerIdentity", "xray:*",
				},
				Resource: policy.ARNAll,
			},
		},
	}
}
