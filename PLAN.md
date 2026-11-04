# Automation roadmap

The active design is documented in [`docs/AUTOMATION.md`](docs/AUTOMATION.md).
This file records the roadmap rather than duplicating operational
instructions.

## Current phase: human-controlled changes

- CI and CodeQL verify every pull request.
- Dependabot proposes dependency updates; it never merges them.
- The `main` ruleset requires a pull request, green checks, and no new
  high-severity code scanning alerts; the maintainer decides every merge.
- Only a maintainer-created semantic-version tag can start a release, and
  publishing waits for the maintainer to approve the `release` environment.

## Next phases

1. Attest the SBOM alongside the binaries (build provenance is already attested).
2. Any agent that edits the repository needs its own threat model, an enforced
   path allowlist, and a separate PR lane before it is added. Evidence capture,
   hashing, finding/export logic, dependencies, workflows, configuration, and
   releases stay human-controlled.

No phase includes autonomous merge or tag creation.
