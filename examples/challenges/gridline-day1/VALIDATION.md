# Live validation

Validated on 6 September 2026 in `eu-central-1` with `jamctl --fake=false`.

- The WASM scorer passed four consecutive rounds: **52/52 checks,
  400/400 operational points**. It ran with the hosted platform's 70 MB memory
  limit. See [the jamctl results](reference/live-scoring.log).
- The original SVG was also embedded byte for byte and initialized successfully
  within the hosted memory limit; the briefing does not use a converted preview.
- The standalone probe passed three consecutive rounds: **39/39 checks**.
  See [the probe results](reference/live-probe.jsonl).
- Both original ARM64 application binaries ran on ECS Fargate with AppConfig
  Agent sidecars, private task addresses, Secrets Manager credential injection,
  and CPU target tracking. Aurora occupied the separate data VPC, reached through
  Transit Gateway.
- Real Cognito director and spectator sign-ins succeeded. Anonymous mutations
  returned 401 at the ALB; spectator mutations returned 403 from the original
  binary. The browser console signed in, listed six tracks, and displayed the
  active wet-track rules.
- A 180-second heat processed six real-time crossings from three drivers. Three
  laps used baseline rules; three used wet-track rules, changed through AppConfig
  without replacing the running containers. Invalid, unknown, premature, and
  post-finish crossings were rejected.
- The unchanged finalizer marked the heat FINISHED and wrote telemetry containing
  six laps and a three-driver podium. SNS delivered the completion to SQS, and
  Firehose delivered all six records to the S3 `laps/` prefix. See
  [the pipeline results](reference/live-pipeline.json).
- Uploading the unchanged console ZIP triggered CodePipeline and CodeBuild,
  published the four console files, and invalidated CloudFront. Only the deployed
  `config.js` was generated with the account's Cognito configuration.
- The original README and every supplied asset passed
  `sha256sum -c reference/checksums.sha256`.
- The plugin and affected Go packages built successfully. The web build succeeded;
  `pnpm check` reported zero errors and one existing warning in `GamePanel.svelte`.
  No test files or dependencies were added.
- Both validation stacks reached DELETE_COMPLETE. Temporary asset buckets,
  sandbox roles and policies, endpoint parameter, generated log groups, and local
  validation images were removed. The account's original VPC was preserved.

The recovered scoring assertions are exercised against the original applications.
The original archive does not include the evaluator source. Its missing design
rubric was subsequently supplied in `results-linus.moser.pdf`; see the
[marking scheme](MARKING.md) for the restored criteria and assessment details.

## Per-attempt scoring and delayed start

The live evidence above predates the per-attempt event and 50-minute setup-period
changes. Those changes were verified locally on 6 September 2026:

- Hosted score storage retained six separate events for `0, 0, 5, 5, 0, 5`
  points, each with maximum 5, while the operational total reached `15/30`.
  Repeated identical failures and successes were recorded. Existing design-score
  replacement and deduplication behavior remained unchanged, and the team total
  matched the accumulated operational points.
- A WASM plugin using the updated SDK produced the same results through jamctl.
- GridLine's evaluation loop emitted no resource requests before its setup timer,
  then produced three rounds and 39 per-attempt events. This smoke run accelerated
  the timer to 200 ms and the interval to 100 ms using a temporary build overlay;
  the shipped plugin retains the 50-minute delay, one-minute interval, and
  310-round limit. Cancellation succeeded.
- The updated GridLine WASM built successfully. Original asset checksums and
  `git diff --check` passed. Temporary smoke programs and overlays were removed;
  no test files or dependencies were added.

## Architectural marking

- Extracted all 59 day 1 criteria from the supplied PDF. Their maxima sum to
  exactly 80 original marks and 124,000 CloudJam design points. Personal results
  and other days' criteria are not included.
- Complete local AWS/runtime evidence passed all 59 criteria. Empty inventory
  earned no marks. Denied reads, world-open SSH, wildcard IAM, unencrypted Aurora,
  and failed functional checks removed the relevant marks.
- A jamctl WASM run with local provider controllers emitted all 59 design score
  events totaling 124,000 points within the hosted 70 MB memory limit. The run
  also verified assessment authentication, the sandbox guardrail, and the absence
  of operational requests during the setup period.
- A simulated heat workflow exercised the additional probe's six valid crossings,
  min/max rejection, standings, completion, and final-four-minute cutoff.
- The updated WASM builds and Python lint checks pass. No test files or repository
  dependencies were added; the assessment Lambda uses its runtime's bundled boto3.

Live AWS validation of the new assessment Lambda is pending because the
`SwissskillsDev` SSO session expired. The earlier live solve above verifies the
original applications and pipeline, not deployment of the new assessor.
