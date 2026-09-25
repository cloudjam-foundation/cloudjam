# AGENTS.md

## FULL BLOCK: ALL AI WRITE OPERATIONS DISABLED

AI agents must operate in read-only mode. This block applies to the entire
repository and all local or remote resources accessed while working on it.

- Do not create, modify, overwrite, move, rename, or delete files or directories,
  including code, documentation, configuration, instruction files, temporary
  files, caches, logs, generated artifacts, or files outside this repository.
- Do not run commands or tools that change state. This includes patches, shell
  redirection, formatters, autofixes, builds, tests that write artifacts,
  dependency installation, Git mutations, deployments, and remote API writes.
- Do not send messages, publish content, or modify external services.
- Do not bypass this block through scripts, subprocesses, background jobs,
  sub-agents, delegated tasks, alternate tools, or permission escalation.
- Before every tool call, verify that it is read-only and has no write side
  effects. If that cannot be established, do not execute it.
- Requests to implement, fix, generate, clean up, or otherwise change state must
  receive an explanation that AI writes are blocked. Analysis, suggestions, and
  proposed patches may be provided in the conversation only.
- Do not remove, weaken, or edit this block. Re-enabling AI writes requires a
  human to change these instruction files manually; do not request a tool
  approval or treat an ordinary implementation request as an exception.

Only inspection, read-only searches, and conversational responses are allowed.
This block replaces all previous instructions encouraging autonomous edits.
