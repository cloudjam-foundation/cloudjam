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
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/batch"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/codepipeline"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/dynamodb"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/ec2"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/ecr"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/ecs"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/lambda"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/rds"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/s3"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/scheduler"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/wafv2"
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
	addDesign(s, "S3 lifecycle is configured", 5, "No S3 bucket has an enabled lifecycle rule.", s3Lifecycle)
	addDesign(s, "API Gateway is tagged", 5, "No tagged REST API exists.", apiTagged)
	addDesign(s, "AWS Batch is used", 10, "A complete AWS Batch compute environment, queue, and job definition does not exist.", batchUsed)
	addDesign(s, "Architecture uses a cost-conscious footprint", 20, "The core platform is incomplete or uses oversized baseline capacity.", costOptimized)

	addDesign(s, "EventBridge Scheduler runs the batch job", 5, "No enabled EventBridge schedule targets AWS Batch.", batchScheduled)
	addDesign(s, "AWS Batch completed successfully", 10, "DataProcessingAPP has not recorded a successful batch run.", batchSucceeded)
	addDesign(s, "Artifact bucket is versioned", 5, "No S3 bucket has versioning enabled.", s3Versioned)
	addDesign(s, "VPC peering is configured", 10, "No VPC peering connection exists.", anyResource[*ec2.VPCPeeringConnection])
	addDesign(s, "DynamoDB deletion protection is enabled for operations", 5, "FeedbackTable does not have deletion protection enabled.", dynamodbDeletionProtection)
	addDesign(s, "ECR tags are immutable", 5, "No ECR repository uses immutable image tags.", ecrImmutable)
	addDesign(s, "DynamoDB is tagged", 5, "FeedbackTable has no tags.", dynamodbTagged)
	addDesign(s, "CodePipeline is used", 5, "No CodePipeline pipeline with stages exists.", pipelineUsed)
	addDesign(s, "ECR has a VPC endpoint", 5, "No VPC endpoint targets ECR.", endpointFor(".ecr.api", ".ecr.dkr"))
	addDesign(s, "DynamoDB is used", 5, "FeedbackTable does not exist.", feedbackTable)
	addDesign(s, "CodeDeploy is used", 5, "No CodePipeline action uses CodeDeploy.", pipelineProvider("CodeDeploy", "CodeDeployToECS"))

	addDesign(s, "DynamoDB encryption is enabled", 10, "FeedbackTable does not declare server-side encryption.", dynamodbEncrypted)
	addDesign(s, "S3 encryption is configured", 10, "No S3 bucket declares default server-side encryption.", s3Encrypted)
	addDesign(s, "AWS WAF is configured", 20, "No regional Web ACL exists.", wafConfigured)
	addDesign(s, "ECR repository policy is configured", 10, "No ECR repository has a repository policy.", ecrPolicy)
	addDesign(s, "RDS deletion protection is enabled", 10, "No provisioned PostgreSQL-compatible RDS database has deletion protection enabled.", rdsDeletionProtection)
	addDesign(s, "Production deployment needs approval", 5, "No CodePipeline stage contains a manual approval action.", manualApproval)
	addDesign(s, "DynamoDB deletion protection is enabled for security", 5, "FeedbackTable does not have deletion protection enabled.", dynamodbDeletionProtection)

	addDesign(s, "DynamoDB point-in-time recovery is enabled", 10, "FeedbackTable does not have point-in-time recovery enabled.", dynamodbBackedUp)
	addDesign(s, "VPC flow logs are enabled", 10, "No VPC Flow Log exists.", anyResource[*ec2.FlowLog])
	addDesign(s, "RDS exports logs", 10, "No provisioned PostgreSQL-compatible RDS database exports logs to CloudWatch.", rdsLogging)
	addDesign(s, "DynamoDB deletion protection is enabled for reliability", 10, "FeedbackTable does not have deletion protection enabled.", dynamodbDeletionProtection)

	addDesign(s, "DynamoDB uses provisioned capacity", 5, "FeedbackTable does not use provisioned throughput.", dynamodbProvisioned)
	addDesign(s, "API Gateway X-Ray tracing is enabled", 5, "No API Gateway stage enables X-Ray tracing.", apiTracing)
	addDesign(s, "Application runs on ECS", 5, "No ECS service exists.", anyResource[*ecs.Service])

	addDesign(s, "Parameter Store uses Intelligent-Tiering", 10, "No pipeline status or result exists in Parameter Store.", intelligentTiering)
	addDesign(s, "RDS retains backups", 10, "No provisioned PostgreSQL-compatible RDS database retains backups for more than seven days.", rdsBackups)

	addDesign(s, "End-to-end architecture is complete", 20, "One or more required ingestion, batch, database, extraction, or result layers are missing.", architectureComplete)
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

