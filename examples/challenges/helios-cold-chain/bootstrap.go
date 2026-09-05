//go:build wasip1

package main

import (
	"fmt"

	"codeberg.org/megakuul/cloudjam/pkg/challenge"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/policy"
)

type iamRole struct {
	AssumeRolePolicyDocument policy.Document `json:"AssumeRolePolicyDocument"`
	Description              *string         `json:"Description,omitempty"`
	Path                     *string         `json:"Path,omitempty"`
	Policies                 []iamRolePolicy `json:"Policies,omitempty"`
	RoleName                 *string         `json:"RoleName,omitempty"`
}

func (*iamRole) CloudJamType() string { return "AWS::IAM::Role" }

type iamRolePolicy struct {
	PolicyDocument policy.Document `json:"PolicyDocument"`
	PolicyName     *string         `json:"PolicyName"`
}

type roleSpec struct {
	name       string
	principals []string
	actions    policy.Actions
}

func bootstrap(*challenge.Scenario) error {
	common := policy.Actions{
		"logs:CreateLogGroup", "logs:CreateLogStream", "logs:PutLogEvents",
		"xray:PutTelemetryRecords", "xray:PutTraceSegments",
	}
	roles := []roleSpec{
		{
			name:       "helios-ingest-role",
			principals: []string{"lambda.amazonaws.com"},
			actions:    append(common, "events:PutEvents"),
		},
		{
			name:       "helios-relay-role",
			principals: []string{"lambda.amazonaws.com"},
			actions:    append(common, "kms:Decrypt", "kms:GenerateDataKey", "sqs:SendMessage"),
		},
		{
			name:       "helios-orchestrator-role",
			principals: []string{"lambda.amazonaws.com"},
			actions: append(common, "kms:Decrypt", "sqs:ChangeMessageVisibility", "sqs:DeleteMessage",
				"sqs:GetQueueAttributes", "sqs:ReceiveMessage", "states:StartSyncExecution"),
		},
		{
			name:       "helios-analyzer-role",
			principals: []string{"lambda.amazonaws.com"},
			actions: append(common, "kms:Decrypt", "kms:Encrypt", "kms:GenerateDataKey", "s3:PutObject",
				"secretsmanager:GetSecretValue", "sns:Publish"),
		},
		{
			name:       "helios-query-role",
			principals: []string{"lambda.amazonaws.com"},
			actions:    append(common, "kms:Decrypt", "s3:GetObject", "s3:ListBucket"),
		},
		{
			name:       "helios-states-role",
			principals: []string{"states.amazonaws.com"},
			actions: policy.Actions{
				"lambda:InvokeFunction", "logs:CreateLogDelivery", "logs:DeleteLogDelivery",
				"logs:DescribeLogGroups", "logs:DescribeResourcePolicies", "logs:GetLogDelivery",
				"logs:ListLogDeliveries", "logs:PutResourcePolicy", "logs:UpdateLogDelivery",
				"xray:GetSamplingRules", "xray:GetSamplingTargets", "xray:PutTelemetryRecords",
				"xray:PutTraceSegments",
			},
		},
	}

	existing, err := resourceIdentifiers[*iamRole]()
	if err != nil {
		return err
	}
	for _, role := range roles {
		if existing[role.name] {
			continue
		}
		if err := createRole(role); err != nil {
			return err
		}
	}
	return nil
}

func resourceIdentifiers[T aws.Resource]() (map[string]bool, error) {
	resources, err := listResources[T]()
	if err != nil {
		return nil, err
	}
	identifiers := make(map[string]bool, len(resources))
	for identifier := range resources {
		identifiers[identifier] = true
	}
	return identifiers, nil
}

func createRole(spec roleSpec) error {
	_, err := aws.Create(&iamRole{
		AssumeRolePolicyDocument: policy.Document{
			Version: policy.Version20121017,
			Statement: []policy.Statement{{
				Effect:    policy.Allow,
				Principal: policy.PrincipalService(spec.principals...),
				Action:    policy.Actions{"sts:AssumeRole"},
			}},
		},
		Description: new("Preconfigured role for the Helios cold-chain challenge"),
		Path:        new("/"),
		Policies: []iamRolePolicy{{
			PolicyDocument: policy.Document{
				Version: policy.Version20121017,
				Statement: []policy.Statement{{
					Effect:   policy.Allow,
					Action:   spec.actions,
					Resource: policy.ARNAll,
				}},
			},
			PolicyName: new(spec.name + "-policy"),
		}},
		RoleName: new(spec.name),
	})
	if err != nil {
		return fmt.Errorf("create %s: %w", spec.name, err)
	}
	return nil
}
