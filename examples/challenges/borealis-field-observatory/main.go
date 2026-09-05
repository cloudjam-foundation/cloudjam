//go:build wasip1

// Command borealis-field-observatory runs an advanced environmental telemetry challenge.
package main

import (
	_ "embed"
	"time"

	"codeberg.org/megakuul/cloudjam/pkg/challenge"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/policy"
)

const endpointParameter = "/cloudjam/borealis/endpoint-url"

var (
	//go:embed assets/borealis-reference.jpg
	referenceDiagram []byte

	//go:embed generated/borealis-api.zip
	apiBuildContext []byte

	//go:embed generated/borealis-probe.zip
	borealisProbe []byte
)

func main() {
	scenario := challenge.New("Borealis Field Observatory", 20*time.Second, bootstrap).
		AddDescription("# Borealis Field Observatory\n\n"+
			"Borealis Research operates sealed plant habitats in remote polar stations. Each station sends humidity and carbon-dioxide observations to a public API. Build a container-based platform that classifies observations from centrally deployed limits, transports them through the managed IoT data plane, retains an immutable analytical history, and lets researchers query an observation by identifier.").
		AddDescription("Use **us-east-1** throughout. Apply the tag `Challenge=borealis-field-observatory` to the principal resources. Do not introduce Lambda, SQS, EventBridge, or Step Functions into the application path.").
		AddDescription("## Initial state\n\n"+
			"Three IAM roles are preconfigured: `borealis-apprunner-access-role`, `borealis-apprunner-instance-role`, and `borealis-firehose-role`. The IoT rule uses `borealis-iot-rule-role`. Competitors cannot create replacement roles.").
		AddDescription("## Stage 1 — Build and run the field API\n\n"+
			"The supplied **borealis-api.zip** is a complete Docker build context. Build it, publish the image as `borealis-api:competition` in a private ECR repository, and deploy it with AWS App Runner. Keep automatic image deployments disabled, use the two matching App Runner roles, expose port 8080 publicly, and configure an HTTP health check on `/health`. Use no more than 1 vCPU and 2 GB memory.").
		AddDescription("## Stage 2 — Classify observations\n\n"+
			"Create an AWS AppConfig application named **borealis-observatory**, the **production** environment, and the hosted **habitat-limits** configuration profile. Add a JSON Schema validator and deploy this configuration:\n\n"+
			"```json\n{\"minimum_humidity_pct\":40,\"maximum_humidity_pct\":70,\"maximum_co2_ppm\":1200}\n```\n\n"+
			"Use a reusable deployment strategy with a final bake period. Configure App Runner with `APPCONFIG_APPLICATION`, `APPCONFIG_ENVIRONMENT`, and `APPCONFIG_PROFILE`; identifiers or names are accepted.").
		AddDescription("## Stage 3 — Deliver the data lake\n\n"+
			"Set `IOT_ENDPOINT` to the account's `iot:Data-ATS` endpoint and `IOT_TOPIC` to **borealis/observations**. Create an enabled IoT Core topic rule using SQL version `2016-03-23` that selects this topic and writes through `borealis-iot-rule-role` to the direct-put Firehose stream **borealis-observations**. Append a newline to every record and send rule errors to a finite-retention CloudWatch log group.\n\n"+
			"Use Firehose data format conversion with the OpenX JSON deserializer, the Glue table schema, and Parquet output. Deliver the converted records to an S3 bucket whose name starts with **borealis-lake-** under `observations/`; use a distinct error prefix, S3 encryption, delivery logging, and a buffer interval of at most 60 seconds. Protect, encrypt, version, and lifecycle the bucket.").
		AddDescription("## Stage 4 — Query the analytical history\n\n"+
			"Create the Glue database **borealis_observatory** and external table **observations** over the delivered Parquet data. It needs the columns `observation_id` string, `habitat` string, `humidity_pct` double, `co2_ppm` int, `recorded_at` string, and `condition` string. Use the Parquet Hive SerDe.\n\n"+
			"Create the enabled Athena workgroup **borealis-analysts**. Enforce its encrypted S3 result location, publish query metrics, and cap bytes scanned per query. Set `ATHENA_DATABASE`, `ATHENA_TABLE`, and `ATHENA_WORKGROUP` on App Runner.").
		AddDescription("## Service contract\n\n"+
			"`POST /observations` accepts:\n\n"+
			"```json\n{\"observation_id\":\"OBS-1042\",\"habitat\":\"fern-house\",\"humidity_pct\":54.2,\"co2_ppm\":810,\"recorded_at\":\"2026-09-04T12:00:00Z\"}\n```\n\n"+
			"It returns HTTP 202 with condition `nominal` or `attention`. `GET /observations/{observation_id}` queries Athena and returns the stored observation when Firehose has delivered it.").
		AddDescription("## Operations and cost\n\n"+
			"Enable ECR scan-on-push, immutable tags, repository lifecycle management, Firehose delivery logging, and finite retention for the IoT error log. Add a Borealis CloudWatch dashboard and an enabled alarm. Keep S3 private with bucket-owner ownership.").
		AddDescription("## Live evaluation\n\n"+
			"When App Runner is ready, store its base URL, including `https://`, as the String parameter `"+endpointParameter+"`. The evaluator continuously sends an out-of-range observation and waits for the same classified record through Athena. Operational points follow the rolling end-to-end success rate: 5 points each at 60%, 70%, 80%, and 90%. **borealis-probe.zip** performs the same test from Linux.").
		AddDescription("## Scoring\n\n"+
			"Points cover security, reliability, operational excellence, performance efficiency, cost optimization, and the complete container-to-lake architecture. Maximum: **320 design + 20 operational**.").
		AddDiagram("Borealis reference architecture", referenceDiagram).
		AddAsset("borealis-api.zip", apiBuildContext).
		AddAsset("borealis-probe.zip", borealisProbe).
		AddClue("where should i begin",
			"Extract borealis-api.zip, create the ECR repository, then build and push its Docker image. The README lists every environment variable the container requires.", -5).
		AddClue("how does the api reach firehose",
			"The API publishes to borealis/observations through the IoT data endpoint. Configure an IoT topic rule whose Firehose action appends a newline and uses borealis-iot-rule-role.", -5).
		AddClue("why does athena return no rows",
			"Confirm that Firehose objects exist below observations/, the Glue table Location points to that exact S3 prefix, and the delivery stream converts newline-delimited JSON to Parquet.", -10).
		AddClue("how do i test the complete system",
			"Run borealis-probe with -endpoint set to the App Runner URL. The first Firehose delivery can take up to the configured buffer interval.", -5).
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
					"appconfig:*", "apprunner:*", "athena:*", "cloudwatch:*", "ecr:*",
					"firehose:*", "glue:*", "iam:GetRole", "iam:ListRoles", "iam:PassRole",
					"iot:*", "kms:DescribeKey", "kms:ListAliases", "logs:*", "s3:*", "ssm:*",
					"sts:GetCallerIdentity",
				},
				Resource: policy.ARNAll,
			},
		},
	}
}
