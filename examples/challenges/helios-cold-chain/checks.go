//go:build wasip1

package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"codeberg.org/megakuul/cloudjam/pkg/challenge"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/cloudwatch"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/events"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/lambda"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/logs"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/s3"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/secretsmanager"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/sns"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/sqs"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/stepfunctions"
)

type trigger func() (bool, error)

const (
	designCheckInterval     = 2 * time.Minute
	resourceCacheDuration   = 90 * time.Second
	resourceRequestInterval = 500 * time.Millisecond
)

type cachedResources struct {
	loaded time.Time
	value  any
}

var (
	resourceCache       = map[string]cachedResources{}
	resourceCacheLock   sync.Mutex
	resourceRequestLock sync.Mutex
	lastResourceRequest time.Time
)

func addDesignChecks(s *challenge.Scenario) {
	addDesign(s, "Custom event bus is tagged", 10, "The helios-cold-chain event bus is missing or untagged.", eventBusTagged)
	addDesign(s, "Lambda functions are tagged", 10, "Five tagged helios Lambda functions do not exist.", lambdaTagged)
	addDesign(s, "Log retention is configured", 10, "One or more helios Lambda log groups has no finite retention.", logRetention)
	addDesign(s, "Operations dashboard is configured", 10, "No Helios CloudWatch dashboard exists.", dashboardConfigured)
	addDesign(s, "Failure alarm is configured", 10, "No enabled Helios CloudWatch alarm exists.", alarmConfigured)
	addDesign(s, "Lambda tracing is active", 10, "One or more helios Lambda functions does not use active X-Ray tracing.", lambdaTracing)

	addDesign(s, "Telemetry queues are encrypted", 10, "The telemetry queue or dead-letter queue does not use KMS encryption.", queuesEncrypted)
	addDesign(s, "Excursion topic is encrypted", 10, "helios-excursions does not use KMS encryption.", topicEncrypted)
	addDesign(s, "Result objects use KMS encryption", 10, "The Helios result bucket does not declare aws:kms default encryption.", bucketEncrypted)
	addDesign(s, "Temperature limits are encrypted", 10, "The temperature-limits secret does not declare a KMS key.", secretEncrypted)
	addDesign(s, "Result bucket is private and owned", 10, "Public access blocking or bucket-owner ownership is missing from the result bucket.", bucketProtected)
	addDesign(s, "Temperature limits are tagged", 10, "The temperature-limits secret is untagged.", secretTagged)

	addDesign(s, "EventBridge invokes the relay", 10, "No enabled helios-cold-chain rule targets the relay Lambda.", eventRuleConfigured)
	addDesign(s, "Event delivery has retries and a DLQ", 10, "The relay rule target has no retry policy or dead-letter queue.", eventDeliveryReliable)
	addDesign(s, "Telemetry queue has a redrive policy", 10, "helios-telemetry does not redrive failures.", queueRedrive)
	addDesign(s, "Dead-letter messages are retained", 10, "helios-telemetry-dlq retains messages for less than seven days.", deadLetterRetention)
	addDesign(s, "Queue consumer reports partial failures", 10, "The SQS event source mapping is missing, disabled, incorrectly sized, or lacks ReportBatchItemFailures.", queueConsumerConfigured)
	addDesign(s, "Workflow is Express", 10, "No Helios Express Step Functions state machine exists.", expressWorkflow)
	addDesign(s, "Workflow logging is enabled", 10, "The Helios workflow does not send execution logs to CloudWatch.", workflowLogging)
	addDesign(s, "Workflow tracing is enabled", 10, "The Helios workflow does not enable X-Ray tracing.", workflowTracing)
	addDesign(s, "Result bucket is versioned", 10, "The Helios result bucket does not enable versioning.", bucketVersioned)
	addDesign(s, "Telemetry archive is configured", 10, "No short-retention archive captures the helios-cold-chain event bus.", archiveConfigured)
	addDesign(s, "Excursion topic is tagged", 10, "The helios-excursions SNS topic is missing or untagged.", topicTagged)

	addDesign(s, "Queue long polling is enabled", 10, "helios-telemetry does not enable long polling.", queueLongPolling)
	addDesign(s, "Queue visibility protects processing", 10, "helios-telemetry has a visibility timeout below 60 seconds.", queueVisibility)
	addDesign(s, "Functions use arm64", 10, "One or more helios Lambda functions does not use arm64.", lambdaArm64)
	addDesign(s, "Function timeouts are bounded", 10, "One or more helios Lambda functions has an excessive timeout.", lambdaTimeout)
	addDesign(s, "Result lifecycle is configured", 10, "The Helios result bucket has no enabled lifecycle rule.", bucketLifecycle)
	addDesign(s, "Results use Intelligent-Tiering", 10, "The Helios result bucket has no enabled Intelligent-Tiering configuration.", bucketIntelligentTiering)
	addDesign(s, "Functions have a small memory footprint", 10, "One or more helios Lambda functions exceeds 512 MB.", lambdaMemory)

	addDesign(s, "End-to-end architecture is complete", 20, "One or more required ingestion, event, queue, workflow, analysis, alert, storage, or query layers is missing.", architectureComplete)
}

