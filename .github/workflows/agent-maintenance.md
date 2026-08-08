---
description: Weekly documentation drift review with a tightly scoped draft-PR output.

on:
  schedule: weekly on wednesday
  workflow_dispatch:

engine: copilot
strict: true
timeout-minutes: 30
max-ai-credits: 500
max-daily-ai-credits: 1000

models:
  default-ai-credits-pricing:
    input: 3.0
    output: 15.0

features:
  group-concurrency-queue: false

concurrency:
  group: agent-maintenance
  cancel-in-progress: false

imports:
  - .github/agents/maintenance.agent.md

permissions:
  contents: read
  copilot-requests: write
network:
  allowed: [defaults, go]

runtimes:
  go:
    version: "1.25.8"

tools:
  bash: ["go", "make"]

safe-outputs:
  create-pull-request:
    title-prefix: "[agent: docs] "
    labels: [agent, documentation]
    draft: true
    max: 1
    if-no-changes: ignore
    fallback-as-issue: false
    signed-commits: true
    allowed-files:
      - README.md
      - docs/**
    protected-files:
      policy: allowed
    max-patch-files: 10
    max-patch-size: 256
    github-token-for-extra-empty-commit: ${{ secrets.GH_AW_GITHUB_TOKEN }}

---

# Weekly documentation maintenance

Follow the imported `maintenance` agent instructions.

Compare the user-facing command documentation with the CLI implementation and
tests. Run the prescribed read-only health checks. Correct only verified
documentation drift in `README.md` or `docs/**`.

If nothing needs to change, produce no pull request. If a correction is needed,
open exactly one draft pull request using the required Plan, Evidence, Risks,
and Rollback sections. Never merge, approve, tag, release, or modify code.
