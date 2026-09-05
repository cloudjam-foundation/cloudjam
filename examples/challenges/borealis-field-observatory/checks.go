//go:build wasip1

package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"codeberg.org/megakuul/cloudjam/pkg/challenge"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/appconfig"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/apprunner"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/athena"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/cloudwatch"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/ecr"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/glue"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/iot"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/kinesisfirehose"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/logs"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/s3"
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
	addDesign(s, "Container repository is hardened", 10, "borealis-api is missing immutable tags, scan-on-push, encryption, or challenge tags.", repositoryHardened)
	addDesign(s, "Container images expire", 10, "borealis-api has no ECR lifecycle policy.", repositoryLifecycle)
	addDesign(s, "App Runner is publicly accessible", 10, "No public Borealis App Runner service exists.", servicePublic)
	addDesign(s, "App Runner uses the competition image", 10, "The service does not use the private borealis-api:competition image with automatic deployments disabled.", serviceImage)
	addDesign(s, "App Runner uses the supplied roles", 10, "The ECR access role or App Runner instance role is not configured.", serviceRoles)
	addDesign(s, "App Runner checks application health", 10, "The service does not use an HTTP /health check.", serviceHealth)
	addDesign(s, "Container capacity is cost-conscious", 10, "The App Runner service exceeds 1 vCPU or 2 GB memory.", serviceCapacity)
	addDesign(s, "Container environment is complete", 10, "One or more required Borealis environment variables is absent.", serviceEnvironment)

	addDesign(s, "IoT rule selects observations", 10, "No enabled tagged IoT rule selects borealis/observations with SQL version 2016-03-23.", topicRuleConfigured)
	addDesign(s, "IoT rule delivers newline JSON", 10, "The IoT rule has no Firehose action, role, or newline separator.", topicRuleDelivery)
	addDesign(s, "IoT rule errors are retained", 10, "The IoT rule has no CloudWatch Logs error action backed by a finite-retention group.", topicRuleErrors)
	addDesign(s, "Firehose accepts direct writes", 10, "borealis-observations is not a tagged direct-put Firehose stream.", firehoseDirectPut)
	addDesign(s, "Firehose writes the lake prefixes", 10, "Firehose does not use the Borealis lake with distinct observation and error prefixes.", firehoseDestination)
	addDesign(s, "Firehose latency is bounded", 10, "The S3 delivery buffer interval is greater than 60 seconds.", firehoseBuffering)
	addDesign(s, "Firehose delivery is observable", 10, "Firehose S3 delivery logging is not enabled.", firehoseLogging)

	addDesign(s, "Lake data is encrypted", 10, "The Borealis lake has no default server-side encryption.", lakeEncrypted)
	addDesign(s, "Lake access is protected", 10, "The Borealis lake does not block public access and enforce bucket-owner ownership.", lakeProtected)
	addDesign(s, "Lake history is managed", 10, "The Borealis lake is not versioned or has no enabled lifecycle rule.", lakeHistory)
	addDesign(s, "Glue catalog is present", 10, "The borealis_observatory database or observations table is missing.", glueCatalog)
	addDesign(s, "Glue schema drives Parquet conversion", 10, "Firehose does not convert observation JSON to Parquet with the Glue table schema.", glueSchema)

	addDesign(s, "Athena workgroup is enforced", 10, "borealis-analysts is not enabled with enforced configuration.", athenaEnforced)
	addDesign(s, "Athena results are protected", 10, "The workgroup lacks an encrypted Borealis S3 result location or query metrics.", athenaResults)
	addDesign(s, "Athena query cost is capped", 10, "The workgroup has no positive bytes-scanned cutoff.", athenaCost)
	addDesign(s, "AppConfig hierarchy is tagged", 10, "The Borealis AppConfig application, environment, or profile is missing or untagged.", appConfigHierarchy)
	addDesign(s, "AppConfig validates limits", 10, "habitat-limits is not a hosted profile with a JSON Schema validator.", appConfigValidator)
	addDesign(s, "AppConfig rollout has a bake period", 10, "No reusable Borealis deployment strategy has a final bake period.", appConfigStrategy)
	addDesign(s, "AppConfig limits are deployed", 10, "No Borealis configuration deployment completed.", appConfigDeployment)

	addDesign(s, "Operations dashboard is configured", 10, "No Borealis CloudWatch dashboard exists.", dashboardConfigured)
	addDesign(s, "Failure alarm is configured", 10, "No enabled Borealis CloudWatch alarm exists.", alarmConfigured)
	addDesign(s, "End-to-end architecture is complete", 30, "One or more required container, configuration, IoT, delivery, catalog, or query layers is missing.", architectureComplete)
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

