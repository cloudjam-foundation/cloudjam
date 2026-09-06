//go:build wasip1

package main

import (
	"embed"
	"encoding/base64"
	"strings"
	"time"

	"codeberg.org/megakuul/cloudjam/pkg/challenge"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/policy"
)

//go:embed README.md assets/architecture.svg game-assets/*
var files embed.FS

func main() {
	readme, _ := files.ReadFile("README.md")
	diagram, _ := files.ReadFile("assets/architecture.svg")
	scenario := challenge.New("GridLine — Day 1", 10*time.Second, bootstrap).
		AddDescription(strings.ReplaceAll(string(readme), "](assets/architecture.svg)", "](data:image/svg+xml;base64,"+base64.StdEncoding.EncodeToString(diagram)+")")).
		AddDescription("## CloudJam live evaluation\n\n" +
			"Submit your CloudFront HTTPS base URL as the String parameter `" + endpointParameter + "` in **eu-central-1**. The evaluator sends the mock authentication headers documented in §9.1; forward them through the edge and set `GRID_DEV_AUTH=1`.\n\n" +
			"The 13 recovered checks run once per minute for six hours, awarding up to **100 operational points per round**. Points accumulate for each successful check. Activity shows each result, including the HTTP method, path, status, latency, and failure details.\n\n" +
			"The supplied ECS binaries require an **AppConfig Agent sidecar** at `http://127.0.0.1:2772`. Prefetch both profiles and start the agent before the application. To meet the README's 401 requirement, configure the ALB to reject protected requests without a Bearer header or both scoring headers; the original binary returns 403 for anonymous heat mutations.\n\n" +
			"All supplied assets and the downloadable README are unchanged. The briefing displays the original architecture SVG.").
		SetPermission(accessPolicy()).
		SetGuardrail(accessPolicy())

	assets, _ := files.ReadDir("game-assets")
	for _, asset := range assets {
		name := "game-assets/" + asset.Name()
		data, _ := files.ReadFile(name)
		scenario.AddAsset(name, data)
	}
	scenario.AddAsset("README.md", readme).
		AddAsset("assets/architecture.svg", diagram)
	scenario.Start()
}

func accessPolicy() policy.Document {
	return policy.Document{
		Version: policy.Version20121017,
		Statement: []policy.Statement{{
			Effect: policy.Allow,
			Action: policy.Actions{
				"acm:*", "apigateway:*", "appconfig:*", "application-autoscaling:*",
				"autoscaling:*", "cloudformation:*", "cloudfront:*", "cloudwatch:*",
				"codebuild:*", "codepipeline:*", "cognito-idp:*", "ec2:*", "ecr:*",
				"ecs:*", "elasticloadbalancing:*", "events:*", "firehose:*", "iam:*",
				"kms:*", "lambda:*", "logs:*", "rds:*", "rds-data:*", "s3:*",
				"secretsmanager:*", "sns:*", "sqs:*", "ssm:*", "states:*",
				"sts:GetCallerIdentity", "tag:*", "xray:*",
			},
			Resource: policy.ARNAll,
		}},
	}
}
