//go:build wasip1

// Command worldskills-2024-day2 recreates the second day of the WorldSkills
// Lyon 2024 Cloud Computing test project as a cloudjam challenge.
package main

import (
	_ "embed"
	"time"

	"codeberg.org/megakuul/cloudjam/pkg/challenge"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/policy"
)

const (
	bloodPressureIngestParameter = "/worldskills/day2/blood-pressure-ingest-url"
	bloodPressureQueryParameter  = "/worldskills/day2/blood-pressure-query-url"
	enterpriseAPIParameter       = "/worldskills/day2/enterprise-api-url"
)

var (
	//go:embed assets/day2-reference.jpg
	referenceDiagram []byte

	//go:embed generated/stub1.zip
	stub1 []byte

	//go:embed generated/day2-probe.zip
	day2Probe []byte
)

func main() {
	scenario := challenge.New("WorldSkills Lyon 2024: Day 2", 20*time.Second, bootstrap).
		AddDescription("# Cloud-based intelligent analytics platform\n\n"+
			"In today's rapidly evolving digital age, data has become one of the most valuable assets for businesses. This is especially true in healthcare, where telemedicine and personal health monitoring make real-time collection and analysis particularly important. "+
			"Your company is a leading artificial-intelligence and big-data analytics company. Its objective is to develop a centralized, secure cloud platform that processes and analyses two types of data sent by POST requests: blood-pressure monitoring data and enterprise operation data.").
		AddDescription("As the solutions architect, you are responsible for controlling resource access and AWS cost, establishing secure and reliable connections while minimizing public internet exposure, monitoring the network, and protecting sensitive data with encryption, access controls, backups, and records of suspicious activity. "+
			"Use **us-east-1** for the complete architecture.").
		AddDescription("## Initial state\n\n"+
			"Design a scalable system capable of handling high volumes of real-time blood-pressure and enterprise operation data. Successful ingestion and correct delivery initiate operational point accumulation. "+
			"The supplied diagram is a simplified architecture for reference only.").
		AddDescription("## Stage 1 — Data reception\n\n"+
			"Build a reception system that handles both POST data streams simultaneously.\n\n"+
			"For **blood-pressure monitoring data**, use Amazon Kinesis for real-time ingestion and processing, Kinesis Data Analytics to identify anomalies, DynamoDB for immediate access to anomaly results, and provisioned Amazon Redshift for historical analysis. Do not use Redshift Serverless.\n\n"+
			"For **enterprise operation data**, use a provisioned Amazon RDS for Aurora database for reliable and efficient access. Serverless databases are not permitted, and the database must retain backups for more than seven days.").
		AddDescription("## Stage 2 — Data query and service development\n\n"+
			"Download **stub1.zip** from the challenge assets and use the supplied x86_64 Linux binary to establish the blood-pressure web service. The service retrieves real-time results from DynamoDB and returns them as JSON with low latency. Capture detailed request logs, including timestamps, client addresses, latency, request paths, and responses.\n\n"+
			"Upload the supplied binary to a versioned competition-account S3 bucket and apply cost-effective lifecycle management. Develop a secure and reliable API that lets users query specific enterprise operation data stored in Aurora.").
		AddDescription("## Stage 3 — Architecture optimization\n\n"+
			"Align the architecture with the six pillars of the AWS Well-Architected Framework: operational excellence, security, reliability, performance efficiency, cost optimization, and sustainability. Implement monitoring, alerting, and evaluation processes, then improve the architecture where those measurements identify weaknesses.\n\n"+
			"Amazon ECS is strongly recommended for the web service. Build for flexibility, scalability, high availability, fault tolerance, and visibility into container configuration and utilization. Automate ECS application deployment with CodePipeline, include a review step, and use CodeDeploy for reliable deployments.").
		AddDescription("## Technical constraints\n\n"+
			"The supplied server application is pre-compiled and must not be modified. **Amazon EKS is prohibited.** Use Systems Manager instead of SSH for instance management. The application is built for x86 architecture and Amazon Linux 2. Monitor cost and avoid unnecessary infrastructure expansion.").
		AddDescription("## Service contracts\n\n"+
			"Blood-pressure data is sent to **POST /post_data** and contains `no`, `systolic`, and `diastolic`. Store real-time analysis results in a highly available DynamoDB table named **cloudraiser-iot** with string partition key **No** and string attributes **Systolic**, **Diastolic**, and **Score**. Delete data older than 48 hours. The supplied service answers **GET /get_value?no=...**.\n\n"+
			"Enterprise updates are sent to **POST /post_data** as `{ \"id\": \"01\", \"column\": \"val1\", \"value\": \"456\" }`. Store string columns `id` and `val1` through `val10`. The query API answers **GET /get_value?id=...** with the complete row as JSON.").
		AddDescription("## IAM roles in the CloudJam account\n\n"+
			"Every role you create must use the customer-managed policy **cloudjam-boundary** as its permissions boundary. Create application roles directly in IAM with the default `/` path and the required service trust policy, then select those existing roles in Lambda, ECS, CodeBuild, CodePipeline, and CodeDeploy. Do not let a service create a role automatically: unbounded roles and roles below the `/service-role/` path are blocked by the competition account guard. "+
			"Set the boundary during role creation to `arn:aws:iam::<account-id>:policy/cloudjam-boundary`, where `<account-id>` is returned by `aws sts get-caller-identity`. The resulting role ARN must look like `arn:aws:iam::<account-id>:role/<name>`, not `arn:aws:iam::<account-id>:role/service-role/<name>`.").
		AddDescription("## Live evaluation\n\n"+
			"Data begins arriving from the start of the game. When the services are ready, submit their base URLs as String parameters in Systems Manager Parameter Store:\n\n"+
			"| Endpoint | Parameter |\n| --- | --- |\n"+
			"| Blood-pressure ingestion | `"+bloodPressureIngestParameter+"` |\n"+
			"| Blood-pressure query | `"+bloodPressureQueryParameter+"` |\n"+
			"| Enterprise ingestion and query | `"+enterpriseAPIParameter+"` |\n\n"+
			"The evaluator starts only when all three parameters exist. It sends unique records through both pipelines and queries the resulting data. Operational points follow the rolling success rate: 5 points each at 60%, 70%, 80%, and 90%. **day2-probe.zip** provides the same client-side smoke test.").
		AddDescription("## Scoring\n\n"+
			"Points are awarded for receiving both data streams, returning correct results with responsive and available services, creating resources that meet the architecture requirements, and aligning the system with the six Well-Architected pillars. Higher totals are earned by correctly configuring more elements. "+
			"Maximum: **350 design + 20 operational**.").
		AddDiagram("Simplified reference architecture", referenceDiagram).
		AddAsset("stub1.zip", stub1).
		AddAsset("day2-probe.zip", day2Probe).
		AddClue("where should i begin",
			"Build the two ingestion paths first. Use Kinesis for blood pressure and API Gateway plus Lambda for enterprise updates. Keep the fixed DynamoDB table and API paths exactly as specified.", -5).
		AddClue("how do i deploy stub1",
			"Extract stub1.zip, build its Dockerfile, push it to ECR, and run at least two Fargate tasks behind an ALB. Give the task role dynamodb:GetItem on cloudraiser-iot and set TABLE_NAME if you changed the default.", -10).
		AddClue("how does live marking find my APIs",
			"Create the three SSM String parameters shown in Live evaluation. Values are base URLs without /post_data or /get_value; the supplied day2-probe binary accepts the same three values.", -5).
		AddClue("what is the hardening path",
			"Work down the Well-Architected pillars: encryption and deletion protection, private service endpoints, logs and alarms, ECS autoscaling and Container Insights, artifact lifecycle, then GuardDuty export.", -15).
		SetPermission(accessPolicy()).
		SetGuardrail(accessPolicy())

	addDesignChecks(scenario)
	addOperationalChecks(scenario)
	scenario.Start()
}

func bootstrap(*challenge.Scenario) error {
	return nil
}

func accessPolicy() policy.Document {
	return policy.Document{
		Version: policy.Version20121017,
		Statement: []policy.Statement{
			{
				Effect: policy.Allow,
				Action: policy.Actions{
					"apigateway:*", "application-autoscaling:*", "cloudfront:*", "cloudwatch:*",
					"codebuild:*", "codedeploy:*", "codepipeline:*", "dynamodb:*", "ec2:*",
					"ecr:*", "ecs:*", "elasticloadbalancing:*", "guardduty:*", "iam:*",
					"kinesis:*", "kinesisanalytics:*", "kms:*", "lambda:*", "logs:*", "rds:*",
					"redshift:*", "s3:*", "ssm:*", "sts:GetCallerIdentity",
				},
				Resource: policy.ARNAll,
			},
		},
	}
}