func borealisRepository() (*ecr.Repository, error) {
	resources, err := listReadResources[*ecr.Repository]()
	if err != nil {
		return nil, err
	}
	for _, repository := range resources {
		if repository.RepositoryName != nil && *repository.RepositoryName == "borealis-api" {
			return repository, nil
		}
	}
	return nil, nil
}

func repositoryHardened() (bool, error) {
	repository, err := borealisRepository()
	if err != nil || repository == nil {
		return false, err
	}
	return repository.ImageTagMutability != nil && *repository.ImageTagMutability == ecr.RepositoryImageTagMutabilityIMMUTABLE &&
		repository.ImageScanningConfiguration != nil && repository.ImageScanningConfiguration.ScanOnPush != nil && *repository.ImageScanningConfiguration.ScanOnPush &&
		repository.EncryptionConfiguration != nil && repository.EncryptionConfiguration.EncryptionType != nil && len(repository.Tags) != 0, nil
}

func repositoryLifecycle() (bool, error) {
	repository, err := borealisRepository()
	return repository != nil && repository.LifecyclePolicy != nil && repository.LifecyclePolicy.LifecyclePolicyText != nil, err
}

func borealisService() (*apprunner.Service, error) {
	resources, err := listReadResources[*apprunner.Service]()
	if err != nil {
		return nil, err
	}
	for _, service := range resources {
		if service.ServiceName != nil && strings.HasPrefix(*service.ServiceName, "borealis") {
			return service, nil
		}
	}
	return nil, nil
}

func servicePublic() (bool, error) {
	service, err := borealisService()
	if err != nil || service == nil {
		return false, err
	}
	public := service.NetworkConfiguration == nil || service.NetworkConfiguration.IngressConfiguration == nil ||
		service.NetworkConfiguration.IngressConfiguration.IsPubliclyAccessible == nil || *service.NetworkConfiguration.IngressConfiguration.IsPubliclyAccessible
	return public, nil
}

func serviceImage() (bool, error) {
	service, err := borealisService()
	if err != nil || service == nil || service.SourceConfiguration == nil || service.SourceConfiguration.ImageRepository == nil {
		return false, err
	}
	image := service.SourceConfiguration.ImageRepository
	return image.ImageIdentifier != nil && strings.Contains(*image.ImageIdentifier, "/borealis-api:competition") &&
		image.ImageRepositoryType != nil && *image.ImageRepositoryType == apprunner.ImageRepositoryImageRepositoryTypeECR &&
		service.SourceConfiguration.AutoDeploymentsEnabled != nil && !*service.SourceConfiguration.AutoDeploymentsEnabled, nil
}

func serviceRoles() (bool, error) {
	service, err := borealisService()
	if err != nil || service == nil || service.SourceConfiguration == nil || service.SourceConfiguration.AuthenticationConfiguration == nil || service.InstanceConfiguration == nil {
		return false, err
	}
	authentication := service.SourceConfiguration.AuthenticationConfiguration
	return authentication.AccessRoleArn != nil && strings.HasSuffix(*authentication.AccessRoleArn, "/borealis-apprunner-access-role") &&
		service.InstanceConfiguration.InstanceRoleArn != nil && strings.HasSuffix(*service.InstanceConfiguration.InstanceRoleArn, "/borealis-apprunner-instance-role"), nil
}

func serviceHealth() (bool, error) {
	service, err := borealisService()
	if err != nil || service == nil || service.HealthCheckConfiguration == nil {
		return false, err
	}
	health := service.HealthCheckConfiguration
	return health.Protocol != nil && *health.Protocol == apprunner.HealthCheckConfigurationProtocolHTTP && health.Path != nil && *health.Path == "/health", nil
}

