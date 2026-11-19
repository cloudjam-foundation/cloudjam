//go:build wasip1

package main

import (
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/policy"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/apigateway"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/dynamodb"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/lambda"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/sqs"
)

var permissions = policy.Document{
	Version: policy.Version20121017,
	Statement: []policy.Statement{
		{
			Sid:      "ApiGatewayAccess",
			Effect:   policy.Allow,
			Action:   policy.ActionsFrom(apigateway.ActionsList, apigateway.ActionsRead, apigateway.ActionsWrite),
			Resource: policy.ARNsFrom("arn:aws:apigateway:*:*:*"),
		},
		{
			Sid:      "SQSAccess",
			Effect:   policy.Allow,
			Action:   policy.ActionsFrom(sqs.ActionsTagging, sqs.ActionsRead, sqs.ActionsWrite),
			Resource: policy.ARNsFrom("arn:aws:sqs:*:*:*"),
		},
		{
			Sid:      "LambdaAccess",
			Effect:   policy.Allow,
			Action:   policy.ActionsFrom(lambda.ActionsList, lambda.ActionsTagging, lambda.ActionsRead, lambda.ActionsWrite),
			Resource: policy.ARNsFrom("arn:aws:lambda:*:*:*"),
		},
		{
			Sid:      "DynamodbAccess",
			Effect:   policy.Allow,
			Action:   policy.ActionsFrom(dynamodb.ActionsList, dynamodb.ActionsTagging, dynamodb.ActionsRead, dynamodb.ActionsWrite),
			Resource: policy.ARNsFrom("arn:aws:dynamodb:*:*:*"),
		},
	},
}
