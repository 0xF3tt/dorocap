# dorocap

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.25-00ADD8.svg)

A pentest evidence and reporting helper. Captures screenshots and files into
a per-engagement evidence vault, tracks findings, and exports a markdown
report — all from the command line.

## Contents

- [Features](#features)
- [Requirements](#requirements)
- [Install](#install)
- [Usage](#usage)
- [Typical workflow](#typical-workflow)
- [Shell autocomplete](#shell-autocomplete)
- [Terminal compatibility](#terminal-compatibility)
- [scope.yaml](#scopeyaml)
- [Development](#development)
- [Automation](#automation)
- [Security](#security)
- [License](#license)

## Features

- Screenshot and file capture into a per-engagement, per-category evidence tree
- Atomic evidence capture with collision-resistant names and SHA-256 metadata
- Vault verification for missing, modified, orphaned, or incorrectly linked evidence
- Timestamped notes, keyed by category
- Lightweight findings tracker (plain markdown, no database)
- Context-aware Zsh, Bash, Fish, and PowerShell completion, including finding IDs and evidence paths
- Automatic global engagement selection when a new engagement is initialized
- Scope-aware markdown reports with severity totals, notes, evidence metadata, and warnings
- Cross-platform, obfuscated release builds (`garble`) via `make release`

## Requirements

- Go 1.25+ to build from source
- `dorocap ss <type>` (screenshot capture) shells out to a native tool:
  - macOS: `screencapture` (built in, no setup needed)
  - Linux: one of `scrot`, `gnome-screenshot`, or ImageMagick's `import`
  - Windows: native capture is not supported yet — use `dorocap ss file <path>` to import a
    screenshot taken by another tool instead

On macOS, allow your terminal application under **System Settings → Privacy &
Security → Screen & System Audio Recording**. Dorocap captures into a private
system-temporary directory before atomically committing the image to the
engagement, which avoids asking the native tool to write into a protected
Downloads folder directly.

## Install

```
make install
```

Builds and installs to `$(go env GOPATH)/bin`. Make sure that directory is
on your `PATH`.

## Usage

```
dorocap init <engagement-name>          scaffold a new engagement folder
dorocap ss <type> [note...]             capture a screenshot into evidence/<type>/
dorocap ss file <src-path> [note...]    copy a file into evidence/files/
dorocap note <type> <text...>           append a timestamped note
dorocap finding add --interactive
dorocap finding add <title> [--severity value] [--asset value] [--status value]
dorocap finding set <id> --interactive
dorocap finding set <id> <field> <value>
dorocap finding link <id> <evidence-path>
dorocap finding list
dorocap verify                          verify evidence files, sidecars, hashes, and finding links
dorocap export                          write the complete draft report to report/draft/report.md
dorocap finalize                        verify and promote the reviewed draft to report/final/report.md
dorocap path [<dir>]                    show or set the fallback engagement path
dorocap info                            show the active engagement and resolution source
dorocap completion <shell>              generate zsh, bash, fish, or PowerShell autocomplete
dorocap version                         show the installed version
```

### Initialize an engagement

```bash
dorocap init acme-2026
```

This creates the engagement structure and immediately makes its absolute path
the global default. Commands can then be run from any directory. Use
`dorocap info` to confirm which engagement is active, or `dorocap path <dir>` to
select a different existing engagement. After initialization, dorocap prints an
annotated directory tree explaining the purpose of the scope file, each
evidence category, notes, findings, and report directories, followed by the
next recommended action. On a color-capable terminal, the tree uses the Neust
palette for structure, paths, descriptions, and action prompts. `NO_COLOR`
keeps the complete summary in plain text.

### Capture or import evidence with `ss`

Capture a screenshot into one of the evidence categories:

```bash
dorocap ss exploitation "Admin session cookie obtained"
```

Screenshot categories are `recon`, `staging`, `exploitation`, and `postex`.
The trailing description is optional and is saved in the evidence metadata.

Import an existing file instead of taking a screenshot:

```bash
dorocap ss file ~/Downloads/nmap.txt "Initial port scan"
```

Every capture is stored with a `.json` sidecar containing its timestamp, host,
category, optional note, source filename for imports, and SHA-256 hash.

### Create and list findings

For a guided workflow, use interactive mode:

```bash
dorocap finding add --interactive
```

Dorocap prompts for each value individually:

```text
Title: SQL injection in login
Severity (crit/high/med/low/info) [info]: high
Status (...) [open]:
Affected asset (host, URL, IP, endpoint, or service): example.com/login
CVSS score (0.0-10.0): 8.8
CWE (for example CWE-89): CWE-89
OWASP category: A03:2021 Injection
Description: Unsanitized input reaches a SQL query.
Business impact: Account data may be exposed.
Technical impact: Database queries can be modified.
Steps to reproduce: Submit a quote in the username field.
Remediation: Use parameterized queries.
References: https://owasp.org/
```

Press Enter to accept a displayed default or skip an optional field. `-i` is a
short alias for `--interactive`. The non-interactive form remains available for
scripts and repeatable automation:

```bash
dorocap finding add "SQL injection in login" --severity high \
  --asset example.com/login --cvss 8.8 --cwe CWE-89 --owasp "A03:2021 Injection"
dorocap finding list
```

Supported severities are `crit`, `high`, `med`, `low`, and `info`. Severity
defaults to `info`; status defaults to `open`. `--asset` associates the
affected host, URL, endpoint, IP, or service with the finding—it is finding
metadata, not evidence. Optional creation fields are `--status`, `--cvss`,
`--cwe`, and `--owasp`.

Update metadata later with `finding set`:

```bash
dorocap finding set 0001 --interactive
dorocap finding set 0001 asset example.com/login
dorocap finding set 0001 status partially-resolved
dorocap finding set 0001 retest-status still-vulnerable
dorocap finding set 0001 retested 2026-08-20
```

Interactive mode shows every current metadata value. Press Enter to keep it,
or enter `-` to clear an optional value. The supported fields are severity,
status, asset, CVSS, CWE, OWASP, retest status, and retested date. The direct
form also accepts `-` to clear a value. Findings are editable Markdown files under
`findings/`; their generated sections cover description, business impact,
technical impact, reproduction, remediation, references, and retest notes.

### Link evidence to a finding

```bash
dorocap finding link 0001 evidence/exploitation/<captured-file>
```

`link` records that an evidence file supports a finding; it does not move or
duplicate the file. Dorocap rejects paths outside the evidence vault, sidecars,
unverified files, and duplicate links. Linked evidence is placed beneath the
corresponding finding when the report is exported.

Link evidence captured during a retest separately:

```bash
dorocap finding link 0001 evidence/exploitation/<retest-file> --retest
```

With shell completion enabled, select both values using Tab:

```text
dorocap finding link <Tab>       # finding IDs
dorocap finding link 0001 <Tab>  # verified, not-yet-linked evidence
```

### Add engagement notes

```bash
dorocap note recon "Ports 22, 80, and 443 are open"
```

Note categories are `recon`, `staging`, `exploitation`, `postex`, and `files`.
Notes are timestamped, stored under `notes/`, and included in the report.

### Verify and export

```bash
dorocap verify
dorocap export
```

`verify` checks sidecars, hashes, orphaned files, and finding links. `export`
writes the complete draft to `report/draft/report.md`, including scope,
findings, linked evidence, notes, and integrity warnings. PNG, JPEG, GIF, and
WebP evidence is embedded directly as a Markdown image; other evidence remains
a clickable link. Generated Markdown uses paths relative to the draft or final
report, such as `../../evidence/exploitation/screenshot.png`. Two parent segments
are required because reports live under `report/draft/` or `report/final/`.

The generated draft also includes cover metadata, document revision history,
an executive summary, severity distribution, linked risk summary, engagement
scope and restrictions, methodology, limitations, structured findings,
captioned evidence, remediation roadmap, notes, and integrity results.

After reviewing and editing the draft, promote it to the final directory:

```bash
dorocap finalize
```

Finalization verifies the current evidence vault and then copies the reviewed
`report/draft/report.md` to `report/final/report.md`. It never regenerates or
overwrites your reviewed draft content.

## Typical workflow

```bash
dorocap init acme-2026
dorocap ss file ~/Downloads/nmap.txt "Initial port scan"
dorocap finding add "Unnecessary service exposed" --severity med
dorocap finding link 0001 <Tab>
dorocap note recon "Service confirmed from the external network"
dorocap verify
dorocap export
# Review and edit report/draft/report.md
dorocap finalize
```

## Shell autocomplete

Load completion for the current shell session:

```bash
# Zsh
source <(dorocap completion zsh)

# Bash
source <(dorocap completion bash)

# Fish
dorocap completion fish | source

# PowerShell
dorocap completion powershell | Out-String | Invoke-Expression
```

To enable Zsh completion on every new terminal, add this line to `~/.zshrc`:

```zsh
source <(dorocap completion zsh)
```

For permanent Fish completion:

```fish
mkdir -p ~/.config/fish/completions
dorocap completion fish > ~/.config/fish/completions/dorocap.fish
exec fish
```

For permanent PowerShell completion, add this line to `$PROFILE`:

```powershell
dorocap completion powershell | Out-String | Invoke-Expression
```

Completion for Zsh, Bash, Fish, and PowerShell covers every command, finding
subcommand, evidence category, severity value, engagement directory, finding
ID, and verified evidence path. For example, after loading it, press Tab after
either argument:

```bash
dorocap finding link <Tab> <Tab>
```

The first completion lists finding IDs. After selecting an ID, the second lists
verified evidence that is not already linked to that finding. Fish suppresses
unrelated filesystem suggestions; files are offered only for `ss file`, and
directories only for `path`.

## Terminal compatibility

Dorocap selects output capabilities conservatively:

- Redirected output, `TERM=dumb`, `NO_COLOR`, and legacy Windows CMD receive no
  ANSI escapes.
- Terminals advertising `COLORTERM=truecolor`, `COLORTERM=24bit`, a direct-color
  `TERM`, or Windows Terminal receive the full Neust 24-bit palette.
- Other ANSI-capable terminals receive a readable 16-color approximation.
- Non-interactive, explicitly ASCII, and legacy Windows output uses ASCII tree
  and banner characters. Set `DOROCAP_ASCII=1` to force this mode.
- Nerd Font icons are opt-in. Set `DOROCAP_NERD=1` to enable them; plain `*` and
  `+` symbols are used by default. `DOROCAP_NO_NERD=1` always disables icons.

For maximum compatibility:

```bash
NO_COLOR=1 DOROCAP_ASCII=1 dorocap help
```

Every `init` scaffolds a `scope.yaml` marker and saves that engagement as the
global default, so `ss`/`note`/`finding`/`export` work from any directory.
The supported capture/note categories are `recon`, `staging`, `exploitation`,
and `postex`; imported files are stored under `files`.

Evidence is committed as a file plus a `.json` sidecar. Sidecars record the
capture time, host, category, note, source basename (for imports), and SHA-256.
Run `dorocap verify` before export or delivery. These checksums detect accidental
or unauthorized changes when compared with a trusted copy, but are not by
themselves a signed chain-of-custody system.

## scope.yaml

`dorocap init <name>` generates this template at the engagement root:

```yaml
# scope.yaml - engagement scope and rules of engagement
engagement: acme-2026
client: ""
assessment_type: "Penetration Test"
testing_mode: ""
classification: "Confidential"
report_version: "0.1"
prepared_by: "@0xF3tt"
start_date: ""
end_date: ""
executive_summary: ""
overall_posture: ""
in_scope:
  - ""
out_of_scope:
  - ""
contacts:
  - ""
restrictions:
  - ""
emergency_stop_conditions:
  - ""
limitations:
  - ""
methodology:
  - Reconnaissance
  - Enumeration
  - Vulnerability analysis
  - Exploitation
  - Post-exploitation
  - Evidence verification
  - Reporting
revision_history:
  - version: "0.1"
    date: ""
    author: "@0xF3tt"
    changes: "Initial draft"
```

Fill it in by hand, e.g.:

```yaml
# scope.yaml - engagement scope and rules of engagement
engagement: acme-2026
client: "Acme Corp"
assessment_type: "External Web Application Penetration Test"
testing_mode: "Gray-box"
classification: "Confidential"
report_version: "1.0"
prepared_by: "@0xF3tt"
start_date: "2026-07-10"
end_date: "2026-07-24"
executive_summary: "Acme requested an assessment of its external application."
overall_posture: "Elevated risk"
in_scope:
  - "*.acme.com"
  - "10.20.30.0/24"
out_of_scope:
  - "billing.acme.com"
  - "corporate VPN infrastructure"
contacts:
  - "Jane Doe <jane@acme.com> (primary technical contact)"
restrictions:
  - "No denial-of-service testing"
emergency_stop_conditions:
  - "Stop immediately if production availability degrades"
limitations:
  - "Payment processing was unavailable during testing"
methodology:
  - Reconnaissance
  - Enumeration
  - Vulnerability analysis
  - Exploitation
  - Post-exploitation
  - Evidence verification
  - Reporting
revision_history:
  - version: "1.0"
    date: "2026-07-25"
    author: "@0xF3tt"
    changes: "Final report"
```

It currently doubles as dorocap's own engagement-root marker (how `ss`/`note`/
`finding`/`export` find the right folder from a subdirectory) — the
`in_scope`/`out_of_scope` fields are documentation only, not yet enforced
against captured evidence.

## Development

```
make test    # unit tests
make lint    # vet + staticcheck + golangci-lint
make audit   # gosec + govulncheck
make tools   # install pinned lint, audit, release, and SBOM tools
make release # macOS/Linux binaries, checksums, and CycloneDX SBOM
```

See `make help` for the full target list. Before opening a PR, make sure
`make lint` and `make audit` are both clean and `make test` passes.

## Automation

CI, CodeQL, Dependabot, the documentation maintenance agent, release gates,
permissions, and repository setup are described in
[`docs/AUTOMATION.md`](docs/AUTOMATION.md).

## Security

dorocap is a local CLI that writes to disk with your own user permissions and
does not expose a network listener. Treat engagement folders and imported
evidence as sensitive data. See
[SECURITY.md](SECURITY.md) for how to report a vulnerability.

## License

MIT — see [LICENSE](LICENSE).
