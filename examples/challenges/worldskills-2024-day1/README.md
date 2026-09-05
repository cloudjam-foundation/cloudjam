# WorldSkills Lyon 2024 — Day 1

This challenge reconstructs Day 1 from the 2024 WorldSkills Cloud Computing test
project and marking sheet. The original application archives are unavailable, so
the replacements in `cmd/data-processing` and `cmd/data-extraction` implement the
documented DynamoDB, PostgreSQL, and Parameter Store pipeline. `cmd/day1-probe`
is a contestant-side end-to-end smoke test.

Generate the Linux amd64 assets and build the challenge with:

~~~sh
go generate ./examples/challenges/worldskills-2024-day1/...
jamctl build -o worldskills-2024-day1.wasm ./examples/challenges/worldskills-2024-day1
~~~

The original live input and scoreboard are represented by an HTTP ingestion
endpoint submitted through Systems Manager and four rolling success-rate bands.
The original ranking mark is represented by an end-to-end architecture check.
