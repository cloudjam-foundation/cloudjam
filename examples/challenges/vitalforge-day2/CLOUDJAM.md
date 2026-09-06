# CloudJam operation

The original README, architecture SVG and all ten files in `game-assets/` are
unchanged copies of the supplied day 2 export. The briefing embeds the original
SVG. Check integrity with `sha256sum -c reference/checksums.sha256` from this
directory. `delete_me` is not needed to build or run the plugin.

From the repository root:

```sh
go run ./cmd/jamctl build -o vitalforge-day2.wasm ./examples/challenges/vitalforge-day2
go run ./cmd/jamctl run aws --fake=false --region eu-central-1 --name vitalforge-day2 ./examples/challenges/vitalforge-day2
```

Use a six-hour CloudJam game and one supported AWS Region. Participants deploy the
application themselves. CloudJam supplies assets, sandbox access and a protected
assessment Lambda/role, excluded from application marking.

Submit the CloudFront HTTPS base URL as a String parameter in the game Region:

```sh
aws ssm put-parameter --region eu-central-1 --name /cloudjam/vitalforge/day2/endpoint-url --type String --value https://YOUR-DISTRIBUTION.cloudfront.net --overwrite
```

The evaluator forwards the original mock authentication headers from §9.1:
`X-Dev-Sub: scoring-coach` / `X-Dev-Groups: coach`, or
`X-Dev-Sub: member-demo-1` / `X-Dev-Groups: member`. Preserve these headers through
CloudFront and API Gateway. `X-CloudJam-Check` identifies each request in access
logs. Authentication stays in the supplied wellness-api binary.

Operational traffic begins after **50 minutes**, then runs once per minute for
**310 rounds**. Each of the twelve recovered checks retains its original name,
ID and points. Each round awards up to 100 points. Every attempt produces an
Activity event containing only its own score and maximum, method, path, status,
latency and assertion details. `Accumulate: true` adds those attempts to the
overall total while retaining individual successes and failures in history.

The captured scoring data does not include evaluator source. Requests are
reconstructed from the README: meals and workouts use member-demo-1; valid sleep
is eight hours and invalid sleep one hour. Dashboard latency is limited to
500 ms, sleep ingest to the captured 1,000 ms, and other routes to the README's
800 ms target. Export reads retry 404 responses for up to ten seconds while the
asynchronous export finishes. Responses must match the documented JSON fields.

The 48 original marking criteria award 124,000 design points alongside 31,000
operational points. Design marks follow the latest assessment and can regress.
See [MARKING.md](MARKING.md) for discovery, EKS assessment access and evidence.

Run the same HTTP checks directly against a deployed solution:

```sh
go run ./examples/challenges/vitalforge-day2/cmd/probe -endpoint https://YOUR-DISTRIBUTION.cloudfront.net -rounds 3
```

The probe starts immediately, emits JSON results and exits unsuccessfully if any
check fails. It does not submit scores or change the plugin's setup delay.

Preserve the supplied console archive and generate the deployed `config.js` with
Cognito settings during CodeBuild. Deploy both original Lambda ZIPs as
`provided.al2023`, `arm64`, handler `bootstrap`. EKS binaries listen on port 8080.

The supplied EKS processes load thresholds at startup. For runtime updates,
use an SSM refresher to write the active document atomically to a shared volume,
set `THRESHOLDS_CONFIG_PATH`, and supervise the original application process so
it restarts when that file changes. Initialize the file before starting the
application. Keep at least two replicas and readiness probes. This leaves the
provided binary unchanged and avoids redeploying Kubernetes pods when thresholds
change. The supplied Lambda handlers read SSM during invocation.

For an API Gateway HTTP proxy to the ALB, preserve the incoming route with
`overwrite:path = $request.path`. Use an uncached API CloudFront behavior that
forwards the scoring headers. Keep API origins, console buckets and application
resources in the selected Region; CloudFront WAF is a global resource managed
through `us-east-1`.
