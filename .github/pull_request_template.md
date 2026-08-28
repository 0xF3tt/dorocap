<!-- File: .github/pull_request_template.md -->

## Linear traceability

- **Linear issue:** `0XF-___`
- **Relationship:** `Fixes` / `Refs`
- **Work type:** Feature / Bug / Improvement / Security / Dependencies / Maintenance / Research / Automation / Compliance / Hardening

> Material changes should include the Linear issue ID in the PR title or description so implementation and work tracking remain linked.

## Plan (required)
- **Goal:**
- **Scope (paths/files):**
- **Steps:**
  1.
  2.
  3.
- **Success criteria (verifiable):**
  - [ ] `make test` passes
  - [ ] `make lint` passes
  - [ ] `make audit` passes (if touching dependencies or security-sensitive code)
- **Risks + mitigations:**
- **Rollback / escalation plan:**

## Security assessment

- **Risk level:** Low / Medium / High / Critical
- **Security impact:** None / Potential / Confirmed

Check all that apply:

- [ ] Evidence integrity, hashing, path/symlink handling, or file permissions changed
- [ ] Sensitive engagement/report data handling changed
- [ ] Native command execution or shell interaction changed
- [ ] Dependency or supply-chain change
- [ ] GitHub Actions, permissions, release, or build pipeline changed
- [ ] New trust boundary or privileged operation introduced
- [ ] Threat model / abuse cases reviewed when required
- [ ] No security trigger applies

## Evidence
- Workflow run(s):
- Security findings / exceptions:
- Notes:

## Review checklist
- [ ] Plan reviewed
- [ ] Diff stays within the agreed scope (see `.github/agents/*.agent.md` when applicable)
- [ ] Required checks satisfied or an explicit exception exists
- [ ] Relevant Linear issue contains required evidence/links
- [ ] Retest completed when this PR remediates a security finding
