# Coding conventions

## Existing style is authoritative

Before modifying code:
- Inspect nearby files and similar implementations first.
- Match the existing architecture, naming, formatting, control-flow style,
  abstraction level, error-handling patterns, and file organization.
- Prefer consistency with the repository over generic best practices.
- Do not refactor unrelated code merely to match your preferences.

## Comments

Match the comment density and style of the surrounding code.

- If nearby code is mostly self-explanatory, avoid adding comments.
- Do not add comments that merely restate the code.
- Add comments only for non-obvious invariants, unusual constraints,
  workarounds, subtle edge cases, or important architectural reasoning.
- Preserve existing comments unless they become inaccurate.
- Match the tone and formatting of existing comments.

## Implementation behavior

For implementation/fix requests:
- Inspect relevant code first.
- Implement the complete change without asking for confirmation for ordinary
  local edits.
- Fix failures caused by your changes.
- Continue until the requested task is complete or genuinely blocked.
- Do not stop after merely explaining what should be changed.

## Scope

- Make the smallest coherent change that fully solves the task.
- Avoid speculative abstractions.
- Reuse existing utilities and patterns where appropriate.
- Do not introduce a new dependency unless there is a strong reason.

## Verification

Before finishing:
- Review the diff.
- Check for regressions and edge cases.
- Remove debugging code and temporary artifacts.
