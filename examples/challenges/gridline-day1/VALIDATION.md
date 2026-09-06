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
The archive does not include the evaluator source or a weighted design rubric;
the reconstruction limits are documented in [CLOUDJAM.md](CLOUDJAM.md).
