# Security Policy

## Supported Versions

dorocap is pre-1.0. Only the latest release is supported — please update
before reporting an issue.

## Reporting a Vulnerability

Please report vulnerabilities privately via GitHub's
[Security Advisories](https://github.com/0xF3tt/dorocap/security/advisories/new)
rather than a public issue.

Include reproduction steps and the affected version/commit. Likely areas of
impact include evidence integrity, path or symlink handling, permissions,
malformed engagement data, native screenshot invocation, and the release/build
pipeline.

## Security Model

- Evidence files and sidecars are created with owner-only file permissions.
  Engagement directories are owner-only (`0700`) as well, so other local users
  cannot list evidence filenames. On Windows, directory ACLs follow the
  process defaults.
- Screenshots are staged in a private system-temporary directory and then
  copied into the engagement with an atomic same-directory rename. Temporary
  capture data is removed after success or failure. macOS capture runs
  `/usr/sbin/screencapture`. On Linux, `import` is used only when that
  absolute path identifies itself as ImageMagick.
- Imported and captured evidence receives a SHA-256 sidecar. The sidecar
  records the operator workstation hostname. Delivering the evidence tree
  therefore reveals that name. `dorocap verify` detects missing, modified,
  orphaned, incorrectly typed, or incorrectly linked evidence, but a checksum
  is not a signed chain-of-custody record.
- Findings, notes, and `scope.yaml` are operator records. They are not covered
  by the evidence checksum. A report that says verification passed is not a
  signature over the finding text.
- Commands that look for an engagement walk parent directories and stop at the
  home directory. Outside the home directory they also stop before entering a
  directory owned by someone else. `dorocap path` refuses a vault the current
  user does not own. Windows skips the owner check.
- Finding links resolve symlinks before containment-checking against the
  engagement evidence root, reject sidecar files, and require a valid checksum
  before being recorded. Vault verification reports symlink entries as invalid
  evidence.
- `dorocap export` writes `report/draft/report.md` and, only when evidence
  verification succeeds, `report/draft/report.md.sha256`. Verification failure
  still writes the draft so the warnings are visible, removes any previous
  checksum, and exits non-zero.
- `dorocap finalize` re-verifies the evidence vault and copies the draft into
  `report/final/` only when the draft bytes match that checksum. It does not
  regenerate the draft. After a hand edit, `dorocap finalize --accept-edits`
  records the edited bytes and then promotes them. Evidence verification still
  has to succeed.
- Generated Markdown reports use relative evidence links. Keep the report with
  its engagement directory, or include the evidence tree when transferring it,
  so linked evidence remains available.
- Shell completion reads finding metadata and evidence sidecars from the active
  engagement. It does not modify evidence. Dynamic evidence completion offers
  only files that currently pass verification.
- Terminal styling is disabled for redirected output, `TERM=dumb`,
  `NO_COLOR`, and legacy Windows CMD detection. Nerd Font private-use glyphs
  are opt-in so untrusted terminal/font assumptions do not alter default output.
- The global engagement path is local user configuration. Use `dorocap info`
  before capturing sensitive material to confirm the active destination.

Treat the engagement directory, generated reports, shell history, and terminal
completion output as potentially sensitive client data. Apply disk encryption,
access controls, retention requirements, and secure deletion procedures that
match the engagement's rules of engagement.
