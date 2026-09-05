//go:build wasip1

package main

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"

	"codeberg.org/megakuul/cloudjam/pkg/challenge"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/apigateway"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/applicationautoscaling"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/cloudfront"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/cloudwatch"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/codepipeline"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/dynamodb"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/ec2"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/ecr"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/ecs"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/elasticloadbalancingv2"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/guardduty"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/kinesis"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/kinesisanalyticsv2"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/lambda"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/rds"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/redshift"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/s3"
)

type trigger func() (bool, error)

const (
	designCheckInterval     = 2 * time.Minute
	resourceRequestInterval = 500 * time.Millisecond
)

var (
	resourceRequestLock sync.Mutex
	lastResourceRequest time.Time
)

func addDesignChecks(s *challenge.Scenario) {
	addDesign(s, "Blood-pressure service runs on Fargate", 10, "No ECS service uses the Fargate launch type or capacity provider.", fargateUsed)
	addDesign(s, "Lambda functions are tagged", 5, "No tagged Lambda function exists.", lambdaTagged)
	addDesign(s, "ECR expires old images", 5, "No ECR repository has a lifecycle policy.", ecrLifecycle)
	addDesign(s, "Architecture uses a cost-conscious footprint", 20, "The core platform is incomplete or uses oversized baseline capacity.", costOptimized)

	addDesign(s, "Artifact bucket is versioned", 5, "No S3 bucket has versioning enabled.", s3Versioned)
	addDesign(s, "VPC is tagged", 5, "No VPC has tags.", vpcTagged)
	addDesign(s, "ECR tags are immutable", 5, "No ECR repository uses immutable image tags.", ecrImmutable)
	addDesign(s, "GuardDuty detector is tagged", 5, "No enabled GuardDuty detector has tags.", guardDutyTagged)
	addDesign(s, "ECR has a VPC endpoint", 5, "No VPC endpoint targets ECR.", endpointFor(".ecr.api", ".ecr.dkr"))
	addDesign(s, "DynamoDB has a VPC endpoint", 5, "No VPC endpoint targets DynamoDB.", endpointFor(".dynamodb"))
	addDesign(s, "S3 has a VPC endpoint", 5, "No VPC endpoint targets S3.", endpointFor(".s3"))
	addDesign(s, "CodePipeline is used", 5, "No CodePipeline pipeline with stages exists.", pipelineUsed)
	addDesign(s, "CodeBuild is used", 5, "No CodePipeline action uses CodeBuild.", pipelineProvider("CodeBuild"))
	addDesign(s, "DynamoDB is used", 5, "The cloudraiser-iot DynamoDB table does not exist.", bloodPressureTable)
	addDesign(s, "CodeDeploy is used", 5, "No CodePipeline action uses CodeDeploy.", pipelineProvider("CodeDeploy", "CodeDeployToECS"))
	addDesign(s, "ALB access logs are enabled", 10, "No Application Load Balancer writes access logs to S3.", loadBalancerLogging)
	addDesign(s, "Kinesis analytics is tagged", 10, "No tagged Kinesis Data Analytics application exists.", analyticsTagged)

	addDesign(s, "SSH is not open to the internet", 10, "A security group allows TCP port 22 from the public internet.", noPublicSSH)
	addDesign(s, "Security groups are not open to the internet", 10, "A security group allows public IPv4 or IPv6 ingress.", noPublicIngress)
	addDesign(s, "S3 encryption is configured", 10, "No S3 bucket declares default server-side encryption.", s3Encrypted)
	addDesign(s, "GuardDuty is enabled", 10, "No enabled GuardDuty detector exists.", guardDutyEnabled)
	addDesign(s, "Kinesis encryption is enabled", 10, "No Kinesis stream uses KMS encryption.", kinesisEncrypted)
	addDesign(s, "Aurora deletion protection is enabled", 10, "No provisioned Aurora cluster has deletion protection enabled.", auroraDeletionProtection)
	addDesign(s, "Production deployment needs approval", 5, "No CodePipeline stage contains a manual approval action.", manualApproval)
	addDesign(s, "DynamoDB deletion protection is enabled", 5, "cloudraiser-iot does not have deletion protection enabled.", dynamodbDeletionProtection)

	addDesign(s, "VPC flow logs are enabled", 10, "No VPC Flow Log exists.", anyResource[*ec2.FlowLog])
	addDesign(s, "A CloudWatch alarm exists", 10, "No CloudWatch alarm exists.", anyResource[*cloudwatch.Alarm])
	addDesign(s, "ECS Container Insights is enabled", 10, "No ECS cluster enables containerInsights.", containerInsights)
	addDesign(s, "ECS service auto scales from two tasks", 10, "No ECS scalable target starts at two tasks and has a scaling policy.", ecsAutoScaling)
	addDesign(s, "ECS tasks send logs", 5, "No ECS task definition configures a log driver.", ecsTaskLogging)
	addDesign(s, "ECR repository policy is configured", 5, "No ECR repository has a repository policy.", ecrPolicy)

	addDesign(s, "CloudFront is used", 5, "No CloudFront distribution exists.", anyResource[*cloudfront.Distribution])
	addDesign(s, "Application Load Balancer is used", 5, "No Application Load Balancer exists.", applicationLoadBalancer)
	addDesign(s, "API Gateway X-Ray tracing is enabled", 5, "No API Gateway stage enables X-Ray tracing.", apiTracing)
	addDesign(s, "Kinesis Data Analytics is used", 5, "No Kinesis Data Analytics application exists.", analyticsUsed)
	addDesign(s, "Kinesis Data Streams is used", 5, "No Kinesis stream exists.", anyResource[*kinesis.Stream])
	addDesign(s, "Kinesis uses one provisioned shard", 5, "No Kinesis stream uses provisioned mode with exactly one shard.", singleKinesisShard)
	addDesign(s, "DynamoDB expires old measurements", 5, "cloudraiser-iot does not have TTL enabled.", dynamodbTTL)

	addDesign(s, "S3 lifecycle is configured", 20, "No S3 bucket has an enabled lifecycle rule.", s3Lifecycle)
	addDesign(s, "Application runs on ECS", 10, "No ECS service exists.", anyResource[*ecs.Service])
	addDesign(s, "Aurora retains backups", 10, "No provisioned Aurora cluster retains backups for more than seven days.", auroraBackups)
	addDesign(s, "GuardDuty findings are exported", 20, "No GuardDuty publishing destination exists.", guardDutyExport)

	addDesign(s, "End-to-end architecture is complete", 20, "One or more required ingestion, analytics, storage, compute, or API layers are missing.", architectureComplete)
}

