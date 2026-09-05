//go:build wasip1

package main

import (
	"fmt"
	"strings"

	"codeberg.org/megakuul/cloudjam/pkg/challenge"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/policy"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/ec2"
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

type iamInstanceProfile struct {
	InstanceProfileName *string  `json:"InstanceProfileName,omitempty"`
	Path                *string  `json:"Path,omitempty"`
	Roles               []string `json:"Roles"`
}

func (*iamInstanceProfile) CloudJamType() string { return "AWS::IAM::InstanceProfile" }

type roleSpec struct {
	name       string
	principals []string
	actions    policy.Actions
}

func bootstrap(*challenge.Scenario) error {
	existingNetworks, err := networkNames()
	if err != nil {
		return err
	}
	for _, network := range []struct {
		name     string
		cidr     string
		internet bool
	}{
		{name: "Data-Ingestion-VPC", cidr: "10.51.0.0/16", internet: true},
		{name: "Data-Processing-VPC", cidr: "10.52.0.0/16", internet: true},
		{name: "Data-Extraction-VPC", cidr: "10.53.0.0/16"},
	} {
		if existingNetworks[network.name] {
			continue
		}
		if err := createNetwork(network.name, network.cidr, network.internet); err != nil {
			return err
		}
	}

	applicationActions := policy.Actions{
		"dynamodb:DescribeTable", "dynamodb:Scan", "logs:CreateLogStream", "logs:PutLogEvents",
		"ssm:GetParameter", "ssm:GetParameters", "ssm:PutParameter",
	}
	roles := []roleSpec{
		{
			name:       "game-ec2-role",
			principals: []string{"ec2.amazonaws.com"},
			actions:    applicationActions,
		},
		{
			name:       "game-cloudwatch-event-role",
			principals: []string{"events.amazonaws.com"},
			actions:    policy.Actions{"batch:SubmitJob"},
		},
		{
			name:       "game-lambda-role",
			principals: []string{"lambda.amazonaws.com"},
			actions: policy.Actions{
				"batch:SubmitJob", "dynamodb:PutItem", "ec2:CreateNetworkInterface",
				"ec2:DeleteNetworkInterface", "ec2:DescribeNetworkInterfaces",
				"logs:CreateLogGroup", "logs:CreateLogStream", "logs:PutLogEvents",
			},
		},
		{
			name:       "game-ecs-role",
			principals: []string{"ecs-tasks.amazonaws.com"},
			actions:    applicationActions,
		},
		{
			name:       "game-ecs-execution-role",
			principals: []string{"ecs-tasks.amazonaws.com"},
			actions: policy.Actions{
				"ecr:BatchCheckLayerAvailability", "ecr:GetAuthorizationToken", "ecr:GetDownloadUrlForLayer",
				"ecr:BatchGetImage", "logs:CreateLogStream", "logs:PutLogEvents",
			},
		},
		{
			name:       "game-scheduler-role",
			principals: []string{"scheduler.amazonaws.com"},
			actions:    policy.Actions{"batch:SubmitJob"},
		},
		{
			name:       "game-codedeploy-role",
			principals: []string{"codedeploy.amazonaws.com"},
			actions: policy.Actions{
				"cloudwatch:*", "ec2:*", "ecs:*", "elasticloadbalancing:*", "lambda:*", "s3:*", "sns:*",
			},
		},
		{
			name:       "game-codepipeline-role",
			principals: []string{"codepipeline.amazonaws.com"},
			actions: policy.Actions{
				"codedeploy:*", "iam:PassRole", "lambda:*", "s3:*",
			},
		},
	}
	existingRoles, err := resourceIdentifiers[*iamRole]()
	if err != nil {
		return err
	}
	for _, role := range roles {
		if existingRoles[role.name] {
			continue
		}
		if err := createRole(role); err != nil {
			return err
		}
	}
	existingProfiles, err := resourceIdentifiers[*iamInstanceProfile]()
	if err != nil {
		return err
	}
	if existingProfiles["game-ec2-profile"] {
		return nil
	}
	if _, err := aws.Create(&iamInstanceProfile{
		InstanceProfileName: new("game-ec2-profile"),
		Path:                new("/"),
		Roles:               []string{"game-ec2-role"},
	}); err != nil {
		return fmt.Errorf("create game-ec2-profile: %w", err)
	}
	return nil
}

func networkNames() (map[string]bool, error) {
	resources, err := listReadResources[*ec2.VPC]()
	if err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for _, vpc := range resources {
		for _, tag := range vpc.Tags {
			if tag.Key != nil && tag.Value != nil && *tag.Key == "Name" {
				names[*tag.Value] = true
			}
		}
	}
	return names, nil
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

func createNetwork(name, cidr string, internet bool) error {
	networkPrefix := strings.TrimSuffix(cidr, ".0.0/16")
	vpc, err := aws.Create(&ec2.VPC{
		CidrBlock:          new(cidr),
		EnableDnsHostnames: new(true),
		EnableDnsSupport:   new(true),
		Tags:               []ec2.VPCTag{{Key: new("Name"), Value: new(name)}},
	})
	if err != nil {
		return fmt.Errorf("create %s: %w", name, err)
	}

	routeTable, err := aws.Create(&ec2.RouteTable{
		VpcId: new(vpc),
		Tags:  []ec2.RouteTableTag{{Key: new("Name"), Value: new(name + "-routes")}},
	})
	if err != nil {
		return fmt.Errorf("create %s route table: %w", name, err)
	}
	for index, zone := range []string{"us-east-1a", "us-east-1b"} {
		subnet, err := aws.Create(&ec2.Subnet{
			AvailabilityZone:    new(zone),
			CidrBlock:           new(fmt.Sprintf("%s.%d.0/24", networkPrefix, index+1)),
			MapPublicIpOnLaunch: new(internet),
			VpcId:               new(vpc),
			Tags:                []ec2.SubnetTag{{Key: new("Name"), Value: new(fmt.Sprintf("%s-%c", name, 'a'+index))}},
		})
		if err != nil {
			return fmt.Errorf("create %s subnet: %w", name, err)
		}
		if _, err := aws.Create(&ec2.SubnetRouteTableAssociation{
			RouteTableId: new(routeTable),
			SubnetId:     new(subnet),
		}); err != nil {
			return fmt.Errorf("associate %s subnet: %w", name, err)
		}
	}
	if !internet {
		return nil
	}

	gateway, err := aws.Create(&ec2.InternetGateway{
		Tags: []ec2.InternetGatewayTag{{Key: new("Name"), Value: new(name + "-igw")}},
	})
	if err != nil {
		return fmt.Errorf("create %s internet gateway: %w", name, err)
	}
	if _, err := aws.Create(&ec2.VPCGatewayAttachment{
		InternetGatewayId: new(gateway),
		VpcId:             new(vpc),
	}); err != nil {
		return fmt.Errorf("attach %s internet gateway: %w", name, err)
	}
	if _, err := aws.Create(&ec2.Route{
		DestinationCidrBlock: new("0.0.0.0/0"),
		GatewayId:            new(gateway),
		RouteTableId:         new(routeTable),
	}); err != nil {
		return fmt.Errorf("route %s internet traffic: %w", name, err)
	}
	return nil
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
		Description: new("Preconfigured role for WorldSkills Lyon 2024 Day 1"),
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
