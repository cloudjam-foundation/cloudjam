# VitalForge day 2 marking

Source: the day 2 section of `results-linus.moser.pdf`, pages 5–7. Only criterion
names, aspects and maximum marks are reproduced; personal results are excluded.
IDs add an ordinal within each original aspect.

The 48 criteria total **80 original marks**. Each mark is multiplied by **1,550**
for **124,000 CloudJam design points**. Operational checks award up to **31,000**
(310 rounds × 100), preserving an **80% design / 20% operational** split.
Half-point maxima are retained to preserve the exact total and relative weights.
Design marks reflect the latest assessment, including regressions.

| ID | Criterion | Original marks | CloudJam design points |
| --- | --- | ---: | ---: |
| A1-01 | DynamoDB tables use on-demand billing | 0.8 | 1,240 |
| A1-02 | EKS node group right-sized for wellness-api and analytics-engine | 1.6 | 2,480 |
| A1-03 | Single CloudFront distribution for console and APIs | 0.8 | 1,240 |
| A1-04 | VPC peering connects app-vpc to files-vpc for EFS (not Transit Gateway) | 1.6 | 2,480 |
| A1-05 | Lambda memory and timeout sized for sleep-ingest and daily-export | 0.8 | 1,240 |
| A1-06 | S3 lifecycle configured on console origin bucket | 0.8 | 1,240 |
| B1-01 | app-vpc and files-vpc exist with documented separation | 0.8 | 1,240 |
| B1-02 | VPC peering enables EFS NFS from approved app-vpc security groups | 1.6 | 2,480 |
| B1-03 | DynamoDB tables per §9.3 with seed-users.json loaded | 1.6 | 2,480 |
| B1-04 | EKS linux/arm64 runs wellness-api and analytics-engine | 1.6 | 2,480 |
| B1-05 | Cognito user pool with coach and member groups (console identity only; §9.1) | 1.6 | 2,480 |
| B1-06 | CloudFront and WAF cover /, /console/*, /api/*, /health, and /sleep | 1.6 | 2,480 |
| B1-07 | SSM /vitalforge/thresholds/active read at runtime without redeploy | 1.6 | 2,480 |
| B1-08 | VPC Lattice service network connects wellness-api to analytics-engine | 1.6 | 2,480 |
| B1-09 | Route 53 private zone analytics.vitalforge.internal resolves Lattice hostname | 0.8 | 1,240 |
| B1-10 | sleep-ingest Lambda on POST /sleep (202/400/422 contract) | 1.6 | 2,480 |
| B1-11 | EventBridge schedule invokes daily-export to EFS exports/{user_id}/latest.json | 1.6 | 2,480 |
| B1-12 | audit-bootstrap.yaml CloudTrail delivers management events to S3 | 0.8 | 1,240 |
| B1-13 | Uploading console.zip alone completes deploy with CloudFront invalidation | 0.8 | 1,240 |
| C1-01 | API Gateway HTTP API has no JWT authorizer on /api/* (auth enforced in wellness-api per §8.13) | 1.85 | 2,867.5 |
| C1-02 | analytics-engine not exposed on public ALB, API Gateway, or CloudFront | 1.85 | 2,867.5 |
| C1-03 | Console S3 origin not publicly readable (OAC or equivalent) | 1.85 | 2,867.5 |
| C1-04 | EFS only in files-vpc; NFS port 2049 restricted to approved security groups | 1.85 | 2,867.5 |
| C1-05 | AWS WAF web ACL associated with CloudFront distribution | 1.85 | 2,867.5 |
| C1-06 | EKS and Lambda task roles follow least-privilege IAM | 1.85 | 2,867.5 |
| C1-07 | No security group allows inbound 0.0.0.0/0 on port 22 | 0.93 | 1,441.5 |
| C1-08 | CloudTrail S3 bucket blocks public access | 0.93 | 1,441.5 |
| C1-09 | Provided wellness-api, analytics-engine, Lambdas, and console.zip unmodified | 0.93 | 1,441.5 |
| C1-10 | EFS access points with IAM authorization and encryption in transit on mounts | 1.85 | 2,867.5 |
| C1-11 | Data store is DynamoDB only (not Aurora or RDS) | 0.93 | 1,441.5 |
| C1-12 | All marked resources deployed in one chosen AWS Region | 0.93 | 1,441.5 |
| D1-01 | EKS workloads span multiple AZs or maintain min replicas >= 2 | 2.34 | 3,627 |
| D1-02 | Meal, workout, and sleep sessions persisted before HTTP success | 2.34 | 3,627 |
| D1-03 | Dashboard analytics block populated via VPC Lattice (not public edge) | 2.34 | 3,627 |
| D1-04 | daily-export overwrites exports/{user_id}/latest.json on EFS | 2.34 | 3,627 |
| D1-05 | CloudWatch logs enabled for EKS pods and Lambda functions | 2.34 | 3,627 |
| D1-06 | EFS file system encrypted at rest | 1.16 | 1,798 |
| D1-07 | Coach POST /api/members/{id}/export triggers or queues daily-export | 2.34 | 3,627 |
| E1-01 | CloudFront is the single public HTTPS entry point | 3.52 | 5,456 |
| E1-02 | EKS horizontal pod autoscaling or equivalent capacity configured | 3.52 | 5,456 |
| E1-03 | CloudFront caches static console assets at / | 1.76 | 2,728 |
| F1-01 | S3 lifecycle transitions console artifacts toward IA or archive | 1.6 | 2,480 |
| F1-02 | EKS scales in when traffic subsides | 1.6 | 2,480 |
| F1-03 | No wasteful duplicate always-on stacks beyond brief scope | 1.6 | 2,480 |
| F1-04 | DynamoDB on-demand avoids idle provisioned RCU/WCU | 1.6 | 2,480 |
| F1-05 | Lambda reserved concurrency or limits avoid runaway export cost | 1.6 | 2,480 |
| G1-01 | End-to-end: log meal, workout, sleep, dashboard, and export readable | 3.2 | 4,960 |
| G1-02 | Submitted CloudFront URL serves /, /health, /api/*, and /sleep | 3.2 | 4,960 |

## Assessment evidence

The assessor reads AWS configuration every two minutes and evaluates every
30 seconds. It discovers EKS clusters whose name contains `vitalforge`; for a
different cluster name, set the String parameter
`/cloudjam/vitalforge/day2/cluster-name` in the game Region. Keep `wellness-api`
and `analytics-engine` in their container names or image references.

EKS must use `API` or `API_AND_CONFIG_MAP` access mode and have an API endpoint
reachable by the assessor. The assessor grants only its own protected IAM role
`AmazonEKSAdminViewPolicy`, a read-only Kubernetes access policy. It reads
workloads, pods, nodes, service accounts, autoscalers and persistent volumes.
It does not request Kubernetes secrets or modify application workloads.

The PDF contains criterion names and example evidence, not evaluator source.
These assessment interpretations are explicit:

- A1 sizing accepts medium through xlarge EKS node instances, and application
  Lambdas with 128–1,024 MiB memory and 3–120 second timeouts.
- B1-03 checks the exact DynamoDB key schemas and the three supplied seed IDs.
- B1-07 compares the runtime dashboard threshold version to active SSM and
  requires successful configuration and sleep checks. It records observed
  version changes without changing participant configuration itself.
- B1-08 checks an active service network association with app-vpc and healthy
  Lattice targets matching the running analytics pods.
- B1-11 and D1-04 require scheduled original daily-export, an EFS access-point
  mount and replacement of generated_at on the existing export path.
- B1-12 requires an active management trail with successful S3 delivery.
- B1-13 requires a console.zip source, successful pipeline stages and a buildspec
  containing CloudFront invalidation.
- C1-06 inspects actual application pod and Lambda role policies, including
  attached policies. Wildcard actions and unscoped application data access fail.
  AWS actions that require an unscoped resource have limited exceptions.
- C1-09 verifies the original Lambda ZIP hashes and running named EKS workloads
  plus the console pipeline. Container payload and deployed console byte identity
  require manual checksum verification; asset downloads are byte-identical.
- C1-10 checks `tls` and `iam` mount options and access-point volume handles on
  EKS EFS volumes, plus the Lambda access point.
- C1-12 inventories the game Region and application bucket locations. IAM,
  CloudFront and CloudFront WAF are global services. Unrelated Regions are not
  exhaustively inventoried.
- D1-02 uses consistent DynamoDB reads for the exact IDs returned by the latest
  successful meal, workout and sleep requests. This establishes persistence after
  acknowledgment, not a direct transaction timing audit.
- D1-05 requires a pod log collector and CloudWatch groups, plus application
  Lambda log groups. Control-plane logs alone do not satisfy pod logging.
- E1-02 and F1-02 inspect active application HPAs and enabled scale-down behavior.
- F1-03 limits the identified solution to one EKS cluster, one deployment per
  application, one EFS system and one instance of each supplied Lambda.
- F1-05 accepts export reserved concurrency or a regional account concurrency
  limit from 1 to 10. The assessor does not require reserved concurrency.

Additional runtime reads and sleep rejection probes start after minute 50.
Failures to read required evidence fail the affected criteria with an explanation.

Source PDF SHA-256: `dc2a8a0ee018f1e35f08cf33289ee3c649a2004a255db913fd491a3e0db67271`.