func addDesign(s *challenge.Scenario, name string, points float64, reason string, check trigger) {
	design := challenge.NewDesignCheck(points, reason, check)
	design.Every = designCheckInterval
	s.AddCheck(name, design)
}

func anyResource[T aws.Resource]() (bool, error) {
	resources, err := listResources[T]()
	return len(resources) != 0, err
}

func matchingResource[T aws.Resource](match func(T) bool) (bool, error) {
	resources, err := listReadResources[T]()
	if err != nil {
		return false, err
	}
	for _, resource := range resources {
		if match(resource) {
			return true, nil
		}
	}
	return false, nil
}

func listResources[T aws.Resource]() (map[string]T, error) {
	waitForResourceRequest()
	return aws.List[T]()
}

func listReadResources[T aws.Resource]() (map[string]T, error) {
	resources, err := listResources[T]()
	if err != nil {
		return nil, err
	}
	return readResources(resources)
}

func listMatchingResources[T aws.Resource](resource T) (map[string]T, error) {
	waitForResourceRequest()
	resources, err := aws.ListMatching(resource)
	if err != nil {
		return nil, err
	}
	return readResources(resources)
}

func readResources[T aws.Resource](resources map[string]T) (map[string]T, error) {
	for identifier := range resources {
		resource, err := readResource[T](identifier)
		if err != nil {
			return nil, err
		}
		resources[identifier] = resource
	}
	return resources, nil
}

func readResource[T aws.Resource](identifier string) (T, error) {
	waitForResourceRequest()
	return aws.Read[T](identifier)
}

// Cloud Control applies a low account-level request burst limit.
func waitForResourceRequest() {
	resourceRequestLock.Lock()
	defer resourceRequestLock.Unlock()

	if wait := resourceRequestInterval - time.Since(lastResourceRequest); wait > 0 {
		time.Sleep(wait)
	}
	lastResourceRequest = time.Now()
}

