# The README GIF

`docs/dorocap-flow.gif` is a 22-second loop that shows how dorocap works, in
five chapters. It is drawn entirely in code from
[`dorocapFlow.ts`](dorocapFlow.ts), with no image assets.

| Chapter | On screen | README section |
|---|---|---|
| 01 init | `dorocap init acme-2026` builds the scope file, an evidence cabinet with one drawer per category, a findings board, and the draft and final trays | Quick start, Engagement folder |
| 02 capture | `ss exploitation`, then `ss file nmap.txt`: each capture drops into its drawer and leaves an orange SHA-256 tag (its sidecar) on the drawer front | Capture evidence |
| 03 finding | `finding add -i` pins a finding to the board; `finding link 0001 ...` ties it to the exploitation capture | Findings, Link evidence to a finding |
| 04 verify | a scanner bar sweeps the drawers and the finding; every tag and the link get a green check | Verify, export, and finalize |
| 05 report | `export` writes the draft with its own orange checksum tag; `finalize` copies that exact draft into the final tray and seals the copy | Verify, export, and finalize |

Orange marks hashes. Green marks what passed `verify` and the sealed final
report. The colors come from
[`docs/dorocap-data-flow.html`](../dorocap-data-flow.html).

Simplified on purpose:

- Notes are not shown.
- The link command is shortened to `finding link 0001 ...`.
- The seal stands for the copy in `report/final/`; dorocap does not stamp files.
- The last half second rewinds the scene so the loop restarts cleanly.

## Changing the GIF

`dorocapFlow.ts` draws any frame from its frame number alone. A cue table at
the top sets when each event happens, in frames at 30 fps, and is checked when
the module loads, so a mistimed cue fails right away.

The file imports the anidoodle engine (`core`, `film`, `drafting`,
`isometricKit`), which is not part of this repository, so rendering needs that
engine. The delivered GIF is 960×540 at 15 fps with one 256-color palette, and
weighs about 410 KB. Keep it under 500 KB, the limit of the pre-commit
`check-added-large-files` hook.
