//go:build wasip1

// Command worldskills-2024-day1 recreates the first day of the WorldSkills
// Lyon 2024 Cloud Computing test project as a cloudjam challenge.
package main

import (
	_ "embed"
	"time"

	"codeberg.org/megakuul/cloudjam/pkg/challenge"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/policy"
)

const (
	ingestEndpointParameter = "/worldskills/day1/ingest-url"
	resultParameterPrefix   = "/worldskills/day1/results"
	batchStatusParameter    = "/worldskills/day1/batch-success"
)

var (
	//go:embed assets/day1-reference.jpg
	referenceDiagram []byte

	//go:embed generated/DataProcessingAPP.zip
	dataProcessingAPP []byte

	//go:embed generated/DataExtractionAPP.zip
	dataExtractionAPP []byte

	//go:embed generated/day1-probe.zip
	day1Probe []byte
)

func main() {
	scenario := challenge.New("WorldSkills Lyon 2024: Day 1", 20*time.Second, bootstrap).
		AddDescription("# Cloud-based batch processing system\n\n"+
			"UnicornTech has identified an anomaly in data returned by devices deployed in customer environments. The devices generate non-standard alphanumeric information codes that the existing system cannot process. Build a cloud-based batch-processing system that securely ingests, processes, categorizes, and stores this data for subsequent analysis and troubleshooting.").
		AddDescription("Effective resource management, AWS cost optimization, and cloud resource organization are essential. Establish secure integration between AWS services, monitor network traffic, optimize performance, and protect data through access controls, encryption, and backups. Use **us-east-1** for the complete architecture.").
		AddDescription("## Initial state\n\n"+
			"The account contains three networks named **Data-Ingestion-VPC**, **Data-Processing-VPC**, and **Data-Extraction-VPC**, together with the preconfigured IAM roles listed below. Do not create additional VPCs, delete these VPCs, or modify their Internet Gateways. The supplied architecture diagram is for reference only.").
		AddDescription("## Stage 1 — Data ingestion\n\n"+
			"Implement a secure REST API that receives abnormal information codes and stores them in DynamoDB. Protect the API against malicious requests. A single Lambda function must handle the API and must be connected to **Data-Ingestion-VPC**.\n\n"+
			"Create a highly available DynamoDB table named **FeedbackTable** with string partition key **id** and string sort key **message**.").
		AddDescription("## Stage 2 — Data preprocessing\n\n"+
			"Establish the preprocessing service in **Data-Processing-VPC**. Download **DataProcessingAPP.zip** from the challenge assets and deploy the supplied x86_64 Linux application to process codes from FeedbackTable. Store processed data in a provisioned PostgreSQL-compatible RDS database; serverless databases are prohibited. Retain backups for more than seven days.\n\n"+
			"Create the required table with `CREATE TABLE IF NOT EXISTS DataProcessingAPP (id SERIAL PRIMARY KEY, message VARCHAR(255));`. The replacement application preserves this table and adds a source identifier used to make repeated batch runs idempotent.").
		AddDescription("## Stage 3 — Data extraction and storage\n\n"+
			"Establish the extraction service in **Data-Extraction-VPC**. Download **DataExtractionAPP.zip** and deploy the supplied application to extract processed codes from RDS and store the final values in Systems Manager Parameter Store. Connect the VPCs without introducing a single point of failure or an unnecessary public path.").
		AddDescription("## Applications and scheduling\n\n"+
			"Both application archives contain the precompiled binary, a Dockerfile, a sample `config.ini`, and deployment notes. The binaries must not be modified. Build their images for x86_64 Amazon Linux 2 compatible containers. Use AWS Batch to execute DataProcessingAPP and EventBridge Scheduler to run it regularly. Run DataExtractionAPP as a managed service.\n\n"+
			"Upload essential files to a versioned competition-account S3 bucket with encryption and cost-effective lifecycle management. Use ECR with immutable tags and a repository policy. A CI/CD pipeline with a review step and CodeDeploy deployment is strongly recommended for the ingestion Lambda.").
		AddDescription("## Service contract\n\n"+
			"The ingestion API accepts **POST /post_data** with `{ \"id\": \"device-01\", \"message\": \"A1B2C3\" }` and returns any 2xx status after persisting both strings in FeedbackTable. DataProcessingAPP copies new records to RDS. DataExtractionAPP writes each completed result to `"+resultParameterPrefix+"/<id>` using the original message as its value.\n\n"+
			"The processing application writes `"+batchStatusParameter+"` after a successful batch run. Keep result and status parameters on the **Intelligent-Tiering** tier.").
		AddDescription("## IAM roles\n\n"+
			"The following roles and instance profile are preconfigured: `game-ec2-profile`, `game-ec2-role`, `game-cloudwatch-event-role`, `game-lambda-role`, `game-ecs-role`, `game-ecs-execution-role`, `game-scheduler-role`, `game-codedeploy-role`, and `game-codepipeline-role`. Select these existing roles when configuring services; do not create replacements.").
		AddDescription("## Live evaluation\n\n"+
			"When the ingestion service is ready, create the String parameter `"+ingestEndpointParameter+"` with its base URL. Do not include `/post_data`. The evaluator sends unique abnormal codes through the complete system and waits for the matching result parameter. Operational points follow the rolling end-to-end success rate: 5 points each at 60%, 70%, 80%, and 90%. **day1-probe.zip** performs the same client-side check using the active AWS credentials.").
		AddDescription("## Scoring\n\n"+
			"Points are awarded for successful end-to-end processing, the expected cloud services, cost-effective configuration, security, reliability, performance efficiency, sustainability, and the overall architecture. Maximum: **270 design + 20 operational**.").
		AddDiagram("Day 1 reference architecture", referenceDiagram).
		AddAsset("DataProcessingAPP.zip", dataProcessingAPP).
		AddAsset("DataExtractionAPP.zip", dataExtractionAPP).
		AddAsset("day1-probe.zip", day1Probe).
		AddClue("where should i begin",
			"Start with FeedbackTable and a POST /post_data Lambda API in Data-Ingestion-VPC. Verify that one request creates an item before adding the batch and extraction stages.", -5).
		AddClue("how do i configure the supplied applications",
			"Both archives document the config.ini keys and equivalent environment variables. DataProcessingAPP needs FeedbackTable and the PostgreSQL URL; DataExtractionAPP needs the PostgreSQL URL and result parameter prefix.", -10).
		AddClue("how should the applications communicate",
			"Peer the processing and extraction VPCs and add both-direction routes. Use private service endpoints where workloads need DynamoDB, ECR, S3, Systems Manager, or CloudWatch Logs.", -10).
		AddClue("how does live marking start",
			"Create /worldskills/day1/ingest-url as a String parameter containing only the API base URL. The final parameters must be written below /worldskills/day1/results by DataExtractionAPP.", -5).
		SetPermission(accessPolicy()).
		SetGuardrail(guardrailPolicy())

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
					"apigateway:*", "backup:*", "batch:*", "cloudwatch:*", "codebuild:*",
					"codedeploy:*", "codepipeline:*", "dynamodb:*", "ec2:*", "ecr:*",
					"ecs:*", "events:*", "iam:CreateServiceLinkedRole", "iam:GetInstanceProfile",
					"iam:GetRole", "iam:ListInstanceProfiles", "iam:ListRoles", "iam:PassRole",
					"kms:*", "lambda:*", "logs:*", "rds:*", "s3:*", "scheduler:*",
					"secretsmanager:*", "ssm:*", "sts:GetCallerIdentity", "wafv2:*", "xray:*",
				},
				Resource: policy.ARNAll,
			},
		},
	}
}

func guardrailPolicy() policy.Document {
	document := accessPolicy()
	document.Statement = append(document.Statement, policy.Statement{
		Effect: policy.Deny,
		Action: policy.Actions{
			"ec2:AttachInternetGateway", "ec2:CreateInternetGateway", "ec2:CreateVpc",
			"ec2:DeleteInternetGateway", "ec2:DeleteVpc", "ec2:DetachInternetGateway",
		},
		Resource: policy.ARNAll,
	})
	return document
}