func fargateUsed() (bool, error) {
	return matchingResource(func(service *ecs.Service) bool {
		if service.LaunchType != nil && strings.EqualFold(string(*service.LaunchType), "FARGATE") {
			return true
		}
		for _, provider := range service.CapacityProviderStrategy {
			if provider.CapacityProvider != nil && strings.HasPrefix(strings.ToUpper(*provider.CapacityProvider), "FARGATE") {
				return true
			}
		}
		return false
	})
}

func lambdaTagged() (bool, error) {
	return matchingResource(func(function *lambda.Function) bool { return len(function.Tags) != 0 })
}

func ecrLifecycle() (bool, error) {
	return matchingResource(func(repository *ecr.Repository) bool {
		return repository.LifecyclePolicy != nil && repository.LifecyclePolicy.LifecyclePolicyText != nil
	})
}

func costOptimized() (bool, error) {
	streams, err := listReadResources[*kinesis.Stream]()
	if err != nil {
		return false, err
	}
	tasks, err := listReadResources[*ecs.TaskDefinition]()
	if err != nil {
		return false, err
	}
	warehouses, err := listReadResources[*redshift.Cluster]()
	if err != nil {
		return false, err
	}
	databases, err := listReadResources[*rds.DBCluster]()
	if err != nil {
		return false, err
	}
	if len(streams) == 0 || len(tasks) == 0 || len(warehouses) == 0 || len(databases) == 0 {
		return false, nil
	}
	for _, stream := range streams {
		if stream.ShardCount != nil && *stream.ShardCount > 1 {
			return false, nil
		}
	}
	for _, task := range tasks {
		if exceeds(task.Cpu, 1024) || exceeds(task.Memory, 2048) {
			return false, nil
		}
	}
	for _, warehouse := range warehouses {
		if warehouse.NumberOfNodes != nil && *warehouse.NumberOfNodes > 1 {
			return false, nil
		}
	}
	return true, nil
}

func exceeds(value *string, maximum int) bool {
	if value == nil {
		return false
	}
	number, err := strconv.Atoi(*value)
	return err == nil && number > maximum
}

func s3Versioned() (bool, error) {
	return matchingResource(func(bucket *s3.Bucket) bool {
		return bucket.VersioningConfiguration != nil && bucket.VersioningConfiguration.Status != nil &&
			strings.EqualFold(string(*bucket.VersioningConfiguration.Status), "Enabled")
	})
}

func vpcTagged() (bool, error) {
	return matchingResource(func(vpc *ec2.VPC) bool { return len(vpc.Tags) != 0 })
}

func ecrImmutable() (bool, error) {
	return matchingResource(func(repository *ecr.Repository) bool {
		return repository.ImageTagMutability != nil && strings.HasPrefix(string(*repository.ImageTagMutability), "IMMUTABLE")
	})
}

func guardDutyTagged() (bool, error) {
	return matchingResource(func(detector *guardduty.Detector) bool {
		return detector.Enable != nil && *detector.Enable && len(detector.Tags) != 0
	})
}

func endpointFor(suffixes ...string) trigger {
	return func() (bool, error) {
		return matchingResource(func(endpoint *ec2.VPCEndpoint) bool {
			if endpoint.ServiceName == nil {
				return false
			}
			for _, suffix := range suffixes {
				if strings.HasSuffix(*endpoint.ServiceName, suffix) {
					return true
				}
			}
			return false
		})
	}
}

func pipelineUsed() (bool, error) {
	return matchingResource(func(pipeline *codepipeline.Pipeline) bool { return len(pipeline.Stages) != 0 })
}

func pipelineProvider(providers ...string) trigger {
	return func() (bool, error) {
		return matchingResource(func(pipeline *codepipeline.Pipeline) bool {
			for _, stage := range pipeline.Stages {
				for _, action := range stage.Actions {
					if action.ActionTypeId == nil || action.ActionTypeId.Provider == nil {
						continue
					}
					for _, provider := range providers {
						if strings.EqualFold(*action.ActionTypeId.Provider, provider) {
							return true
						}
					}
				}
			}
			return false
		})
	}
}

func bloodPressureTable() (bool, error) {
	return matchingResource(func(table *dynamodb.Table) bool {
		return table.TableName != nil && *table.TableName == "cloudraiser-iot"
	})
}

