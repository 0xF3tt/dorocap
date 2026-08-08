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

- Evidence files and sidecars are created with owner-only file permissions;
  engagement directories allow traversal only to the owner and group.
- Screenshots are staged in a private system-temporary directory and then
  copied into the engagement with an atomic same-directory rename. Temporary
  capture data is removed after success or failure.
- Imported and captured evidence receives a SHA-256 sidecar. `dorocap verify`
  detects missing, modified, orphaned, or incorrectly linked evidence, but a
  checksum is not a signed chain-of-custody record.
- Finding links resolve symlinks before containment-checking against the
  engagement evidence root, reject sidecar files, and require a valid checksum
  before being recorded. Vault verification reports symlink entries as invalid
  evidence.
- `dorocap finalize` re-verifies the evidence vault before copying the reviewed
  draft into `report/final/`; it does not regenerate or silently replace edits
  made to the draft report.
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
