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

//go:embed README.md MARKING.md CLOUDJAM.md assets/architecture.svg game-assets/*
var files embed.FS

func main() {
	readme, _ := files.ReadFile("README.md")
	diagram, _ := files.ReadFile("assets/architecture.svg")
	scenario := challenge.New("VitalForge — Day 2", 10*time.Second, bootstrap).
		AddDescription(strings.ReplaceAll(string(readme), "](assets/architecture.svg)", "](data:image/svg+xml;base64,"+base64.StdEncoding.EncodeToString(diagram)+")")).
		AddDescription("## CloudJam live evaluation\n\n" +
			"Submit your CloudFront HTTPS base URL as the String parameter `" + endpointParameter + "` in the game's AWS Region. Forward the original `X-Dev-Sub` and `X-Dev-Groups` scoring headers through CloudFront and API Gateway, as documented in §9.1.\n\n" +
			"The **12 operational checks** start **50 minutes after the challenge starts**, then run every minute for the remaining **310 minutes** of this six-hour module. Each round awards up to **100 points**. Activity records each attempt's score, HTTP method, path, status, latency and assertion details. Successful attempts accumulate in the overall score.\n\n" +
			"The original **48 marking criteria** retain their relative weights: **124,000 design points** and **31,000 operational points** give an **80% / 20%** split. Design marks reflect the latest assessment and can be lost after regressions. The assessment Lambda and its role are provided by CloudJam and excluded from the solution's footprint.\n\n" +
			"The supplied EKS processes load thresholds at startup. Use an SSM-to-file refresher and a process supervisor to apply changes without redeploying pods; keep the original binaries unchanged. See CLOUDJAM.md.\n\n" +
			"The original README, architecture SVG and all ten application assets are available unchanged. See MARKING.md for assessment evidence and CLOUDJAM.md for operating instructions.").
		SetPermission(accessPolicy()).SetGuardrail(guardrailPolicy())
	assets, _ := files.ReadDir("game-assets")
	for _, asset := range assets {
		name := "game-assets/" + asset.Name()
		data, _ := files.ReadFile(name)
		scenario.AddAsset(name, data)
	}
	marking, _ := files.ReadFile("MARKING.md")
	operation, _ := files.ReadFile("CLOUDJAM.md")
	scenario.AddAsset("README.md", readme).AddAsset("assets/architecture.svg", diagram).
		AddAsset("CLOUDJAM.md", operation).AddAsset("MARKING.md", marking).AddAsset("design-marking.json", designMarking)
	scenario.Start()
}

func guardrailPolicy() policy.Document {
	document := accessPolicy()
	document.Statement = append(document.Statement, policy.Statement{
		Effect: policy.Deny, Action: policy.Actions{"iam:*", "lambda:*"},
		Resource: policy.ARNsFrom("arn:aws:iam::*:role/cloudjam-vitalforge-assessor-*", "arn:aws:lambda:*:*:function:cloudjam-vitalforge-assessor-*"),
	})
	return document
}

func accessPolicy() policy.Document {
	return policy.Document{Version: policy.Version20121017, Statement: []policy.Statement{{
		Effect: policy.Allow, Resource: policy.ARNAll, Action: policy.Actions{
			"acm:*", "apigateway:*", "application-autoscaling:*", "autoscaling:*",
			"cloudformation:*", "cloudfront:*", "cloudtrail:*", "cloudwatch:*",
			"codebuild:*", "codepipeline:*", "cognito-idp:*", "dynamodb:*", "ec2:*",
			"ecr:*", "eks:*", "eks-auth:*", "elasticfilesystem:*", "elasticloadbalancing:*",
			"events:*", "iam:*", "kms:*", "lambda:*", "logs:*", "route53:*", "s3:*",
			"scheduler:*", "secretsmanager:*", "sns:*", "sqs:*", "ssm:*", "sts:GetCallerIdentity",
			"tag:*", "vpc-lattice:*", "wafv2:*", "xray:*",
		},
	}}}
}
