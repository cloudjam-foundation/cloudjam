# Borealis Field Observatory

An advanced CloudJam challenge built around a container and AWS's IoT and analytical services instead of a conventional serverless message pipeline.

The contestant deploys a supplied Go container to App Runner, reads classification limits from AppConfig, publishes observations through IoT Core, delivers them with Firehose, catalogs the S3 lake with Glue, and queries the result through Athena.

Generate the supplied artifacts with:

```sh
go generate ./examples/challenges/borealis-field-observatory/internal/generate
```

Build the challenge with:

```sh
jamctl build ./examples/challenges/borealis-field-observatory
```
