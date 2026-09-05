# WorldSkills Lyon 2024 — Day 2

This challenge reconstructs Day 2 from the 2024 WorldSkills Cloud Computing test
project and its marking sheet. The original `stub1.zip` is no longer available,
so the replacement in `cmd/stub1` implements the documented DynamoDB query
contract. `cmd/day2-probe` is a contestant-side smoke test for both pipelines.

Generate the Linux amd64 assets and build the challenge with:

~~~sh
go generate ./examples/challenges/worldskills-2024-day2/...
jamctl build -o worldskills-2024-day2.wasm ./examples/challenges/worldskills-2024-day2
~~~

The old CloudRaiser input form is represented by three SSM String parameters.
Their names and the assumed HTTP contracts are part of the challenge
description. The two leaderboard marks from the original multi-competitor event
are represented by an end-to-end architecture check.
