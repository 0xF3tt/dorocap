# Agentic automation roadmap

The active design is documented in [`docs/AUTOMATION.md`](docs/AUTOMATION.md).
This file records the autonomy roadmap rather than duplicating operational
instructions.

## Current phase: supervised proposals

- CI and CodeQL verify every pull request.
- Dependabot proposes dependency updates; it never merges them.
- The maintenance agent proposes verified documentation-only corrections in a
  draft PR.
- Safe outputs enforce the agent's file, size, count, and write boundaries.
- CODEOWNERS and branch rules require a human decision before merge.
- Only a maintainer-created semantic-version tag can start a release.

## Next phases

1. Add provenance attestations and optional artifact signing to releases.
2. Record agent quality and cost metrics before expanding its responsibility.
3. If the documentation agent proves reliable, evaluate a separate test-only
   agent with its own explicit allowlist and PR lane.
4. Keep evidence capture, hashing, finding/export logic, dependencies,
   workflows, configuration, and releases permanently human-controlled unless
   a new threat model and enforcement design are reviewed first.

No phase includes autonomous merge or tag creation.