func serviceCapacity() (bool, error) {
	service, err := borealisService()
	if err != nil || service == nil || service.InstanceConfiguration == nil {
		return false, err
	}
	cpu, cpuErr := capacity(value(service.InstanceConfiguration.Cpu), "vCPU", 1024)
	memory, memoryErr := capacity(value(service.InstanceConfiguration.Memory), "GB", 1024)
	if cpuErr != nil || memoryErr != nil {
		return false, nil
	}
	return cpu <= 1024 && memory <= 2048, nil
}

func capacity(configured, unit string, multiplier float64) (float64, error) {
	configured = strings.TrimSpace(configured)
	if strings.HasSuffix(configured, unit) {
		amount := strings.TrimSpace(strings.TrimSuffix(configured, unit))
		parsed, err := strconv.ParseFloat(amount, 64)
		return parsed * multiplier, err
	}
	configured = strings.TrimSpace(strings.TrimSuffix(configured, "MB"))
	return strconv.ParseFloat(configured, 64)
}

func serviceEnvironment() (bool, error) {
	service, err := borealisService()
	if err != nil || service == nil || service.SourceConfiguration == nil || service.SourceConfiguration.ImageRepository == nil || service.SourceConfiguration.ImageRepository.ImageConfiguration == nil {
		return false, err
	}
	configured := map[string]bool{}
	for _, variable := range service.SourceConfiguration.ImageRepository.ImageConfiguration.RuntimeEnvironmentVariables {
		if variable.Name != nil && variable.Value != nil && *variable.Value != "" {
			configured[*variable.Name] = true
		}
	}
	for _, name := range []string{"IOT_ENDPOINT", "IOT_TOPIC", "APPCONFIG_APPLICATION", "APPCONFIG_ENVIRONMENT", "APPCONFIG_PROFILE", "ATHENA_DATABASE", "ATHENA_TABLE", "ATHENA_WORKGROUP"} {
		if !configured[name] {
			return false, nil
		}
	}
	return true, nil
}

func borealisTopicRule() (*iot.TopicRule, error) {
	resources, err := listReadResources[*iot.TopicRule]()
	if err != nil {
		return nil, err
	}
	for _, rule := range resources {
		if rule.RuleName != nil && strings.HasPrefix(*rule.RuleName, "borealis") {
			return rule, nil
		}
	}
	return nil, nil
}

func topicRuleConfigured() (bool, error) {
	rule, err := borealisTopicRule()
	if err != nil || rule == nil || rule.TopicRulePayload == nil {
		return false, err
	}
	payload := rule.TopicRulePayload
	return len(rule.Tags) != 0 && payload.RuleDisabled != nil && !*payload.RuleDisabled && payload.AwsIotSqlVersion != nil && *payload.AwsIotSqlVersion == "2016-03-23" &&
		payload.Sql != nil && strings.Contains(strings.ToLower(*payload.Sql), "borealis/observations"), nil
}

func topicRuleDelivery() (bool, error) {
	rule, err := borealisTopicRule()
	if err != nil || rule == nil || rule.TopicRulePayload == nil {
		return false, err
	}
	for _, action := range rule.TopicRulePayload.Actions {
		if action.Firehose != nil && action.Firehose.DeliveryStreamName != nil && *action.Firehose.DeliveryStreamName == "borealis-observations" &&
			action.Firehose.RoleArn != nil && strings.HasSuffix(*action.Firehose.RoleArn, "/borealis-iot-rule-role") &&
			action.Firehose.Separator != nil && strings.Contains(*action.Firehose.Separator, "\n") {
			return true, nil
		}
	}
	return false, nil
}

func topicRuleErrors() (bool, error) {
	rule, err := borealisTopicRule()
	if err != nil || rule == nil || rule.TopicRulePayload == nil || rule.TopicRulePayload.ErrorAction == nil || rule.TopicRulePayload.ErrorAction.CloudwatchLogs == nil {
		return false, err
	}
	action := rule.TopicRulePayload.ErrorAction.CloudwatchLogs
	if action.LogGroupName == nil || action.RoleArn == nil || !strings.HasSuffix(*action.RoleArn, "/borealis-iot-rule-role") {
		return false, nil
	}
	groups, err := listReadResources[*logs.LogGroup]()
	if err != nil {
		return false, err
	}
	for _, group := range groups {
		if group.LogGroupName != nil && *group.LogGroupName == *action.LogGroupName && group.RetentionInDays != nil && *group.RetentionInDays > 0 {
			return true, nil
		}
	}
	return false, nil
}

