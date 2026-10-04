package aws

import (
	"encoding/json"
	"fmt"
)

type policyDocument struct {
	Version   string            `json:"Version"`
	Statement []policyStatement `json:"Statement"`
}

type policyStatement struct {
	Sid         string                    `json:"Sid"`
	Effect      string                    `json:"Effect"`
	Action      []string                  `json:"Action,omitempty"`
	NotAction   []string                  `json:"NotAction,omitempty"`
	Principal   map[string]string         `json:"Principal,omitempty"`
	Resource    []string                  `json:"Resource,omitempty"`
	NotResource []string                  `json:"NotResource,omitempty"`
	Condition   map[string]map[string]any `json:"Condition,omitempty"`
}

// guardPolicy constructs the primary security SCP for sandbox accounts.
// - protects the accounts root and sandbox role
// - blocks possible account escape routes
// - requires the boundaryPolicy to be attached to every new entity created.
//
// IMPORTANT: We give a lot of effort to keep this system clean and avoid privilege escalation inside the account,
// however, we are dealing with highly complex security by configuration here; keep in mind that it will 100% have at least one small
// bug or escape hatch. The retarded AWS system unfortunately does not allow a serious security concept here.
// It MUST be clearly mentioned in the ToS and other papers that escaping the account guards is NOT allowed
// and not part of the challenge (even if technically possible).
func guardControlPolicy(adminRole, sandboxRole, boundaryPolicy string) ([]byte, error) {
	adminRoleARN := fmt.Sprintf("arn:aws:iam::*:role/%s", adminRole)
	sandboxRoleARN := fmt.Sprintf("arn:aws:iam::*:role/%s", sandboxRole)
	boundaryPolicyARN := fmt.Sprintf("arn:aws:iam::*:policy/%s", boundaryPolicy)
	return json.Marshal(policyDocument{
		Version: "2012-10-17",
		Statement: []policyStatement{
			{
				Sid:    "FreezeUngovernedPrincipals",
				Effect: "Deny",
				Action: []string{
					"iam:UpdateAssumeRolePolicy",
					"iam:AttachRolePolicy",
					"iam:PutRolePolicy",
					"iam:DetachRolePolicy",
					"iam:DeleteRolePolicy",
					"iam:PutRolePermissionsBoundary",
					"iam:DeleteRolePermissionsBoundary",
					"iam:UpdateRole",
					"iam:TagRole",
					"iam:UntagRole",
					"iam:PassRole",
				},
				Resource: []string{
					"arn:aws:iam::*:role/aws-service-role/*",
					"arn:aws:iam::*:role/aws-reserved/*",
					"arn:aws:iam::*:role/service-role/*",
					"arn:aws:iam::*:role/OrganizationAccountAccessRole",
					"arn:aws:iam::*:role/AWSControlTowerExecution",
				},
				Condition: map[string]map[string]any{
					"ArnNotLike": {"aws:PrincipalARN": adminRoleARN},
				},
			},
			{
				Sid:    "DenyPoolEscape",
				Effect: "Deny",
				Action: []string{
					"account:CloseAccount",
					"iam:CreateAccountAlias",
					"iam:DeleteAccountAlias",
					"organizations:LeaveOrganization",
				},
				Resource: []string{"*"},
				Condition: map[string]map[string]any{
					"ArnNotLike": {"aws:PrincipalARN": adminRoleARN},
				},
			},
			{
				Sid:      "DenyGuardTamper",
				Effect:   "Deny",
				Action:   []string{"*"},
				Resource: []string{adminRoleARN, sandboxRoleARN},
				Condition: map[string]map[string]any{
					"ArnNotLike": {"aws:PrincipalARN": adminRoleARN},
				},
			},
			{
				Sid:    "DenyIdentityWithoutBoundary",
				Effect: "Deny",
				Action: []string{
					"iam:CreateUser",
					"iam:CreateRole",
					"iam:PutUserPolicy",
					"iam:PutRolePolicy",
					"iam:AttachUserPolicy",
					"iam:AttachRolePolicy",
				},
				Resource: []string{"*"},
				Condition: map[string]map[string]any{
					"ArnNotLike": {"aws:PrincipalARN": adminRoleARN, "iam:PermissionsBoundary": boundaryPolicyARN},
				},
			},
			{
				Sid:    "DenyBoundaryDetach",
				Effect: "Deny",
				Action: []string{
					"iam:DeleteUserPermissionsBoundary",
					"iam:DeleteRolePermissionsBoundary",
				},
				Resource: []string{"*"},
				Condition: map[string]map[string]any{
					"ArnNotLike": {"aws:PrincipalARN": adminRoleARN},
				},
			},
			{
				Sid:    "DenyBoundarySwap",
				Effect: "Deny",
				Action: []string{
					"iam:PutUserPermissionsBoundary",
					"iam:PutRolePermissionsBoundary",
				},
				Resource: []string{"*"},
				Condition: map[string]map[string]any{
					"ArnNotLike": {"aws:PrincipalARN": adminRoleARN, "iam:PermissionsBoundary": boundaryPolicyARN},
				},
			},
			{
				Sid:    "DenyBoundaryPolicyTamper",
				Effect: "Deny",
				Action: []string{
					"iam:CreatePolicyVersion",
					"iam:DeletePolicy",
					"iam:DeletePolicyVersion",
					"iam:SetDefaultPolicyVersion",
				},
				Resource: []string{boundaryPolicyARN},
				Condition: map[string]map[string]any{
					"ArnNotLike": {"aws:PrincipalARN": adminRoleARN},
				},
			},
		},
	})
}

