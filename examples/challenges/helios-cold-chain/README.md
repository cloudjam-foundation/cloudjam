# Helios Cold-Chain Relay

Helios is an original advanced CloudJam challenge built around a durable
serverless telemetry pipeline. Competitors connect Lambda Function URLs,
EventBridge, SQS, Step Functions, Secrets Manager, S3, and SNS, then harden and
operate the result.

Generate the Linux assets and build the challenge with:

~~~sh
go generate ./examples/challenges/helios-cold-chain/...
jamctl build -o helios-cold-chain.wasm ./examples/challenges/helios-cold-chain
~~~

The five Lambda archives are deployable arm64 `provided.al2023` applications.
`helios-probe.zip` contains a Linux client that submits a safe reading and a
temperature excursion, then verifies both results through the query endpoint.
The challenge awards 320 design points and 20 rolling operational points.