func loadBalancerLogging() (bool, error) {
	return matchingResource(func(loadBalancer *elasticloadbalancingv2.LoadBalancer) bool {
		if loadBalancer.Type != nil && !strings.EqualFold(*loadBalancer.Type, "application") {
			return false
		}
		for _, attribute := range loadBalancer.LoadBalancerAttributes {
			if attribute.Key != nil && attribute.Value != nil && *attribute.Key == "access_logs.s3.enabled" && *attribute.Value == "true" {
				return true
			}
		}
		return false
	})
}

func analyticsTagged() (bool, error) {
	return matchingResource(func(application *kinesisanalyticsv2.Application) bool { return len(application.Tags) != 0 })
}

func noPublicSSH() (bool, error) {
	groups, err := listReadResources[*ec2.SecurityGroup]()
	if err != nil || len(groups) == 0 {
		return false, err
	}
	for _, group := range groups {
		for _, rule := range group.SecurityGroupIngress {
			if public(rule.CidrIp, rule.CidrIpv6) && portIncludes(rule.FromPort, rule.ToPort, 22) {
				return false, nil
			}
		}
	}
	return true, nil
}

func noPublicIngress() (bool, error) {
	groups, err := listReadResources[*ec2.SecurityGroup]()
	if err != nil || len(groups) == 0 {
		return false, err
	}
	for _, group := range groups {
		for _, rule := range group.SecurityGroupIngress {
			if public(rule.CidrIp, rule.CidrIpv6) {
				return false, nil
			}
		}
	}
	return true, nil
}

func public(ipv4, ipv6 *string) bool {
	return ipv4 != nil && *ipv4 == "0.0.0.0/0" || ipv6 != nil && *ipv6 == "::/0"
}

func portIncludes(from, to *int, port int) bool {
	return from == nil || to == nil || *from <= port && *to >= port
}

func s3Encrypted() (bool, error) {
	return matchingResource(func(bucket *s3.Bucket) bool {
		return bucket.BucketEncryption != nil && len(bucket.BucketEncryption.ServerSideEncryptionConfiguration) != 0
	})
}

func guardDutyEnabled() (bool, error) {
	return matchingResource(func(detector *guardduty.Detector) bool { return detector.Enable != nil && *detector.Enable })
}

func kinesisEncrypted() (bool, error) {
	return matchingResource(func(stream *kinesis.Stream) bool {
		return stream.StreamEncryption != nil && stream.StreamEncryption.EncryptionType != nil &&
			*stream.StreamEncryption.EncryptionType == kinesis.StreamEncryptionEncryptionTypeKMS
	})
}

func provisionedAurora(cluster *rds.DBCluster) bool {
	return cluster.Engine != nil && strings.HasPrefix(*cluster.Engine, "aurora") &&
		(cluster.EngineMode == nil || !strings.EqualFold(*cluster.EngineMode, "serverless")) &&
		cluster.ServerlessV2ScalingConfiguration == nil
}

func auroraDeletionProtection() (bool, error) {
	return matchingResource(func(cluster *rds.DBCluster) bool {
		return provisionedAurora(cluster) && cluster.DeletionProtection != nil && *cluster.DeletionProtection
	})
}

func manualApproval() (bool, error) {
	return matchingResource(func(pipeline *codepipeline.Pipeline) bool {
		for _, stage := range pipeline.Stages {
			for _, action := range stage.Actions {
				if action.ActionTypeId != nil && action.ActionTypeId.Provider != nil &&
					strings.EqualFold(*action.ActionTypeId.Provider, "Manual") {
					return true
				}
			}
		}
		return false
	})
}

func dynamodbDeletionProtection() (bool, error) {
	return matchingResource(func(table *dynamodb.Table) bool {
		return table.TableName != nil && *table.TableName == "cloudraiser-iot" &&
			table.DeletionProtectionEnabled != nil && *table.DeletionProtectionEnabled
	})
}

func containerInsights() (bool, error) {
	return matchingResource(func(cluster *ecs.Cluster) bool {
		for _, setting := range cluster.ClusterSettings {
			if setting.Name != nil && setting.Value != nil && *setting.Name == "containerInsights" && *setting.Value == "enabled" {
				return true
			}
		}
		return false
	})
}