// whitelistControlPolicy generates an SCP that blocks every service and region not specified in this allowlist.
// The idea is to ensure users never use expensive or new bleeding edge AI bullshit that was not blocked by challenge guardrails.
func whitelistControlPolicy(regions []string) ([]byte, error) {
	s3Locations := append([]string(nil), regions...)
	s3LocationCondition := map[string]map[string]any{}
	for _, region := range regions {
		if region == "" {
			return nil, fmt.Errorf("costControlPolicy: region must not be empty")
		}
		if region == "eu-west-1" {
			s3Locations = append(s3Locations, "EU")
		}
		if region == "us-east-1" {
			s3LocationCondition["Null"] = map[string]any{
				"s3:LocationConstraint": "false",
			}
		}
	}
	s3LocationCondition["StringNotEquals"] = map[string]any{
		"s3:LocationConstraint": s3Locations,
	}
	return json.Marshal(policyDocument{
		Version: "2012-10-17",
		Statement: []policyStatement{
			{
				Sid:    "DenyServicesNotExplicitlyAllowed",
				Effect: "Deny",
				NotAction: []string{
					"ec2:*", "ec2-instance-connect:*", "ecr:*", "ecs:*",
					"lambda:*", "cloudformation:*", "elasticloadbalancing:*",
					"application-autoscaling:*",

					"s3:*", "backup:*", "dynamodb:*", "rds:*", "rds-db:*",
					"kinesis:*", "firehose:*", "athena:*", "glue:*", "lakeformation:*",

					"apigateway:*", "execute-api:*", "events:*", "pipes:*",
					"scheduler:*", "states:*", "sns:*", "sqs:*",
					"cognito-identity:*", "cognito-idp:*",

					"route53:*", "servicediscovery:*",
					"cloudfront:*", "cloudfront-keyvaluestore:*",

					"codeartifact:*", "codebuild:*", "codecommit:*",
					"codeconnections:*", "codestar-connections:*",
					"codedeploy:*", "codepipeline:*",

					"cloudwatch:*", "logs:*", "cloudtrail:*", "xray:*",
					"iam:*", "sts:*", "acm:*", "kms:*", "secretsmanager:*",
					"ssm:*", "ssmmessages:*", "ec2messages:*",
					"resource-groups:*", "resource-explorer-2:*", "tag:*",
				},
				Resource: []string{"*"},
			},
			{
				Sid:    "DenyOutsideRegions",
				Effect: "Deny",
				NotAction: []string{
					"iam:*",
					"sts:*",
					"route53:*",
					"cloudfront:*",
					"cloudfront-keyvaluestore:*",
					"waf:*",
					"wafv2:*",

					"s3:ListAllMyBuckets",
					"s3:GetAccountPublicAccessBlock",
					"s3:PutAccountPublicAccessBlock",
				},
				Resource: []string{"*"},
				Condition: map[string]map[string]any{
					"StringNotEquals": {"aws:RequestedRegion": regions},
				},
			},
			{
				Sid:       "DenyS3BucketLocationsOutsideRegions",
				Effect:    "Deny",
				Action:    []string{"s3:CreateBucket"},
				Resource:  []string{"*"},
				Condition: s3LocationCondition,
			},
			{
				Sid:    "DenyCrossRegionReplicationSetup",
				Effect: "Deny",
				Action: []string{
					"s3:PutReplicationConfiguration", "s3:CreateMultiRegionAccessPoint",
					"ecr:PutReplicationConfiguration",
					"dynamodb:CreateGlobalTable", "dynamodb:UpdateGlobalTable",
					"rds:CreateGlobalCluster", "rds:ModifyGlobalCluster",
					"rds:StartDBInstanceAutomatedBackupsReplication",
					"kms:ReplicateKey", "secretsmanager:ReplicateSecretToRegions",
					"lambda:EnableReplication*",
				},
				Resource: []string{"*"},
			},
		},
	})
}

