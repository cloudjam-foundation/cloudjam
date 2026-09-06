# Live validation

Validated on 6 September 2026 in `eu-central-1` using `jamctl --fake=false` and
the supplied original applications.

- **48/48 design criteria: 124,000/124,000 points.** The AWS assessment Lambda
  read actual AWS and Kubernetes configuration and submitted scores through the
  WASM plugin. Half-point maxima preserve the original relative weights exactly.
  See [final design evidence](reference/live-design.json).
- **48/48 operational attempts: 400/400 points** across four consecutive jamctl
  rounds. Each event retained its own score and maximum, HTTP method, path,
  response status, latency and assertion result. See
  [the jamctl score history](reference/live-scoring.log).
- **72/72 standalone checks** passed across three baseline rounds and three
  cut-phase rounds. See [the HTTP probe results](reference/live-probe.jsonl).
- Two ARM64 EKS nodes in separate AZs ran two replicas of each original binary,
  with pod identity, HPAs, EFS CSI mounts, readiness probes and pod log collection.
  SHA-256 of `/app` inside each application container matched the supplied binary.
- Dashboard analytics used a healthy VPC Lattice service and the Route 53 private
  `analytics.vitalforge.internal` name. The analytics pods were not registered
  with the public ALB or API Gateway.
- Meal, workout and sleep IDs returned by successful requests were found with
  consistent DynamoDB reads. Sleep records carried the active threshold version.
- Changing SSM from the original baseline document to the original cut-phase
  document changed the dashboard and sleep validation. A 6.5-hour session changed
  from HTTP 202 to HTTP 422; an eight-hour session retained HTTP 202 and stored
  `cut-phase-2026`. Pod identities remained unchanged during the configuration
  change. See [runtime evidence](reference/live-runtime.json).
- The EKS binaries load thresholds at process startup. Validation used an SSM
  refresher and a small process supervisor inside the containers; these restarted
  the unchanged application processes when the document changed, without
  redeploying the pods. This requirement is documented in CLOUDJAM.md and the
  additional CloudJam briefing.
- The unchanged daily-export Lambda mounted the EFS access point across VPC
  peering, ran on a schedule and on direct invocation, and replaced the member's
  export. The EFS JSON file exactly matched the public export response.
- All four suggested Cognito users signed in with their supplied passwords.
  Coaches listed members; member accounts received HTTP 403 for that route and
  HTTP 200 for their own dashboards. See
  [Cognito/API evidence](reference/live-identity.json).
- The original console ZIP drove a successful CodePipeline/CodeBuild deployment
  and CloudFront invalidation. Its source ZIP remained byte-identical, as did
  deployed HTML, JavaScript and CSS; only the intended deployed `config.js` was
  populated with Cognito settings. The console origin used OAC and public access
  blocks. CloudFront had an associated WAF web ACL.
- The provided audit template delivered management events to its protected S3
  bucket. The architectural assessor verified successful delivery.
- During the SSM change, B1-07 lost points while runtime and cached configuration
  evidence disagreed and regained them after they matched, demonstrating design
  score regression and recovery.

The production plugin retains a **50-minute setup period**, **one-minute rounds**
and **310-round limit** within the six-hour module. The live scoring run used a
**temporary Go build overlay** with a 90-second setup period and four-round limit;
no assertion, point value or evaluator logic was bypassed. The standalone probe
starts immediately by design. A full six-hour timed game was not run.

The WASM plugin initialized under the same memory limit used by the hosted
platform and included the original architecture SVG and all application assets.
The final WASM and native probe build, Python lint, diff checks and all twelve
original file checksum comparisons passed. No test files or repository
dependencies were added. The assessment Lambda uses its runtime's bundled boto3.

All four validation CloudFormation stacks reached `DELETE_COMPLETE`. The final
inventory confirmed no validation functions, roles, buckets, VPCs, clusters,
Lattice resources, private zones, WAF ACLs or SSM parameters remained. The
pre-existing EKS cluster and VPC were preserved. See
[cleanup evidence](reference/live-cleanup.json).