func borealisFirehose() (*kinesisfirehose.DeliveryStream, error) {
	resources, err := listReadResources[*kinesisfirehose.DeliveryStream]()
	if err != nil {
		return nil, err
	}
	for _, stream := range resources {
		if stream.DeliveryStreamName != nil && *stream.DeliveryStreamName == "borealis-observations" {
			return stream, nil
		}
	}
	return nil, nil
}

func firehoseDirectPut() (bool, error) {
	stream, err := borealisFirehose()
	return stream != nil && len(stream.Tags) != 0 && stream.DeliveryStreamType != nil && *stream.DeliveryStreamType == kinesisfirehose.DeliveryStreamDeliveryStreamTypeDirectPut, err
}

func firehoseDestination() (bool, error) {
	stream, err := borealisFirehose()
	if err != nil || stream == nil || stream.ExtendedS3DestinationConfiguration == nil {
		return false, err
	}
	destination := stream.ExtendedS3DestinationConfiguration
	return destination.BucketARN != nil && strings.Contains(*destination.BucketARN, ":borealis-lake-") &&
		destination.RoleARN != nil && strings.HasSuffix(*destination.RoleARN, "/borealis-firehose-role") &&
		destination.Prefix != nil && strings.HasPrefix(*destination.Prefix, "observations/") &&
		destination.ErrorOutputPrefix != nil && *destination.ErrorOutputPrefix != "" && *destination.ErrorOutputPrefix != *destination.Prefix, nil
}

func firehoseBuffering() (bool, error) {
	stream, err := borealisFirehose()
	if err != nil || stream == nil || stream.ExtendedS3DestinationConfiguration == nil || stream.ExtendedS3DestinationConfiguration.BufferingHints == nil {
		return false, err
	}
	interval := stream.ExtendedS3DestinationConfiguration.BufferingHints.IntervalInSeconds
	return interval != nil && *interval >= 0 && *interval <= 60, nil
}

func firehoseLogging() (bool, error) {
	stream, err := borealisFirehose()
	if err != nil || stream == nil || stream.ExtendedS3DestinationConfiguration == nil || stream.ExtendedS3DestinationConfiguration.CloudWatchLoggingOptions == nil {
		return false, err
	}
	logging := stream.ExtendedS3DestinationConfiguration.CloudWatchLoggingOptions
	return logging.Enabled != nil && *logging.Enabled && logging.LogGroupName != nil && logging.LogStreamName != nil, nil
}

func borealisLake(match func(*s3.Bucket) bool) (bool, error) {
	return matchingResource(func(bucket *s3.Bucket) bool {
		return bucket.BucketName != nil && strings.HasPrefix(*bucket.BucketName, "borealis-lake-") && match(bucket)
	})
}

func lakeEncrypted() (bool, error) {
	return borealisLake(func(bucket *s3.Bucket) bool {
		return bucket.BucketEncryption != nil && len(bucket.BucketEncryption.ServerSideEncryptionConfiguration) != 0
	})
}

func lakeProtected() (bool, error) {
	return borealisLake(func(bucket *s3.Bucket) bool {
		block := bucket.PublicAccessBlockConfiguration
		owned := bucket.OwnershipControls
		return block != nil && block.BlockPublicAcls != nil && *block.BlockPublicAcls && block.BlockPublicPolicy != nil && *block.BlockPublicPolicy &&
			block.IgnorePublicAcls != nil && *block.IgnorePublicAcls && block.RestrictPublicBuckets != nil && *block.RestrictPublicBuckets &&
			owned != nil && len(owned.Rules) != 0 && owned.Rules[0].ObjectOwnership != nil && *owned.Rules[0].ObjectOwnership == s3.OwnershipControlsRuleObjectOwnershipBucketOwnerEnforced
	})
}

