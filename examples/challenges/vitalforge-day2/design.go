//go:build wasip1

package main

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"codeberg.org/megakuul/cloudjam/examples/challenges/vitalforge-day2/internal/scoring"
	"codeberg.org/megakuul/cloudjam/pkg/challenge"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/api"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/policy"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/lambda"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/ssm"
	"github.com/google/uuid"
)

//go:embed reference/design-marking.json
var designMarking []byte

//go:embed assessor.py
var assessorCode string

type designCriterion struct {
	ID     string  `json:"id"`
	Aspect string  `json:"aspect"`
	Name   string  `json:"name"`
	Points float64 `json:"points"`
}

type designEvidence struct {
	Endpoint string                    `json:"endpoint"`
	Checks   map[string]scoring.Result `json:"checks"`
	Probe    json.RawMessage           `json:"probe,omitempty"`
	Active   bool                      `json:"active"`
}

type designEvaluator struct {
	lock     sync.Mutex
	evidence designEvidence
}

type assessorRole struct {
	RoleName                 string          `json:"RoleName"`
	Arn                      string          `json:"Arn,omitempty"`
	AssumeRolePolicyDocument policy.Document `json:"AssumeRolePolicyDocument"`
	Policies                 []struct {
		PolicyName     string          `json:"PolicyName"`
		PolicyDocument policy.Document `json:"PolicyDocument"`
	} `json:"Policies"`
}

func (*assessorRole) CloudJamType() string { return "AWS::IAM::Role" }

