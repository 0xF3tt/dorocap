# Security Policy

## Supported versions

dorocap is pre-1.0. Only the latest release gets fixes, so update before you
report an issue. Only macOS and Linux are supported.

## Reporting a vulnerability

Report vulnerabilities privately through
[GitHub Security Advisories](https://github.com/0xF3tt/dorocap/security/advisories/new),
not in a public issue. Include the steps to reproduce and the affected version
or commit.

A bug matters most in evidence integrity, path and symlink handling, file
permissions, malformed engagement files, the screenshot tools dorocap runs,
and the release pipeline.

## Security model

- **Permissions.** dorocap creates folders with mode `0700` and files with mode
  `0600`, so other local users cannot list or read an engagement.
- **Capture.** Screenshots are written to a private temporary folder and moved
  into the engagement with an atomic rename. The temporary data is removed
  whether the capture succeeds or fails. On macOS dorocap runs
  `/usr/sbin/screencapture`. On Linux it runs `import` only if that binary
  identifies itself as ImageMagick.
- **Hashes, not signatures.** Every capture gets a SHA-256 sidecar.
  `dorocap verify` detects missing, modified, orphaned, misfiled, or wrongly
  linked evidence. A checksum is not a signed chain-of-custody record.
- **Host name.** Each sidecar records the host name of the machine that made
  the capture, so delivering the evidence folder reveals that name.
- **Operator records.** Findings, notes, and `scope.yaml` are not hashed. A
  report that says verification passed does not vouch for the finding text.
- **Engagement lookup.** Commands search parent folders for `scope.yaml` and
  stop at your home folder. Outside your home folder, they also stop before a
  folder owned by another user. `dorocap path` refuses an engagement you do not
  own. Run `dorocap info` before capturing sensitive material to confirm where
  it will be saved.
- **Links.** `finding link` resolves symlinks before checking that the path is
  inside `evidence/`, refuses sidecars, and requires a valid hash. `verify`
  reports symlinks as invalid evidence.
- **Report checksum.** `export` writes `report/draft/report.md.sha256` only when
  verification passes. On failure it removes any old checksum and exits with
  an error. `finalize` verifies the evidence again and copies the draft only if
  it still matches that checksum. `finalize --accept-edits` records a new
  checksum for a hand-edited draft, and verification must still pass.
- **Relative links.** Reports link to evidence by relative path. Keep a report
  with its engagement folder, or send the evidence folder with it.
- **Completion.** Shell completion only reads findings and sidecars, and it
  offers only evidence that currently passes verification.

Treat engagement folders, reports, shell history, and completion output as
client data. Use the disk encryption, access controls, retention, and secure
deletion that the engagement's rules of engagement require.