func lakeHistory() (bool, error) {
	return borealisLake(func(bucket *s3.Bucket) bool {
		if bucket.VersioningConfiguration == nil || bucket.VersioningConfiguration.Status == nil || *bucket.VersioningConfiguration.Status != s3.VersioningConfigurationStatusEnabled || bucket.LifecycleConfiguration == nil {
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

func borealisDatabase() (*glue.Database, error) {
	resources, err := listReadResources[*glue.Database]()
	if err != nil {
		return nil, err
	}
	for _, database := range resources {
		name := database.DatabaseName
		if name == nil && database.DatabaseInput != nil {
			name = database.DatabaseInput.Name
		}
		if name != nil && *name == "borealis_observatory" {
			return database, nil
		}
	}
	return nil, nil
}

func glueCatalog() (bool, error) {
	database, err := borealisDatabase()
	return database != nil, err
}

func glueSchema() (bool, error) {
	stream, err := borealisFirehose()
	if err != nil || stream == nil || stream.ExtendedS3DestinationConfiguration == nil || stream.ExtendedS3DestinationConfiguration.DataFormatConversionConfiguration == nil {
		return false, err
	}
	conversion := stream.ExtendedS3DestinationConfiguration.DataFormatConversionConfiguration
	return conversion.Enabled != nil && *conversion.Enabled && conversion.InputFormatConfiguration != nil &&
		conversion.InputFormatConfiguration.Deserializer != nil && conversion.InputFormatConfiguration.Deserializer.OpenXJsonSerDe != nil &&
		conversion.OutputFormatConfiguration != nil && conversion.OutputFormatConfiguration.Serializer != nil && conversion.OutputFormatConfiguration.Serializer.ParquetSerDe != nil &&
		conversion.SchemaConfiguration != nil && conversion.SchemaConfiguration.DatabaseName != nil && *conversion.SchemaConfiguration.DatabaseName == "borealis_observatory" &&
		conversion.SchemaConfiguration.TableName != nil && *conversion.SchemaConfiguration.TableName == "observations", nil
}

func borealisWorkgroup() (*athena.WorkGroup, error) {
	resources, err := listReadResources[*athena.WorkGroup]()
	if err != nil {
		return nil, err
	}
	for _, workgroup := range resources {
		if workgroup.Name != nil && *workgroup.Name == "borealis-analysts" {
			return workgroup, nil
		}
	}
	return nil, nil
}

func athenaEnforced() (bool, error) {
	workgroup, err := borealisWorkgroup()
	return workgroup != nil && workgroup.State != nil && *workgroup.State == athena.WorkGroupStateENABLED && workgroup.WorkGroupConfiguration != nil &&
		workgroup.WorkGroupConfiguration.EnforceWorkGroupConfiguration != nil && *workgroup.WorkGroupConfiguration.EnforceWorkGroupConfiguration, err
}

func athenaResults() (bool, error) {
	workgroup, err := borealisWorkgroup()
	if err != nil || workgroup == nil || workgroup.WorkGroupConfiguration == nil || workgroup.WorkGroupConfiguration.ResultConfiguration == nil {
		return false, err
	}
	configuration := workgroup.WorkGroupConfiguration
	result := configuration.ResultConfiguration
	return configuration.PublishCloudWatchMetricsEnabled != nil && *configuration.PublishCloudWatchMetricsEnabled &&
		result.OutputLocation != nil && strings.Contains(*result.OutputLocation, "borealis-lake-") &&
		result.EncryptionConfiguration != nil && result.EncryptionConfiguration.EncryptionOption != nil, nil
}

func athenaCost() (bool, error) {
	workgroup, err := borealisWorkgroup()
	return workgroup != nil && workgroup.WorkGroupConfiguration != nil && workgroup.WorkGroupConfiguration.BytesScannedCutoffPerQuery != nil && *workgroup.WorkGroupConfiguration.BytesScannedCutoffPerQuery > 0, err
}

func appConfigHierarchy() (bool, error) {
	applicationID, environmentID, err := borealisAppConfig()
	if err != nil {
		return false, err
	}
	if applicationID == "" || environmentID == "" {
		return false, nil
	}
	profiles, err := listMatchingResources(&appconfig.ConfigurationProfile{ApplicationId: &applicationID})
	if err != nil {
		return false, err
	}
	profileFound := false
	for _, profile := range profiles {
		if profile.ApplicationId != nil && *profile.ApplicationId == applicationID && profile.Name != nil && *profile.Name == "habitat-limits" && len(profile.Tags) != 0 {
			profileFound = true
		}
	}
	return profileFound, nil
}

func appConfigValidator() (bool, error) {
	applicationID, _, err := borealisAppConfig()
	if err != nil || applicationID == "" {
		return false, err
	}
	profiles, err := listMatchingResources(&appconfig.ConfigurationProfile{ApplicationId: &applicationID})
	if err != nil {
		return false, err
	}
	for _, profile := range profiles {
		if profile.Name == nil || *profile.Name != "habitat-limits" || profile.LocationUri == nil || *profile.LocationUri != "hosted" {
			continue
		}
		for _, validator := range profile.Validators {
			if validator.Type != nil && strings.EqualFold(*validator.Type, "JSON_SCHEMA") && validator.Content != nil && *validator.Content != "" {
				return true, nil
			}
		}
	}
	return false, nil
}

func appConfigStrategy() (bool, error) {
	return matchingResource(func(strategy *appconfig.DeploymentStrategy) bool {
		return strategy.Name != nil && strings.HasPrefix(strings.ToLower(*strategy.Name), "borealis") && len(strategy.Tags) != 0 &&
			strategy.FinalBakeTimeInMinutes != nil && *strategy.FinalBakeTimeInMinutes > 0
	})
}

func appConfigDeployment() (bool, error) {
	applicationID, environmentID, err := borealisAppConfig()
	if err != nil || applicationID == "" || environmentID == "" {
		return false, err
	}
	deployments, err := listMatchingResources(&appconfig.Deployment{ApplicationId: &applicationID, EnvironmentId: &environmentID})
	if err != nil {
		return false, err
	}
	for _, deployment := range deployments {
		if len(deployment.Tags) != 0 && deployment.State != nil && strings.EqualFold(string(*deployment.State), "COMPLETE") {
			return true, nil
		}
	}
	return false, nil
}

func borealisAppConfig() (string, string, error) {
	applications, err := listReadResources[*appconfig.Application]()
	if err != nil {
		return "", "", err
	}
	applicationID := ""
	for _, application := range applications {
		if application.Name != nil && *application.Name == "borealis-observatory" && len(application.Tags) != 0 && application.ApplicationId != nil {
			applicationID = *application.ApplicationId
		}
	}
	if applicationID == "" {
		return "", "", nil
	}
	environments, err := listMatchingResources(&appconfig.Environment{ApplicationId: &applicationID})
	if err != nil {
		return "", "", err
	}
	for _, environment := range environments {
		if environment.Name != nil && *environment.Name == "production" && len(environment.Tags) != 0 && environment.EnvironmentId != nil {
			return applicationID, *environment.EnvironmentId, nil
		}
	}
	return applicationID, "", nil
}

func dashboardConfigured() (bool, error) {
	return matchingResource(func(dashboard *cloudwatch.Dashboard) bool {
		return dashboard.DashboardName != nil && strings.HasPrefix(strings.ToLower(*dashboard.DashboardName), "borealis") && dashboard.DashboardBody != nil
	})
}

func alarmConfigured() (bool, error) {
	return matchingResource(func(alarm *cloudwatch.Alarm) bool {
		return alarm.AlarmName != nil && strings.HasPrefix(strings.ToLower(*alarm.AlarmName), "borealis") && (alarm.ActionsEnabled == nil || *alarm.ActionsEnabled)
	})
}

func architectureComplete() (bool, error) {
	checks := []trigger{repositoryHardened, serviceImage, serviceRoles, serviceEnvironment, topicRuleDelivery, firehoseDestination, glueSchema, athenaEnforced, appConfigDeployment}
	for _, check := range checks {
		passed, err := check()
		if err != nil || !passed {
			return false, err
		}
	}
	return true, nil
}

func value(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
