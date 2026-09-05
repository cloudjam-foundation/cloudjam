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
	roles := []roleSpec{
		{
			name:       "borealis-apprunner-access-role",
			principals: []string{"build.apprunner.amazonaws.com"},
			actions: policy.Actions{
				"ecr:GetAuthorizationToken", "ecr:BatchCheckLayerAvailability", "ecr:BatchGetImage", "ecr:GetDownloadUrlForLayer",
			},
		},
		{
			name:       "borealis-apprunner-instance-role",
			principals: []string{"tasks.apprunner.amazonaws.com"},
			actions: policy.Actions{
				"appconfig:GetLatestConfiguration", "appconfig:StartConfigurationSession",
				"athena:GetQueryExecution", "athena:GetQueryResults", "athena:StartQueryExecution",
				"glue:GetDatabase", "glue:GetDatabases", "glue:GetPartition", "glue:GetPartitions", "glue:GetTable", "glue:GetTables",
				"iot:Publish", "s3:GetBucketLocation", "s3:GetObject", "s3:ListBucket", "s3:PutObject",
			},
		},
		{
			name:       "borealis-firehose-role",
			principals: []string{"firehose.amazonaws.com"},
			actions: policy.Actions{
				"glue:GetTable", "glue:GetTableVersion", "glue:GetTableVersions",
				"logs:PutLogEvents", "s3:AbortMultipartUpload", "s3:GetBucketLocation", "s3:GetObject", "s3:ListBucket", "s3:ListBucketMultipartUploads", "s3:PutObject",
			},
		},
		{
			name:       "borealis-iot-rule-role",
			principals: []string{"iot.amazonaws.com"},
			actions:    policy.Actions{"firehose:PutRecord", "firehose:PutRecordBatch", "logs:CreateLogStream", "logs:PutLogEvents"},
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
		Description: new("Preconfigured role for the Borealis field observatory challenge"),
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