func addDesign(s *challenge.Scenario, name string, points float64, reason string, check trigger) {
	design := challenge.NewDesignCheck(points, reason, check)
	design.Every = designCheckInterval
	s.AddCheck(name, design)
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

func readResource[T aws.Resource](identifier string) (T, error) {
	waitForResourceRequest()
	return aws.Read[T](identifier)
}

func listReadResources[T aws.Resource]() (map[string]T, error) {
	var resource T
	key := fmt.Sprintf("%T", resource)
	return cachedReadResources(key, func() (map[string]T, error) { return listResources[T]() })
}

func listMatchingResources[T aws.Resource](resource T) (map[string]T, error) {
	encoded, err := json.Marshal(resource)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("%T:%s", resource, encoded)
	return cachedReadResources(key, func() (map[string]T, error) {
		waitForResourceRequest()
		return aws.ListMatching(resource)
	})
}

func cachedReadResources[T aws.Resource](key string, list func() (map[string]T, error)) (map[string]T, error) {
	resourceCacheLock.Lock()
	defer resourceCacheLock.Unlock()
	if cached, exists := resourceCache[key]; exists && time.Since(cached.loaded) < resourceCacheDuration {
		return cached.value.(map[string]T), nil
	}
	resources, err := list()
	if err != nil {
		return nil, err
	}
	for identifier := range resources {
		resources[identifier], err = readResource[T](identifier)
		if err != nil {
			return nil, err
		}
	}
	resourceCache[key] = cachedResources{loaded: time.Now(), value: resources}
	return resources, nil
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

func eventBusTagged() (bool, error) {
	return matchingResource(func(bus *events.EventBus) bool {
		return bus.Name != nil && *bus.Name == "helios-cold-chain" && len(bus.Tags) != 0
	})
}

func heliosFunctions() (map[string]*lambda.Function, error) {
	resources, err := listReadResources[*lambda.Function]()
	if err != nil {
		return nil, err
	}
	functions := map[string]*lambda.Function{}
	for identifier, function := range resources {
		if function.FunctionName != nil && strings.HasPrefix(*function.FunctionName, "helios-") {
			functions[identifier] = function
		}
	}
	return functions, nil
}

func allHeliosFunctions(match func(*lambda.Function) bool) (bool, error) {
	functions, err := heliosFunctions()
	if err != nil || len(functions) < 5 {
		return false, err
	}
	for _, function := range functions {
		if !match(function) {
			return false, nil
		}
	}
	return true, nil
}

func lambdaTagged() (bool, error) {
	return allHeliosFunctions(func(function *lambda.Function) bool { return len(function.Tags) != 0 })
}

func logRetention() (bool, error) {
	groups, err := listReadResources[*logs.LogGroup]()
	if err != nil {
		return false, err
	}
	count := 0
	for _, group := range groups {
		if group.LogGroupName == nil || !strings.HasPrefix(*group.LogGroupName, "/aws/lambda/helios-") {
			continue
		}
		count++
		if group.RetentionInDays == nil || *group.RetentionInDays <= 0 {
			return false, nil
		}
	}
	return count >= 5, nil
}

func dashboardConfigured() (bool, error) {
	return matchingResource(func(dashboard *cloudwatch.Dashboard) bool {
		return dashboard.DashboardName != nil && strings.HasPrefix(strings.ToLower(*dashboard.DashboardName), "helios") && dashboard.DashboardBody != nil
	})
}

func alarmConfigured() (bool, error) {
	return matchingResource(func(alarm *cloudwatch.Alarm) bool {
		return alarm.AlarmName != nil && strings.HasPrefix(strings.ToLower(*alarm.AlarmName), "helios") &&
			(alarm.ActionsEnabled == nil || *alarm.ActionsEnabled)
	})
}

func lambdaTracing() (bool, error) {
	return allHeliosFunctions(func(function *lambda.Function) bool {
		return function.TracingConfig != nil && function.TracingConfig.Mode != nil &&
			*function.TracingConfig.Mode == lambda.TracingConfigModeActive
	})
}

func namedQueue(name string) (*sqs.Queue, error) {
	queues, err := listReadResources[*sqs.Queue]()
	if err != nil {
		return nil, err
	}
	for _, queue := range queues {
		if queue.QueueName != nil && *queue.QueueName == name {
			return queue, nil
		}
	}
	return nil, nil
}

func queuesEncrypted() (bool, error) {
	for _, name := range []string{"helios-telemetry", "helios-telemetry-dlq"} {
		queue, err := namedQueue(name)
		if err != nil || queue == nil || queue.KmsMasterKeyId == nil || *queue.KmsMasterKeyId == "" {
			return false, err
		}
	}
	return true, nil
}

func topicEncrypted() (bool, error) {
	return matchingResource(func(topic *sns.Topic) bool {
		return topic.TopicName != nil && *topic.TopicName == "helios-excursions" &&
			topic.KmsMasterKeyId != nil && *topic.KmsMasterKeyId != ""
	})
}

func resultBucket(match func(*s3.Bucket) bool) (bool, error) {
	return matchingResource(func(bucket *s3.Bucket) bool {
		return bucket.BucketName != nil && strings.HasPrefix(*bucket.BucketName, "helios-results-") && match(bucket)
	})
}

func bucketEncrypted() (bool, error) {
	return resultBucket(func(bucket *s3.Bucket) bool {
		if bucket.BucketEncryption == nil {
			return false
		}
		for _, rule := range bucket.BucketEncryption.ServerSideEncryptionConfiguration {
			if rule.ServerSideEncryptionByDefault != nil && rule.ServerSideEncryptionByDefault.SSEAlgorithm != nil &&
				*rule.ServerSideEncryptionByDefault.SSEAlgorithm == s3.ServerSideEncryptionByDefaultSSEAlgorithmAwsKms {
				return true
			}
		}
		return false
	})
}

func temperatureSecret(match func(*secretsmanager.Secret) bool) (bool, error) {
	return matchingResource(func(secret *secretsmanager.Secret) bool {
		return secret.Name != nil && *secret.Name == "helios/temperature-limits" && match(secret)
	})
}

func secretEncrypted() (bool, error) {
	return temperatureSecret(func(secret *secretsmanager.Secret) bool {
		return secret.KmsKeyId != nil && *secret.KmsKeyId != ""
	})
}

func bucketProtected() (bool, error) {
	return resultBucket(func(bucket *s3.Bucket) bool {
		block := bucket.PublicAccessBlockConfiguration
		return block != nil && block.BlockPublicAcls != nil && *block.BlockPublicAcls &&
			block.BlockPublicPolicy != nil && *block.BlockPublicPolicy &&
			block.IgnorePublicAcls != nil && *block.IgnorePublicAcls &&
			block.RestrictPublicBuckets != nil && *block.RestrictPublicBuckets &&
			bucket.OwnershipControls != nil && len(bucket.OwnershipControls.Rules) != 0
	})
}

func secretTagged() (bool, error) {
	return temperatureSecret(func(secret *secretsmanager.Secret) bool { return len(secret.Tags) != 0 })
}

func heliosRule(match func(*events.Rule) bool) (bool, error) {
	rules, err := listMatchingResources(&events.Rule{EventBusName: new("helios-cold-chain")})
	if err != nil {
		return false, err
	}
	for _, rule := range rules {
		if match(rule) {
			return true, nil
		}
	}
	return false, nil
}

func eventRuleConfigured() (bool, error) {
	return heliosRule(func(rule *events.Rule) bool {
		if rule.State == nil || *rule.State != events.RuleStateENABLED {
			return false
		}
		for _, target := range rule.Targets {
			if target.Arn != nil && strings.Contains(*target.Arn, ":function:helios-relay") {
				return true
			}
		}
		return false
	})
}

func eventDeliveryReliable() (bool, error) {
	return heliosRule(func(rule *events.Rule) bool {
		for _, target := range rule.Targets {
			if target.Arn != nil && strings.Contains(*target.Arn, ":function:helios-relay") &&
				target.DeadLetterConfig != nil && target.DeadLetterConfig.Arn != nil &&
				target.RetryPolicy != nil && target.RetryPolicy.MaximumRetryAttempts != nil &&
				*target.RetryPolicy.MaximumRetryAttempts > 0 {
				return true
			}
		}
		return false
	})
}

func queueRedrive() (bool, error) {
	queue, err := namedQueue("helios-telemetry")
	return queue != nil && len(queue.RedrivePolicy) != 0, err
}

func deadLetterRetention() (bool, error) {
	queue, err := namedQueue("helios-telemetry-dlq")
	return queue != nil && queue.MessageRetentionPeriod != nil && *queue.MessageRetentionPeriod >= 7*24*60*60, err
}

func queueConsumerConfigured() (bool, error) {
	return matchingResource(func(mapping *lambda.EventSourceMapping) bool {
		if mapping.Enabled == nil || !*mapping.Enabled || mapping.EventSourceArn == nil ||
			!strings.Contains(*mapping.EventSourceArn, ":helios-telemetry") || mapping.BatchSize == nil ||
			*mapping.BatchSize < 2 || *mapping.BatchSize > 10 {
			return false
		}
		for _, response := range mapping.FunctionResponseTypes {
			if response == lambda.EventSourceMappingFunctionResponseTypesItemReportBatchItemFailures {
				return true
			}
		}
		return false
	})
}

func heliosWorkflow(match func(*stepfunctions.StateMachine) bool) (bool, error) {
	return matchingResource(func(machine *stepfunctions.StateMachine) bool {
		name := machine.StateMachineName
		if name == nil {
			name = machine.Name
		}
		return name != nil && strings.HasPrefix(strings.ToLower(*name), "helios") && match(machine)
	})
}

func expressWorkflow() (bool, error) {
	return heliosWorkflow(func(machine *stepfunctions.StateMachine) bool {
		return machine.StateMachineType != nil && *machine.StateMachineType == stepfunctions.StateMachineStateMachineTypeEXPRESS
	})
}

func workflowLogging() (bool, error) {
	return heliosWorkflow(func(machine *stepfunctions.StateMachine) bool {
		return machine.LoggingConfiguration != nil && len(machine.LoggingConfiguration.Destinations) != 0 &&
			machine.LoggingConfiguration.Level != nil && !strings.EqualFold(string(*machine.LoggingConfiguration.Level), "OFF")
	})
}

func workflowTracing() (bool, error) {
	return heliosWorkflow(func(machine *stepfunctions.StateMachine) bool {
		return machine.TracingConfiguration != nil && machine.TracingConfiguration.Enabled != nil &&
			*machine.TracingConfiguration.Enabled
	})
}

func bucketVersioned() (bool, error) {
	return resultBucket(func(bucket *s3.Bucket) bool {
		return bucket.VersioningConfiguration != nil && bucket.VersioningConfiguration.Status != nil &&
			*bucket.VersioningConfiguration.Status == s3.VersioningConfigurationStatusEnabled
	})
}

func archiveConfigured() (bool, error) {
	return matchingResource(func(archive *events.Archive) bool {
		return archive.SourceArn != nil && strings.HasSuffix(*archive.SourceArn, "/helios-cold-chain") &&
			archive.RetentionDays != nil && *archive.RetentionDays > 0 && *archive.RetentionDays <= 7
	})
}

func topicTagged() (bool, error) {
	return matchingResource(func(topic *sns.Topic) bool {
		return topic.TopicName != nil && *topic.TopicName == "helios-excursions" && len(topic.Tags) != 0
	})
}

func queueLongPolling() (bool, error) {
	queue, err := namedQueue("helios-telemetry")
	return queue != nil && queue.ReceiveMessageWaitTimeSeconds != nil && *queue.ReceiveMessageWaitTimeSeconds > 0, err
}

func queueVisibility() (bool, error) {
	queue, err := namedQueue("helios-telemetry")
	return queue != nil && queue.VisibilityTimeout != nil && *queue.VisibilityTimeout >= 60, err
}

func lambdaArm64() (bool, error) {
	return allHeliosFunctions(func(function *lambda.Function) bool {
		return len(function.Architectures) == 1 && function.Architectures[0] == lambda.FunctionArchitecturesItemArm64
	})
}

func lambdaTimeout() (bool, error) {
	return allHeliosFunctions(func(function *lambda.Function) bool {
		return function.Timeout != nil && *function.Timeout > 0 && *function.Timeout <= 30
	})
}

func bucketLifecycle() (bool, error) {
	return resultBucket(func(bucket *s3.Bucket) bool {
		if bucket.LifecycleConfiguration == nil {
			return false
		}
		for _, rule := range bucket.LifecycleConfiguration.Rules {
			if rule.Status != nil && *rule.Status == s3.RuleStatusEnabled {
				return true
			}
		}
		return false
	})
}

func bucketIntelligentTiering() (bool, error) {
	return resultBucket(func(bucket *s3.Bucket) bool {
		for _, configuration := range bucket.IntelligentTieringConfigurations {
			if configuration.Status != nil && *configuration.Status == s3.IntelligentTieringConfigurationStatusEnabled {
				return true
			}
		}
		return false
	})
}

func lambdaMemory() (bool, error) {
	return allHeliosFunctions(func(function *lambda.Function) bool {
		return function.MemorySize != nil && *function.MemorySize <= 512
	})
}

func architectureComplete() (bool, error) {
	checks := []trigger{
		func() (bool, error) { return eventBusTagged() },
		func() (bool, error) { return eventRuleConfigured() },
		func() (bool, error) {
			queue, err := namedQueue("helios-telemetry")
			return queue != nil, err
		},
		func() (bool, error) {
			functions, err := heliosFunctions()
			return len(functions) >= 5, err
		},
		func() (bool, error) { return expressWorkflow() },
		func() (bool, error) { return resultBucket(func(*s3.Bucket) bool { return true }) },
		func() (bool, error) { return topicTagged() },
		func() (bool, error) { return temperatureSecret(func(*secretsmanager.Secret) bool { return true }) },
	}
	for _, check := range checks {
		passed, err := check()
		if err != nil || !passed {
			return false, err
		}
	}

	return heliosWorkflow(func(machine *stepfunctions.StateMachine) bool {
		definition := machine.DefinitionString
		if definition == nil && len(machine.Definition) != 0 {
			encoded, _ := json.Marshal(machine.Definition)
			text := string(encoded)
			definition = &text
		}
		return definition != nil && strings.Contains(strings.ToLower(*definition), "lambda")
	})
}