// costControlPolicy generates an SCP that blocks particularly expensive and well known services.
// The idea is to provide some heuristics to catch common cost traps.
func costControlPolicy(regions []string) ([]byte, error) {
	return json.Marshal(policyDocument{
		Version: "2012-10-17",
		Statement: []policyStatement{
			{
				Sid:      "DenyExpensiveInstanceTypes",
				Effect:   "Deny",
				Action:   []string{"ec2:RunInstances", "ec2:StartInstances"},
				Resource: []string{"arn:aws:ec2:*:*:instance/*"},
				Condition: map[string]map[string]any{
					"StringNotLike": {
						"ec2:InstanceType": []string{"t2.*", "t3.*", "t3a.*", "t4g.*", "*.micro", "*.small", "*.medium"},
					},
				},
			},
			{
				Sid:      "DenyDedicatedInstanceTenancy",
				Effect:   "Deny",
				Action:   []string{"ec2:RunInstances"},
				Resource: []string{"arn:aws:ec2:*:*:instance/*"},
				Condition: map[string]map[string]any{
					"StringNotEquals": {
						"ec2:Tenancy": "default",
					},
				},
			},
			{
				Sid:      "DenyLargeVolumes",
				Effect:   "Deny",
				Action:   []string{"ec2:CreateVolume", "ec2:RunInstances"},
				Resource: []string{"arn:aws:ec2:*:*:volume/*"},
				Condition: map[string]map[string]any{
					"NumericGreaterThan": {
						"ec2:VolumeSize": "50",
					},
				},
			},
			{
				Sid:      "DenyExtremeVolumeIOPS",
				Effect:   "Deny",
				Action:   []string{"ec2:CreateVolume", "ec2:RunInstances", "ec2:ModifyVolume"},
				Resource: []string{"arn:aws:ec2:*:*:volume/*"},
				Condition: map[string]map[string]any{
					"NumericGreaterThan": {
						"ec2:VolumeIops": "6000",
					},
				},
			},
			{
				Sid:      "DenyExtremeVolumeThroughput",
				Effect:   "Deny",
				Action:   []string{"ec2:CreateVolume", "ec2:RunInstances", "ec2:ModifyVolume"},
				Resource: []string{"arn:aws:ec2:*:*:volume/*"},
				Condition: map[string]map[string]any{
					"NumericGreaterThan": {
						"ec2:VolumeThroughput": "250",
					},
				},
			},
			{
				Sid:    "DenyExpensiveRDSInstanceClasses",
				Effect: "Deny",
				Action: []string{
					"rds:CreateDBInstance",
					"rds:CreateDBInstanceReadReplica",
					"rds:ModifyDBInstance",
					"rds:RestoreDBInstanceFromDBSnapshot",
					"rds:RestoreDBInstanceFromS3",
					"rds:RestoreDBInstanceToPointInTime",
				},
				Resource: []string{"*"},
				Condition: map[string]map[string]any{
					"StringNotEquals": {
						"rds:DatabaseClass": []string{
							"db.t3.micro",
							"db.t3.small",
							"db.t3.medium",

							"db.t4g.micro",
							"db.t4g.small",
							"db.t4g.medium",
						},
					},
				},
			},
			{
				Sid:    "DenyCommitments",
				Effect: "Deny",
				Action: []string{
					// AWS Marketplace purchases/subscriptions.
					"aws-marketplace:AcceptAgreementApprovalRequest",
					"aws-marketplace:Subscribe",

					// Savings Plans.
					"savingsplans:CreateSavingsPlan",

					// CloudFront flat-rate / paid plans.
					"pricingplanmanager:CreateSubscription",
					"pricingplanmanager:ApprovePaidSubscription",
					"pricingplanmanager:UpdateSubscription",

					// EC2 commitments / dedicated hardware.
					"ec2:AllocateHosts",
					"ec2:CreateCapacityReservation",
					"ec2:CreateCapacityReservationFleet",
					"ec2:ModifyReservedInstances",
					"ec2:PurchaseCapacityBlock",
					"ec2:PurchaseCapacityBlockExtension",
					"ec2:PurchaseHostReservation",
					"ec2:PurchaseReservedInstancesOffering",
					"ec2:PurchaseScheduledInstances",

					// Database/cache reserved capacity.
					"dynamodb:PurchaseReservedCapacityOfferings",
					"rds:PurchaseReservedDBInstancesOffering",
					"elasticache:PurchaseReservedCacheNodesOffering",
					"memorydb:PurchaseReservedNodesOffering",
					"redshift:PurchaseReservedNodeOffering",
					"es:PurchaseReservedElasticsearchInstanceOffering",
					"es:PurchaseReservedInstanceOffering",

					// Other explicit purchasing APIs.
					"devicefarm:PurchaseOffering",
					"mediaconnect:PurchaseOffering",
					"medialive:PurchaseOffering",

					// Domain registrations are real purchases / renewals.
					"route53domains:RegisterDomain",
					"route53domains:RenewDomain",
					"route53domains:TransferDomain",

					// Paid account-level subscriptions.
					"shield:CreateSubscription",
					"ses:PutDeliverabilityDashboardOption",
				},
				Resource: []string{"*"},
			},
			{
				Sid:    "DenyExpensiveProvisionedCapacity",
				Effect: "Deny",
				Action: []string{
					"lambda:PutProvisionedConcurrencyConfig",
					"kinesis:UpdateAccountSettings",
					"kinesis:UpdateStreamWarmThroughput",
					"bedrock:CreateProvisionedModelThroughput",
					"bedrock:UpdateProvisionedModelThroughput",
				},
				Resource: []string{"*"},
			},
			{
				Sid:    "DenyHighBaselineCostResources",
				Effect: "Deny",
				Action: []string{
					"acm-pca:CreateCertificateAuthority",
					"cloudhsm:CreateCluster",
					"cloudhsm:CreateHsm",
					"outposts:CreateOutpost",
					"snowball:CreateCluster",
					"snowball:CreateJob",
					"kendra:CreateIndex",
					"redshift-serverless:CreateWorkgroup",
				},
				Resource: []string{"*"},
			},
			{
				Sid:      "DenyExportableCertificates",
				Effect:   "Deny",
				Action:   []string{"acm:RequestCertificate"},
				Resource: []string{"*"},
				Condition: map[string]map[string]any{
					"StringEquals": {
						"acm:Export": "ENABLED",
					},
				},
			},
			{
				Sid:    "DenyCertificateExport",
				Effect: "Deny",
				Action: []string{
					"acm:ExportCertificate",
					"acm:UpdateCertificateOptions",
				},
				Resource: []string{"*"},
			},
			{
				Sid:    "DenyMonitoringCostTraps",
				Effect: "Deny",
				Action: []string{
					"cloudwatch:PutInsightRule",
					"cloudwatch:PutManagedInsightRules",
					"cloudwatch:PutMetricStream",
				},
				Resource: []string{"*"},
			},
			{
				Sid:    "DenyRetentionLocks",
				Effect: "Deny",
				Action: []string{
					"backup:PutBackupVaultLockConfiguration",
					"backup:CreateLogicallyAirGappedBackupVault",

					"glacier:InitiateVaultLock",
					"glacier:CompleteVaultLock",

					"s3:PutBucketObjectLockConfiguration",
					"s3:PutObjectLegalHold",
					"s3:PutObjectRetention",

					"s3-object-lambda:PutObjectLegalHold",
					"s3-object-lambda:PutObjectRetention",

					"ec2:LockSnapshot",

					"rbin:LockRule",
				},
				Resource: []string{"*"},
			},
			{
				Sid:    "DenyPremiumSnapshotFeatures",
				Effect: "Deny",
				Action: []string{
					"ec2:EnableFastSnapshotRestores",
					"ec2:EnableFastLaunch",
				},
				Resource: []string{"*"},
			},
		},
	})
}
