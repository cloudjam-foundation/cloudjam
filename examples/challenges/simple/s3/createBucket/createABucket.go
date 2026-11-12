//go:build wasip1

package main

import (
	_ "embed"
	"strings"
	"time"

	"codeberg.org/megakuul/cloudjam/pkg/challenge"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/policy"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/aws/services/s3"
)

// This makes it easy to reuse the challenge code
// and to just change the prefixes and the region

var REGION = "us-east-1"
var RESOURCE_PREFIX = "cloudjam"

//go:embed brief.md
var brief []byte

//go:embed clues/clue1.md
var clue []byte

func bootstrap(s *challenge.Scenario) error {
	s.AddDescription(string(brief))

	// I have made this clue consume all of the points you can
	// get in this challenge because it tells the competitor
	// the exact steps to complete it.
	s.AddClue("Step by step solution", string(clue), 100)

	// This is a design check as it does not send requests to the bucket.
	s.AddCheck("Does the S3 exist", challenge.NewDesignCheck(100, "The S3 Bucket Exists", func() (bool, error) {
		bucketList, err := aws.List[*s3.Bucket]()

		if err != nil {
			return false, err
		}

		for bucket := range bucketList {
			if strings.HasPrefix(bucket, RESOURCE_PREFIX) {
				return true, nil
			}
		}

		return false, nil
	}))

	return nil
}

func main() {
	simpleChallenge := challenge.New("Create an S3 Bucket", 1*time.Second, bootstrap)

	s3Policy := policy.Document{
		Version: policy.Version20121017,
		Statement: []policy.Statement{
			{
				Sid:    "onlyAllowS3BucketCreation",
				Effect: policy.Allow,
				Action: policy.Actions{
					s3.ActionListBucket,
					s3.ActionCreateBucket,
				},
				Resource: policy.ARNs{
					"arn:aws:s3:::cloudjam-*",
					"arn:aws:s3:::cloudjam-*/*",
				},
				Condition: policy.Conditions{
					policy.StringEquals(policy.KeyRequestedRegion, REGION),
				},
			},
		},
	}

	simpleChallenge.SetGuardrail(s3Policy)
	simpleChallenge.SetPermission(s3Policy)
	simpleChallenge.Start()
}
