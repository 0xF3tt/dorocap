# DevSecOps workflow

This repository follows the workspace `DevSecOps Operating Model v1` maintained in Linear, alongside the repository-specific controls documented in `docs/AUTOMATION.md` and `SECURITY.md`.

## Systems of record

- **Linear:** work, risk, prioritization, decisions, remediation evidence, and retest status.
- **GitHub:** source code, branches, pull requests, automated checks, and release history.

Linear project: **0xF3tt · Dorocap**  
Repository routing label: **dorocap**

## Change traceability

Material changes should follow:

`Source / Requirement / Finding → Linear Issue → Pull Request → CI / Security Review → Merge → Release / Verification → Done`

Use the Linear identifier in the PR title or description. Prefer:

- `Fixes 0XF-123` when merge should complete the issue according to Linear Git automation.
- `Refs 0XF-123` when the PR contributes to an issue but should not complete it.

## GitHub Issues routing

`.github/workflows/auto-label-issues.yml` adds the `dorocap` label to new GitHub Issues. GitHub Issues Sync plus the Linear Triage rule can then route them to **0xF3tt · Dorocap**.

## Dependabot policy

- Runs weekly on Monday in `America/Monterrey`.
- GitHub Actions minor/patch updates are grouped.
- Go module minor/patch updates are grouped.
- Major updates remain individual PRs for focused review.
- Security updates remain risk-classified and explicitly traceable in Linear.

`.github/workflows/dependabot-linear-sync.yml` periodically discovers open Dependabot PRs without a Linear identifier, creates the corresponding issue directly in **0xF3tt · Dorocap**, and adds the returned identifier to the PR title.

### Required secret

Create the repository Actions secret:

`LINEAR_API_KEY`

Use a dedicated Linear personal API key for this automation. The workflow uses a scheduled/manual event rather than a Dependabot-PR event because GitHub restricts secrets and token permissions for workflows initiated by Dependabot.

If the secret is missing, the workflow exits successfully without creating or modifying issues.

## Existing security controls

Dorocap already has a mature repository security baseline, including:

- cross-platform build and test gates
- minimum supported Go compatibility
- formatting, vet, static analysis, and linting
- `gosec`
- `govulncheck`
- CodeQL for Go and GitHub Actions
- human-controlled merges and releases
- CycloneDX SBOM generation on release
- SHA-256 release checksums
- restricted automation permissions and agent safe-output boundaries

See `docs/AUTOMATION.md` for the authoritative repository automation model.

## Definition of Done

An applicable Linear issue is complete only when scope is satisfied, required checks pass or an explicit exception exists, implementation/release evidence is linked, and security remediation has been retested when required.
