# CloudJam operation

`README.md`, `assets/architecture.svg`, and all nine files in `game-assets/`
are unchanged copies of the original GridLine day 1 artifacts. `delete_me`
is not needed to build or run this plugin. The briefing displays the original
SVG, also available unchanged in the assets.

Build and run from the repository root:

```sh
go run ./cmd/jamctl build -o gridline-day1.wasm ./examples/challenges/gridline-day1
go run ./cmd/jamctl run aws --fake=false --region eu-central-1 --name gridline-day1 ./examples/challenges/gridline-day1
```

Configure the CloudJam AWS provider for `eu-central-1`, using an empty competition
account and a six-hour CloudJam game. The plugin provisions no application
infrastructure. The asset bucket and sandbox access
are managed by CloudJam. Participants create all resources and IAM roles
described in the original README.

CloudJam additionally provisions a small assessment Lambda, its execution role,
and an authenticated assessment URL. These use the `cloudjam-gridline-assessor-`
prefix and are excluded from application marking. The sandbox boundary protects
the assessment function and role. The function reads AWS configuration and
runtime evidence; it does not provision any part of the participant's solution.

After deploying the CloudFront distribution, submit its HTTPS base URL:

```sh
aws ssm put-parameter --region eu-central-1 --name /cloudjam/gridline/day1/endpoint-url --type String --value https://YOUR-DISTRIBUTION.cloudfront.net --overwrite
```

Set `GRID_DEV_AUTH=1` on the original `race-api` container and forward
`X-Dev-Sub` and `X-Dev-Groups` through CloudFront and API Gateway, as required
by the original brief. The evaluator uses `scoring-director` and
`scoring-spectator`. Requests also carry `X-CloudJam-Check` for correlation
with application and API access logs.

The 13 check IDs, names, and point values are copied from the captured scoring
data. Evaluation starts 50 minutes after the challenge starts. No requests or
operational score events are emitted during this setup period. Checks then run
every minute for the remaining 310 minutes of the six-hour module, earning up to
100 points per round (31,000 maximum over the module).

Each Activity event and `jamctl` result contains only that attempt's score and
maximum: for example, health reports `5/5` for success or `0/5` for failure.
Consecutive successes and failures each produce their own event. The round,
HTTP method, path, response status, latency, and assertion details are included.
A prerequisite failure is identified as a request that was not sent.

The SDK submission uses `Accumulate: true`: the host adds each attempt to the
overall score and maximum while keeping the event's values unchanged. Failed
checks do not subtract previously earned points. Score items are created by the
first submission; no unattempted round is added during registration. The
operational maximum reflects all attempted rounds after the setup period,
including rounds before an endpoint is submitted.

The capture includes scoring results, not the original evaluator source.
Recorded failures establish 800 ms for health, 1,000 ms for start, 300 ms for
leaderboard, and 500 ms for rules and RUNNING. Other APIs use the README's
500 ms target. Heat creation, pre-start rejection, start, RUNNING, unknown
transponder rejection, and leaderboard checks use the same newly created heat.
The RUNNING check allows up to ten seconds for the asynchronous workflow.
Unrecorded request fixtures and retry behavior cannot be recovered exactly.
The subsequently supplied `results-linus.moser.pdf` provides all 59 day 1 marking
criteria, totaling 80 marks. Their names and weights are preserved in
[the marking scheme](MARKING.md); personal results are not part of the plugin.
Multiplying each original mark by 1,550 gives 124,000 design points alongside the
31,000 operational maximum, preserving an 80% design / 20% operational split.

Design marks reflect the latest assessment, including regressions. The assessor
runs every 30 seconds and caches AWS configuration for two minutes. It reuses the
current operational checks for the PDF's functional and success-rate criteria.
Latency criteria use p95 over the latest 20 recorded responses, with at least
three observations per path. Design visibility follows CloudJam's existing rules.

Additional functional probes start after the same 50-minute setup period. They
create a three-minute heat, send six real-time crossings from three drivers,
check both lap-time bounds, inspect standings, and verify completion and the
matching S3 telemetry object. SNS-to-SQS messages are observed without deletion.
These probes stop starting new heats during the last four minutes of the module.
Assessment details and the few original evidence proxies are listed in MARKING.md.

Run the same evaluator directly against a deployed solution:

```sh
go run ./examples/challenges/gridline-day1/cmd/probe -endpoint https://YOUR-DISTRIBUTION.cloudfront.net -rounds 3
```

The probe emits JSON results and exits unsuccessfully if any check fails.
Passing these 13 checks verifies the recovered scoring criteria; the original
brief also requires valid lap processing, runtime rules, finalization, telemetry,
SNS delivery, Firehose delivery, and console CI/CD.

The original runtime document contains empty deployment-specific values, and
the console archive includes an empty `config.js`. Keep the supplied files
unchanged; fill runtime values in the hosted AppConfig deployment and generate
the deployed console configuration in CodeBuild after extracting the archive.
The supplied finalizer requires database environment variables, as explicitly
documented in the original README.

The original ECS binaries also require an AWS AppConfig Agent sidecar at
`http://127.0.0.1:2772`. Set its task role permissions to retrieve configuration,
prefetch both profiles, and allow it to start before the application container.
The binaries support `APPCONFIG_AGENT_URL` and `APPCONFIG_POLL_SECONDS`.

The original binary returns 403 for anonymous heat mutations. To satisfy the
README's 401 requirement as well, configure the ALB to return 401 when neither
a Bearer authorization header nor both scoring headers are present on protected
routes. Keep `/health` public and let the binary enforce director/spectator roles.
For an HTTP proxy integration, forward the original request path to the ALB.