func ecsAutoScaling() (bool, error) {
	targets, err := listMatchingResources(&applicationautoscaling.ScalableTarget{ServiceNamespace: new("ecs")})
	if err != nil {
		return false, err
	}
	policies, err := listMatchingResources(&applicationautoscaling.ScalingPolicy{ServiceNamespace: new("ecs")})
	if err != nil {
		return false, err
	}
	if len(policies) == 0 {
		return false, nil
	}
	for _, target := range targets {
		if target.ServiceNamespace != nil && *target.ServiceNamespace == "ecs" &&
			target.MinCapacity != nil && *target.MinCapacity >= 2 && target.MaxCapacity != nil && *target.MaxCapacity >= 2 {
			return true, nil
		}
	}
	return false, nil
}

func ecsTaskLogging() (bool, error) {
	return matchingResource(func(task *ecs.TaskDefinition) bool {
		for _, container := range task.ContainerDefinitions {
			if container.LogConfiguration != nil {
				return true
			}
		}
		return false
	})
}

func ecrPolicy() (bool, error) {
	return matchingResource(func(repository *ecr.Repository) bool {
		return len(json.RawMessage(repository.RepositoryPolicyText)) != 0
	})
}

func applicationLoadBalancer() (bool, error) {
	return matchingResource(func(loadBalancer *elasticloadbalancingv2.LoadBalancer) bool {
		return loadBalancer.Type == nil || strings.EqualFold(*loadBalancer.Type, "application")
	})
}

func apiTracing() (bool, error) {
	apis, err := listResources[*apigateway.RestApi]()
	if err != nil {
		return false, err
	}
	for identifier, api := range apis {
		apiID := identifier
		if api.RestApiId != nil {
			apiID = *api.RestApiId
		}
		stages, err := listMatchingResources(&apigateway.Stage{RestApiId: new(apiID)})
		if err != nil {
			return false, err
		}
		for _, stage := range stages {
			if stage.TracingEnabled != nil && *stage.TracingEnabled {
				return true, nil
			}
		}
	}
	return false, nil
}

func analyticsUsed() (bool, error) {
	return anyResource[*kinesisanalyticsv2.Application]()
}

func singleKinesisShard() (bool, error) {
	return matchingResource(func(stream *kinesis.Stream) bool {
		return stream.ShardCount != nil && *stream.ShardCount == 1 &&
			(stream.StreamModeDetails == nil || stream.StreamModeDetails.StreamMode == nil ||
				*stream.StreamModeDetails.StreamMode == kinesis.StreamModeDetailsStreamModePROVISIONED)
	})
}

func dynamodbTTL() (bool, error) {
	return matchingResource(func(table *dynamodb.Table) bool {
		return table.TableName != nil && *table.TableName == "cloudraiser-iot" && table.TimeToLiveSpecification != nil &&
			table.TimeToLiveSpecification.Enabled != nil && *table.TimeToLiveSpecification.Enabled
	})
}

func s3Lifecycle() (bool, error) {
	return matchingResource(func(bucket *s3.Bucket) bool {
		if bucket.LifecycleConfiguration == nil {
			return false
		}
		for _, rule := range bucket.LifecycleConfiguration.Rules {
			if rule.Status != nil && strings.EqualFold(string(*rule.Status), "Enabled") {
				return true
			}
		}
		return false
	})
}

func auroraBackups() (bool, error) {
	return matchingResource(func(cluster *rds.DBCluster) bool {
		return provisionedAurora(cluster) && cluster.BackupRetentionPeriod != nil && *cluster.BackupRetentionPeriod > 7
	})
}

func guardDutyExport() (bool, error) {
	detectors, err := listResources[*guardduty.Detector]()
	if err != nil {
		return false, err
	}
	for identifier, detector := range detectors {
		detectorID := identifier
		if detector.Id != nil {
			detectorID = *detector.Id
		}
		destinations, err := listMatchingResources(&guardduty.PublishingDestination{DetectorId: new(detectorID)})
		if err != nil {
			return false, err
		}
		if len(destinations) != 0 {
			return true, nil
		}
	}
	return false, nil
}

func architectureComplete() (bool, error) {
	checks := []trigger{
		anyResource[*kinesis.Stream], analyticsUsed, bloodPressureTable, anyResource[*redshift.Cluster],
		func() (bool, error) { return matchingResource(provisionedAurora) }, anyResource[*ecs.Service],
		anyResource[*elasticloadbalancingv2.LoadBalancer], anyResource[*apigateway.RestApi],
	}
	for _, check := range checks {
		passed, err := check()
		if err != nil || !passed {
			return false, err
		}
	}
	return true, nil
}