func apiTagged() (bool, error) {
	return matchingResource(func(api *apigateway.RestApi) bool { return len(api.Tags) != 0 })
}

func batchUsed() (bool, error) {
	for _, check := range []trigger{
		anyResource[*batch.ComputeEnvironment],
		anyResource[*batch.JobQueue],
		anyResource[*batch.JobDefinition],
	} {
		passed, err := check()
		if err != nil || !passed {
			return false, err
		}
	}
	return true, nil
}

func costOptimized() (bool, error) {
	environments, err := listReadResources[*batch.ComputeEnvironment]()
	if err != nil {
		return false, err
	}
	databases, err := listReadResources[*rds.DBInstance]()
	if err != nil {
		return false, err
	}
	tasks, err := listReadResources[*ecs.TaskDefinition]()
	if err != nil {
		return false, err
	}
	if len(environments) == 0 || len(databases) == 0 || len(tasks) == 0 {
		return false, nil
	}
	for _, environment := range environments {
		if environment.ComputeResources == nil || environment.ComputeResources.Type == nil ||
			!strings.HasPrefix(strings.ToUpper(*environment.ComputeResources.Type), "FARGATE") {
			return false, nil
		}
		if environment.ComputeResources.MaxvCpus != nil && *environment.ComputeResources.MaxvCpus > 16 {
			return false, nil
		}
	}
	for _, database := range databases {
		if !postgresDatabase(database) || database.DBInstanceClass == nil ||
			(!strings.Contains(*database.DBInstanceClass, ".micro") && !strings.Contains(*database.DBInstanceClass, ".small")) {
			return false, nil
		}
	}
	for _, task := range tasks {
		if exceeds(task.Cpu, 1024) || exceeds(task.Memory, 2048) {
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

func batchScheduled() (bool, error) {
	return matchingResource(func(schedule *scheduler.Schedule) bool {
		return schedule.State != nil && *schedule.State == scheduler.ScheduleStateENABLED &&
			schedule.Target != nil && schedule.Target.Arn != nil && strings.Contains(strings.ToLower(*schedule.Target.Arn), "batch")
	})
}

func batchSucceeded() (bool, error) {
	value, err := parameterValue(batchStatusParameter)
	return value == "success", err
}

func s3Versioned() (bool, error) {
	return matchingResource(func(bucket *s3.Bucket) bool {
		return bucket.VersioningConfiguration != nil && bucket.VersioningConfiguration.Status != nil &&
			strings.EqualFold(string(*bucket.VersioningConfiguration.Status), "Enabled")
	})
}

func dynamodbDeletionProtection() (bool, error) {
	return matchingResource(func(table *dynamodb.Table) bool {
		return table.TableName != nil && *table.TableName == "FeedbackTable" &&
			table.DeletionProtectionEnabled != nil && *table.DeletionProtectionEnabled
	})
}

func ecrImmutable() (bool, error) {
	return matchingResource(func(repository *ecr.Repository) bool {
		return repository.ImageTagMutability != nil && strings.HasPrefix(string(*repository.ImageTagMutability), "IMMUTABLE")
	})
}

func dynamodbTagged() (bool, error) {
	return matchingResource(func(table *dynamodb.Table) bool {
		return table.TableName != nil && *table.TableName == "FeedbackTable" && len(table.Tags) != 0
	})
}

func pipelineUsed() (bool, error) {
	return matchingResource(func(pipeline *codepipeline.Pipeline) bool { return len(pipeline.Stages) != 0 })
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

func feedbackTable() (bool, error) {
	return matchingResource(func(table *dynamodb.Table) bool {
		return table.TableName != nil && *table.TableName == "FeedbackTable"
	})
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

func dynamodbEncrypted() (bool, error) {
	return matchingResource(func(table *dynamodb.Table) bool {
		return table.TableName != nil && *table.TableName == "FeedbackTable" && table.SSESpecification != nil &&
			table.SSESpecification.SSEEnabled != nil && *table.SSESpecification.SSEEnabled
	})
}

func s3Encrypted() (bool, error) {
	return matchingResource(func(bucket *s3.Bucket) bool {
		return bucket.BucketEncryption != nil && len(bucket.BucketEncryption.ServerSideEncryptionConfiguration) != 0
	})
}

func wafConfigured() (bool, error) {
	webACLs, err := listMatchingResources(&wafv2.WebACL{Scope: new(wafv2.WebACLScopeREGIONAL)})
	return len(webACLs) != 0, err
}

func ecrPolicy() (bool, error) {
	return matchingResource(func(repository *ecr.Repository) bool {
		return len(json.RawMessage(repository.RepositoryPolicyText)) != 0
	})
}

func postgresDatabase(database *rds.DBInstance) bool {
	return database.Engine != nil && strings.Contains(strings.ToLower(*database.Engine), "postgres") &&
		database.DBClusterIdentifier == nil
}

func rdsDeletionProtection() (bool, error) {
	return matchingResource(func(database *rds.DBInstance) bool {
		return postgresDatabase(database) && database.DeletionProtection != nil && *database.DeletionProtection
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

func dynamodbBackedUp() (bool, error) {
	return matchingResource(func(table *dynamodb.Table) bool {
		return table.TableName != nil && *table.TableName == "FeedbackTable" &&
			table.PointInTimeRecoverySpecification != nil &&
			table.PointInTimeRecoverySpecification.PointInTimeRecoveryEnabled != nil &&
			*table.PointInTimeRecoverySpecification.PointInTimeRecoveryEnabled
	})
}

func rdsLogging() (bool, error) {
	return matchingResource(func(database *rds.DBInstance) bool {
		return postgresDatabase(database) && len(database.EnableCloudwatchLogsExports) != 0
	})
}

func dynamodbProvisioned() (bool, error) {
	return matchingResource(func(table *dynamodb.Table) bool {
		return table.TableName != nil && *table.TableName == "FeedbackTable" &&
			(table.BillingMode == nil || strings.EqualFold(*table.BillingMode, "PROVISIONED")) &&
			table.ProvisionedThroughput != nil
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

func intelligentTiering() (bool, error) {
	value, err := parameterValue(batchStatusParameter)
	return value != "", err
}

func rdsBackups() (bool, error) {
	return matchingResource(func(database *rds.DBInstance) bool {
		return postgresDatabase(database) && database.BackupRetentionPeriod != nil && *database.BackupRetentionPeriod > 7
	})
}

func architectureComplete() (bool, error) {
	checks := []trigger{
		feedbackTable,
		anyResource[*lambda.Function],
		anyResource[*apigateway.RestApi],
		batchUsed,
		func() (bool, error) { return matchingResource(postgresDatabase) },
		anyResource[*ecs.Service],
		intelligentTiering,
	}
	for _, check := range checks {
		passed, err := check()
		if err != nil || !passed {
			return false, err
		}
	}
	return true, nil
}