func addDesignChecks(s *challenge.Scenario, started time.Time) (*designEvaluator, error) {
	var criteria []designCriterion
	if err := json.Unmarshal(designMarking, &criteria); err != nil {
		return nil, err
	}
	for _, criterion := range criteria {
		if _, err := api.RegisterScore(api.RegisterScoreInput{
			Name: criterion.ID + " " + criterion.Name, Type: api.ScoreTypeDesign,
			Maximum: criterion.Points * 1550,
		}); err != nil {
			return nil, err
		}
	}
	name := "cloudjam-vitalforge-assessor-" + uuid.NewString()[:8]
	token := uuid.NewString() + uuid.NewString()
	role := &assessorRole{RoleName: name, AssumeRolePolicyDocument: policy.Document{
		Version:   policy.Version20121017,
		Statement: []policy.Statement{{Effect: policy.Allow, Principal: policy.PrincipalService("lambda.amazonaws.com"), Action: policy.Actions{"sts:AssumeRole"}}},
	}}
	role.Policies = append(role.Policies, struct {
		PolicyName     string          `json:"PolicyName"`
		PolicyDocument policy.Document `json:"PolicyDocument"`
	}{PolicyName: "assessment", PolicyDocument: policy.Document{
		Version: policy.Version20121017,
		Statement: []policy.Statement{{Effect: policy.Allow, Resource: policy.ARNAll, Action: policy.Actions{
			"ec2:Describe*", "rds:Describe*", "eks:List*", "eks:Describe*", "eks:AccessKubernetesApi",
			"elasticfilesystem:Describe*", "dynamodb:ListTables", "dynamodb:DescribeTable", "dynamodb:Scan", "dynamodb:GetItem",
			"ssm:GetParameter", "ssm:GetParametersByPath", "cloudtrail:DescribeTrails", "cloudtrail:GetTrailStatus", "cloudtrail:GetEventSelectors",
			"scheduler:ListSchedules", "scheduler:GetSchedule", "lambda:GetFunctionConcurrency", "lambda:GetAccountSettings", "elasticloadbalancing:Describe*",
			"cloudfront:List*", "cloudfront:Get*",
			"s3:ListBucket", "s3:GetBucket*", "s3:GetLifecycleConfiguration", "s3:GetObject",
			"lambda:ListFunctions",
			"apigateway:GET",
			"cognito-idp:ListUserPools", "cognito-idp:ListGroups",
			"events:List*", "events:DescribeRule",
			"codepipeline:ListPipelines", "codepipeline:GetPipeline", "codepipeline:GetPipelineState",
			"codebuild:BatchGetProjects", "iam:GetRolePolicy", "iam:ListRolePolicies", "iam:ListAttachedRolePolicies",
			"iam:GetPolicy", "iam:GetPolicyVersion",
			"route53:ListHostedZones", "route53:GetHostedZone", "route53:ListResourceRecordSets",
			"vpc-lattice:List*", "vpc-lattice:Get*", "logs:DescribeLogGroups",
		}}, {
			Effect: policy.Allow, Resource: policy.ARNsFrom("arn:aws:eks:*:*:cluster/*"),
			Action:    policy.Actions{"eks:CreateAccessEntry"},
			Condition: policy.Conditions{policy.ArnLike("eks:principalArn", "arn:aws:iam::*:role/"+name)},
		}, {
			Effect: policy.Allow, Resource: policy.ARNsFrom("arn:aws:eks:*:*:access-entry/*/role/*/" + name + "/*"),
			Action:    policy.Actions{"eks:AssociateAccessPolicy"},
			Condition: policy.Conditions{policy.ArnEquals("eks:policyArn", "arn:aws:eks::aws:cluster-access-policy/AmazonEKSAdminViewPolicy")},
		}, {
			Effect: policy.Allow, Resource: policy.ARNsFrom("arn:aws:logs:*:*:log-group:/aws/lambda/" + name + ":*"),
			Action: policy.Actions{"logs:CreateLogGroup", "logs:CreateLogStream", "logs:PutLogEvents"},
		}},
	}})
	roleID, err := aws.Create(role)
	if err != nil {
		return nil, err
	}
	role, err = aws.Read[*assessorRole](roleID)
	if err != nil {
		return nil, err
	}
	exporter, _ := files.ReadFile("game-assets/daily-export.zip")
	exporterHash := sha256.Sum256(exporter)
	sleeper, _ := files.ReadFile("game-assets/sleep-ingest.zip")
	sleeperHash := sha256.Sum256(sleeper)
	functionID, err := aws.Create(&lambda.Function{
		FunctionName: &name, Role: &role.Arn, Runtime: new("python3.13"), Handler: new("index.handler"),
		Code: &lambda.Code{ZipFile: &assessorCode}, MemorySize: new(1024), Timeout: new(30),
		Environment: &lambda.Environment{Variables: map[string]string{
			"ASSESSOR_ROLE": role.Arn, "ASSESSOR_TOKEN": token, "EXPORT_SHA256": base64.StdEncoding.EncodeToString(exporterHash[:]),
			"SLEEP_SHA256": base64.StdEncoding.EncodeToString(sleeperHash[:]),
		}},
	})
	if err != nil {
		return nil, err
	}
	urlID, err := aws.Create(&lambda.Url{TargetFunctionArn: &functionID, AuthType: new(lambda.UrlAuthTypeNONE)})
	if err != nil {
		return nil, err
	}
	for _, permission := range []*lambda.Permission{
		{FunctionName: &functionID, Principal: new("*"), Action: new("lambda:InvokeFunctionUrl"), FunctionUrlAuthType: new(lambda.PermissionFunctionUrlAuthTypeNONE)},
		{FunctionName: &functionID, Principal: new("*"), Action: new("lambda:InvokeFunction"), InvokedViaFunctionUrl: new(true)},
	} {
		if _, err := aws.Create(permission); err != nil {
			return nil, err
		}
	}
	url, err := aws.Read[*lambda.Url](urlID)
	if err != nil || url.FunctionUrl == nil {
		return nil, fmt.Errorf("read assessor URL: %v", err)
	}
	evaluator := &designEvaluator{evidence: designEvidence{
		Checks: map[string]scoring.Result{},
	}}
	s.AddEvent("VitalForge architectural marking", challenge.Event{
		Trigger: func() (bool, error) { return true, nil },
		Event: func(ctx context.Context, s *challenge.Scenario) error {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for time.Since(started) < 6*time.Hour {
				endpoint := ""
				if parameter, err := aws.Read[*ssm.Parameter](endpointParameter); err == nil && parameter.Value != nil {
					endpoint = strings.TrimRight(strings.TrimSpace(*parameter.Value), "/")
				}
				evaluator.lock.Lock()
				evaluator.evidence.Endpoint = endpoint
				evaluator.evidence.Active = time.Since(started) >= 50*time.Minute
				body, err := json.Marshal(evaluator.evidence)
				evaluator.lock.Unlock()
				var report struct {
					Results map[string]struct {
						OK     bool   `json:"ok"`
						Detail string `json:"detail"`
					} `json:"results"`
					Probe json.RawMessage `json:"probe"`
				}
				if err == nil {
					response, requestErr := s.SendHTTP(api.SendHTTPInput{
						Method: "POST", URL: *url.FunctionUrl, Body: body, TimeoutMillis: 30_000,
						Headers: map[string][]string{"Content-Type": {"application/json"}, "Authorization": {"Bearer " + token}},
					})
					err = requestErr
					if err == nil && response.StatusCode != 200 {
						err = fmt.Errorf("assessor HTTP %d: %.240s", response.StatusCode, response.Body)
					}
					if err == nil {
						err = json.Unmarshal(response.Body, &report)
					}
				}
				if err == nil {
					evaluator.lock.Lock()
					evaluator.evidence.Probe = report.Probe
					evaluator.lock.Unlock()
				}
				for _, criterion := range criteria {
					result, exists := report.Results[criterion.ID]
					if err != nil {
						result.OK, result.Detail = false, err.Error()
					} else if !exists {
						result.Detail = "Assessor did not return this criterion"
					}
					maximum, score := criterion.Points*1550, 0.0
					if result.OK {
						score = maximum
					}
					if _, err := api.SubmitScore(api.SubmitScoreInput{
						Name: criterion.ID + " " + criterion.Name, Type: api.ScoreTypeDesign,
						Score: score, Maximum: maximum, Reason: result.Detail,
					}); err != nil {
						slog.Error(fmt.Sprintf("submit %s: %v", criterion.ID, err))
					}
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-ticker.C:
				}
			}
			return nil
		},
	})
	return evaluator, nil
}
