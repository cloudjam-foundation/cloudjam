# GridLine day 1 marking

Source: the day 1 section of `results-linus.moser.pdf`, pages 1–4.
Only criterion names, aspects and maximum marks are reproduced. Personal results
and the day 2/day 3 schemes are not included. IDs add an ordinal within each
original aspect.

The 59 criteria total **80 original marks**. Each mark is multiplied by **1,550**
for **124,000 CloudJam design points**. The operational maximum is **31,000**
(310 rounds × 100), giving an **80% / 20%** split at the six-hour module maximum.

| ID | Criterion | Original marks | CloudJam design points |
| --- | --- | ---: | ---: |
| A1-01 | S3 lifecycle configured on console origin bucket | 1.04 | 1,612 |
| A1-02 | S3 lifecycle configured on laps/ data-lake bucket | 0.84 | 1,302 |
| A1-03 | Aurora Serverless v2 or right-sized provisioned cluster | 1.68 | 2,604 |
| A1-04 | ECS Fargate task CPU/memory not grossly over-provisioned | 0.84 | 1,302 |
| A1-05 | Single Transit Gateway fabric for track-vpc ↔ data-vpc DB path | 1.68 | 2,604 |
| A1-06 | Firehose delivery stream uses compression or buffering sensibly | 0.84 | 1,302 |
| B1-01 | track-vpc and data-vpc exist with documented separation | 0.84 | 1,302 |
| B1-02 | Transit Gateway connects track-vpc to data-vpc for Aurora access | 1.68 | 2,604 |
| B1-03 | Aurora PostgreSQL griddb loaded from init.sql (6 tracks, 40 drivers) | 1.68 | 2,604 |
| B1-04 | ECS Fargate linux/arm64 runs race-api and timing-engine | 1.68 | 2,604 |
| B1-05 | Cognito user pool with director and spectator groups | 1.68 | 2,604 |
| B1-06 | CloudFront behaviours cover /, /api/*, /health, and /transponder | 1.68 | 2,604 |
| B1-07 | AppConfig publishes runtime-config and timing rules (baseline + wet-track) | 1.68 | 2,604 |
| B1-08 | POST /transponder API Gateway → Lambda ingests laps to Aurora | 1.68 | 2,604 |
| B1-09 | SQS lap-events consumed by timing-engine; standings updated | 0.84 | 1,302 |
| B1-10 | EventBridge HeatStartRequested (gridline.race) starts Step Functions | 0.84 | 1,302 |
| B1-11 | Step Functions workflow invokes provided finalize-heat Lambda | 0.84 | 1,302 |
| B1-12 | SNS topic publishes heat.finished to subscribed SQS notifications queue | 0.84 | 1,302 |
| B1-13 | Firehose delivers lap JSON records to S3 under laps/ prefix | 0.84 | 1,302 |
| B1-14 | Uploading console.zip alone completes deploy with CloudFront invalidation | 0.84 | 1,302 |
| B1-15 | telemetry/{heat_id}.json object written on heat completion | 0.84 | 1,302 |
| C1-01 | race-api enforces auth on protected /api/* routes | 1.68 | 2,604 |
| C1-02 | Unauthenticated GET /api/tracks returns 401 | 0.84 | 1,302 |
| C1-03 | Spectator token cannot POST /api/heats (403) | 0.84 | 1,302 |
| C1-04 | Aurora credentials stored in Secrets Manager only | 1.68 | 2,604 |
| C1-05 | timing-engine is not reachable from the public internet | 1.68 | 2,604 |
| C1-06 | Console S3 origin bucket is not publicly readable | 1.68 | 2,604 |
| C1-07 | Aurora cluster runs only in data-vpc (not track-vpc subnets) | 1.68 | 2,604 |
| C1-08 | Database path uses Transit Gateway (not VPC peering or PrivateLink) | 0.84 | 1,302 |
| C1-09 | Banned services not used: Route 53, PrivateLink, ElastiCache, VPC Lattice | 0.84 | 1,302 |
| C1-10 | ECS and Lambda task roles follow least-privilege IAM | 1.68 | 2,604 |
| C1-11 | No security group allows inbound 0.0.0.0/0 on port 22 | 0.84 | 1,302 |
| C1-12 | All marked resources deployed in eu-central-1 | 0.84 | 1,302 |
| C1-13 | AppConfig validation prevents invalid timing rules reaching production | 1.68 | 2,604 |
| C1-14 | Aurora storage encrypted at rest | 0.84 | 1,302 |
| D1-01 | Aurora PostgreSQL cluster is Multi-AZ | 1.68 | 2,604 |
| D1-02 | ECS services span multiple AZs or maintain min tasks >= 2 | 1.68 | 2,604 |
| D1-03 | Heat progresses SCHEDULED → RUNNING → FINISHED via Step Functions | 1.68 | 2,604 |
| D1-04 | Valid transponder lap persisted to Aurora before HTTP 202 | 1.68 | 2,604 |
| D1-05 | scoring.json functional checks pass | 1.68 | 2,604 |
| D1-06 | CloudWatch logs enabled for ECS tasks and Lambda functions | 1.68 | 2,604 |
| D1-07 | lap-events queue poison messages do not block processing indefinitely | 0.84 | 1,302 |
| D1-08 | Success rate (SRR) >= 90% from scoring checks | 1.68 | 2,604 |
| D1-09 | Success rate (SRR) >= 80% | 0.84 | 1,302 |
| D1-10 | Success rate (SRR) >= 70% | 0.84 | 1,302 |
| E1-01 | CloudFront is the single public HTTPS entry point | 1.68 | 2,604 |
| E1-02 | GET /api/heats/{id}/leaderboard p95 <= 300ms under scoring | 1.68 | 2,604 |
| E1-03 | Other API paths p95 <= 500ms under scoring | 1.68 | 2,604 |
| E1-04 | ECS service autoscaling configured for race-api and/or timing-engine | 1.68 | 2,604 |
| E1-05 | CloudFront caches static console assets at / | 0.84 | 1,302 |
| E1-06 | Transponder ingest returns 422 for rule violations (min/max lap_ms) | 1.68 | 2,604 |
| F1-01 | S3 lifecycle transitions laps/ or telemetry/ objects toward IA/archive | 2.52 | 3,906 |
| F1-02 | ECS Fargate Spot capacity provider configured (optional credit) | 0.84 | 1,302 |
| F1-03 | Autoscaling scales in when race-day load subsides | 1.68 | 2,604 |
| F1-04 | No wasteful duplicate always-on stacks beyond brief scope | 1.68 | 2,604 |
| G1-01 | Core scoring.json checks pass during marking window | 1.68 | 2,604 |
| G1-02 | heat.finished SNS→SQS notification path configured (§9.9) | 1.68 | 2,604 |
| G1-03 | End-to-end heat: create → start → laps → leaderboard → finish within module | 1.68 | 2,604 |
| G1-04 | Submitted CloudFront URL serves /, /health, /api/*, and /transponder | 1.68 | 2,604 |

## Assessment behavior

All criteria are registered in the design category and updated throughout the
module. A passed criterion can lose its points if the latest assessment fails.
Operational attempts continue to accumulate separately and display per-attempt
values in Activity.

The assessor uses AWS service APIs for infrastructure and storage evidence. It
identifies application ECS services from the `race-api` and `timing-engine`
container names or image references, follows their task definitions and subnets,
and identifies the supplied finalizer by its deployment ZIP SHA-256. Keep those
application identifiers in container names when using differently named ECR
repositories. Resource names otherwise remain flexible.

After minute 50, a separate probe creates a 180-second heat, observes RUNNING,
sends two real-time batches of three crossings, checks min/max lap rejection,
reads standings, and observes FINISHED. S3 telemetry must match that heat. A valid
SNS notification is read from a subscribed SQS queue with zero visibility timeout
and is not deleted. The assessment creates no application resources and never writes
to the database outside the public application API.

The PDF supplies criterion names, weights, and example evidence, not evaluator
source. The following interpretation details are explicit:

- A1 sizing accepts Aurora Serverless v2 or medium/large provisioned instances,
  and application tasks up to 1 vCPU / 4 GiB. Firehose buffering follows the AWS
  supported bounds shown in the source evidence.
- B1-03 follows the original scoring proxy: Aurora `griddb` plus six seeded tracks
  through the original API. It does not directly count the forty database drivers.
- B1-08 and D1-04 verify accepted crossings becoming visible in standings. This
  is runtime evidence of persistence, not a direct SQL transaction audit.
- B1-14 requires an S3 `console.zip` pipeline source, a CodeBuild buildspec with
  CloudFront invalidation, and successful execution of every pipeline stage.
- C1-04 allows Secrets Manager references for ECS and preserves the README's
  explicit exception for database environment variables on the unchanged
  finalizer. Other application Lambdas must not contain a plain DB password.
- C1-10 checks the actual application roles, including attached managed policies,
  for wildcard actions and unscoped resources. Actions that require `Resource: *`
  have limited exceptions; the assessment role is excluded.
- C1-12 examines marked regional resources and application bucket locations. It
  does not inventory every unrelated AWS Region. CloudFront and IAM are global.
- C1-13 verifies that timing-rule AppConfig profiles have validators configured.
  It does not deploy deliberately invalid documents into the player's environment.
- D1 success-rate bands use the latest complete round of thirteen functional
  checks, matching the PDF's current-check success-rate evidence.
- E1 latency uses the most recent twenty measured responses per check and the
  nearest-rank p95, with at least three samples.
- F1-04 requires one race-api service, one timing-engine service, one Aurora
  cluster, one CloudFront distribution and at most sixteen core resources.

Source PDF SHA-256: `dc2a8a0ee018f1e35f08cf33289ee3c649a2004a255db913fd491a3e0db67271`.
