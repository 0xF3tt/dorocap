// HOW DOROCAP WORKS. An isometric explainer, silent, looping, for the README.
//
// MEDIUM: flat vector planes on a true 2:1 isometric grid (isometricKit). Three tones per
// material (top lightest, +y face middle, +x face darkest), crisp edges, no gradients, no blur.
// Shadows are flat planes in a darker tone of the surface they fall on. Captions are single-stroke
// upright gothic capitals drawn as pen strokes (drafting.ts), never a font.
// ORDER: construction first. The grid is ruled, then massing boxes, then each piece of furniture
// is traced and filled, then details, then light and shadow. Actions follow in story order.
// LIGHT: one light, upper left, carried by the face tones; the verify sweep is the one moving
// light and is poured as flat bands.
// STORY (BRIEF.md): init builds the engagement; every capture lands in its drawer and leaves an
// orange SHA-256 tag; a finding is pinned and linked to its evidence; verify sweeps the vault and
// badges every tag; export writes the draft with its own orange checksum tag; finalize copies that
// exact draft to the final tray and seals the copy. TOKEN: the orange SHA-256 tag. TURN: the sweep.
// REALISM (from knowledge): a laptop is a thin base plus a screen panel hinged at its back edge;
// a filing cabinet has full-width drawers that slide out on runners, a pull and a label plate;
// an office in-tray is a low open box with walls on four sides.
// Palette from docs/dorocap-data-flow.html: paper #f3f1ec, ink #2a312e, muted #5c665f,
// accent #d86a2a (hash, verify), report #2f6b45.
import type { Ctx, Env, Medium } from "./core";
import { Gfx } from "./core";
import type { Film } from "./film";
import { letter } from "./drafting";
import { Iso, type Mat, type P2, type P3, clamp01, ease, span } from "./isometricKit";

const FPS = 30, BPM = 120, BEAT = (60 / BPM) * FPS; // 15 frames

// ---------------------------------------------------------------- the cue table (every frame number lives here)
const CH = { init: 0, capture: 105, finding: 255, verify: 360, report: 450, hold: 585, rewind: 645, end: 660 } as const;
const DURATION = CH.end;
type R = [number, number];
const E = {
  // 01 init: the engagement is built
  grid: [0, 20], plinth: [5, 30], mass: [15, 35], termBase: [25, 40], termPanel: [35, 50], termScreen: [45, 55],
  scope: [40, 55], scopeLines: [55, 65], vault: [45, 65], drawers: [60, 85], hardware: [75, 90], board: [55, 70],
  trays: [65, 85], keys: [80, 95], shadows: [85, 100], labels: [90, 105], line0: [55, 65],
  // 02 capture: ss exploitation, then ss file
  line1: [130, 140], card1Make: [135, 140], card1Fly: [140, 165], d2Open: [150, 165], card1Drop: [165, 175], d2Close: [175, 190], tag1: [185, 195], hash1: [190, 200],
  line2: [220, 230], card2Make: [220, 225], card2Fly: [225, 245], d0Open: [235, 245], card2Drop: [245, 250], d0Close: [250, 260], tag2: [255, 265], hash2: [260, 270],
  // 03 finding: add, then link
  line3: [275, 285], fCard: [275, 295], fStripe: [290, 300], fLines: [295, 310], fPin: [300, 305],
  line4: [325, 335], thread: [325, 350], rings: [345, 350],
  // 04 verify: the sweep down the vault, then across the finding
  line5: [375, 385], barIn: [380, 385], sweep: [385, 430], barOut: [430, 435], badge2: [405, 415], badge0: [425, 435],
  boardSweep: [425, 440], linkBadge: [435, 445], lineOk: [445, 455],
  // 05 report: export, then finalize
  line6: [465, 475], sheets: [470, 500], sheetText: [495, 505], csTag: [500, 510], csHash: [505, 515],
  line7: [530, 540], slide: [535, 560], seal: [560, 570], sealTick: [570, 580],
} satisfies Record<string, R>;

// captions: one step label, one or two commands typed in turn, one meaning line, per chapter
type Txt = { s: string; w: R; e?: R };
type Cap = { step: Txt; cmd: Txt[]; sub: Txt };
const CAPS: Cap[] = [
  { step: { s: "01  INIT", w: [0, 10], e: [95, 100] }, cmd: [{ s: "DOROCAP INIT ACME-2026", w: [5, 30], e: [95, 100] }],
    sub: { s: "SCOPE, EVIDENCE VAULT, FINDINGS, REPORTS", w: [30, 55], e: [95, 100] } },
  { step: { s: "02  CAPTURE", w: [105, 115], e: [245, 250] }, cmd: [{ s: "DOROCAP SS EXPLOITATION", w: [110, 130], e: [190, 195] }, { s: "DOROCAP SS FILE NMAP.TXT", w: [200, 220], e: [245, 250] }],
    sub: { s: "EVERY CAPTURE GETS A SHA-256 SIDECAR", w: [130, 150], e: [245, 250] } },
  { step: { s: "03  FINDING", w: [255, 265], e: [350, 360] }, cmd: [{ s: "DOROCAP FINDING ADD -I", w: [260, 275], e: [300, 305] }, { s: "DOROCAP FINDING LINK 0001 ...", w: [310, 325], e: [350, 360] }],
    sub: { s: "A MARKDOWN FINDING, LINKED TO ITS EVIDENCE", w: [275, 295], e: [350, 360] } },
  { step: { s: "04  VERIFY", w: [360, 370], e: [440, 445] }, cmd: [{ s: "DOROCAP VERIFY", w: [365, 375], e: [440, 445] }],
    sub: { s: "EVERY HASH, SIDECAR AND LINK IS CHECKED", w: [375, 395], e: [440, 445] } },
  { step: { s: "05  REPORT", w: [450, 460] }, cmd: [{ s: "DOROCAP EXPORT", w: [455, 465], e: [510, 515] }, { s: "DOROCAP FINALIZE", w: [515, 530] }],
    sub: { s: "DRAFT + CHECKSUM, THEN AN EXACT COPY GOES FINAL", w: [465, 490] } },
];
// terminal lines: the command length sets the bar length; the last one is verify's green OK
const LINES: { at: R; len: number; ok?: boolean }[] = [
  { at: E.line0, len: 1.15 }, { at: E.line1, len: 1.2 }, { at: E.line2, len: 1.25 }, { at: E.line3, len: 1.0 },
  { at: E.line4, len: 1.3 }, { at: E.line5, len: 0.75 }, { at: E.lineOk, len: 0.45, ok: true }, { at: E.line6, len: 0.75 }, { at: E.line7, len: 0.85 },
];
// The grid is part of the contract: chapter cuts on the beat, every event on a multiple of 5.
(() => {
  const bad: string[] = [];
  Object.entries(CH).forEach(([k, f]) => { if (f % BEAT || f > DURATION) bad.push(`chapter ${k}=${f}`); });
  const ranges: [string, R][] = [...Object.entries(E) as [string, R][], ...CAPS.flatMap((c, i) => [c.step, ...c.cmd, c.sub].flatMap((t, j) => [[`cap${i}.${j}w`, t.w] as [string, R], ...(t.e ? [[`cap${i}.${j}e`, t.e] as [string, R]] : [])]))];
  ranges.forEach(([k, [a, b]]) => { if (a % 5 || b % 5 || b <= a || b > CH.rewind) bad.push(`${k}=[${a},${b}]`); });
  if (bad.length) throw new Error(`dorocapFlow cue table off the grid: ${bad.join(", ")}`);
})();

// ---------------------------------------------------------------- palette
const PAPER = "#f3f1ec", INK = "#2a312e", MUTED = "#5c665f", SOFT = "#d8d3c8", ACCENT = "#d86a2a", GREEN = "#2f6b45";
const GRIDLINE = "#e4dfd3", MASS = "#6f8cc4";
const M: Record<string, Mat> = {
  plinth: { top: "#e9e4d8", left: "#d2cbbb", right: "#b7af9e" },
  body: { top: "#3e4944", left: "#2a312e", right: "#1c221f" },     // terminal
  steel: { top: "#d9d4c8", left: "#bdb7aa", right: "#9d968a" },    // vault cabinet
  drawer: { top: "#e6e2d8", left: "#cbc5b8", right: "#aaa396" },   // drawer fronts, a step lighter than the cabinet
  paper: { top: "#fffdf8", left: "#e9e3d7", right: "#d3cbbb" },    // cards, sheets, scope
  tag: { top: "#f2b183", left: ACCENT, right: "#a34c16" },          // the SHA-256 sidecar: the token
  board: { top: "#d6ccb6", left: "#c4b698", right: "#a0937a" },    // findings pinboard
  tray: { top: "#cdd6cf", left: "#adbbb1", right: "#8d9c92" },     // report trays
  seal: { top: "#7eae92", left: GREEN, right: "#1d4a2e" },
  key: { top: "#4b5751", left: "#39433e", right: "#2a322e" },
};
const SHADOW = "#d9d3c5";       // the plinth top with the shadow on it: an opaque flat tone, no alpha
const SCREEN = "#16201c", TEXT = "#9fd0ae", OKTXT = "#63c487", SLOT = "#4d483f", HOLLOW = "#8f887b", HOLLOW_WALL = "#a49d90";
const SEV = "#b8432f";           // high severity stripe on the finding card
const LIGHT = "#ffe1c2", LIGHT_CORE = "#fff1e2";

const CRISP: Medium = { nib: 1, taper: 0, pressure: 0, retrace: false, wobble: 0, rough: 0 };

// ---------------------------------------------------------------- world layout (units; x to screen lower-right, y lower-left, z up)
const U = 36, OX = 568, OY = 252;
const PL = { x0: 0, x1: 13.5, y0: 0, y1: 9.5, z0: -0.6 };
const TERM = { x0: 1.2, x1: 4.0, y0: 5.9, y1: 8.7, base: 0.22, px1: 1.42, top: 2.35 };
const SCR = { x: 1.43, y0: 6.22, y1: 8.38, z0: 0.45, z1: 2.15 };
const SCOPE = { x0: 0.6, x1: 2.8, y0: 2.4, y1: 4.4 };
const V = { x0: 4.8, x1: 8.4, y0: 0.8, y1: 3.8, z1: 3.5, fx0: 5.02, fx1: 8.18, depth: 0.14 };
const VC = (V.x0 + V.x1) / 2;
const dz0 = (k: number) => 0.22 + k * 0.66, dz1 = (k: number) => dz0(k) + 0.56, dzc = (k: number) => dz0(k) + 0.28;
const OPEN = 1.35;
const BOARD = { x0: 9.4, x1: 12.9, y0: 0.8, y1: 1.05, z1: 3.2 };
const CARD = { x0: 10.0, x1: 12.4, z0: 1.1, z1: 2.7, y: 1.065 };
const TRAY = { y0: 6.2, y1: 8.4, h: 0.42, t: 0.12, draft: [5.6, 8.2] as R, final: [8.8, 11.4] as R }; // trays side by side: draft left, final right
const STACK = { x0: 5.95, x1: 7.85, y0: 6.55, y1: 8.05 };
const SLIDE = TRAY.final[0] - TRAY.draft[0];
const TAGW = 0.44, TAGX = VC + 0.62;                         // tag slots on a drawer front, right of the pull

// ---------------------------------------------------------------- small pure helpers
const hex = (h: string) => [1, 3, 5].map((i) => parseInt(h.slice(i, i + 2), 16));
const lerp = (a: number, b: number, t: number) => a + (b - a) * t;
const L3 = (a: P3, b: P3, t: number): P3 => [lerp(a[0], b[0], t), lerp(a[1], b[1], t), lerp(a[2], b[2], t)];
// write between w, unwrite between e: a caption is made and unmade stroke by stroke, never faded
const prog = (t: number, w: R, e?: R) => (e && t >= e[0] ? 1 - span(t, e[0], e[1]) : span(t, w[0], w[1]));

const draw = (ctx: Ctx, local: number, env: Env) => {
  // The story clock. The last second rewinds the whole build so frame 659 leads back into frame 0.
  // It rewinds at constant speed from the last event (the seal's tick), not from the hold, so its
  // first frame already moves and its last frame is one step short of frame 0.
  const t = local < CH.rewind ? local : E.sealTick[1] * (1 - (local - CH.rewind + 1) / (CH.end - CH.rewind));
  const at = (r: R) => span(t, r[0], r[1]);
  ctx.setTransform(env.scale, 0, 0, env.scale, 0, 0);
  ctx.fillStyle = PAPER; ctx.fillRect(0, 0, env.W, env.H);
  const iso = new Iso(ctx, U, OX, OY);
  const P = (x: number, y: number, z: number) => iso.p(x, y, z);
  const box = (p: number, x0: number, y0: number, z0: number, x1: number, y1: number, z1: number, m: Mat) => { if (p > 0) iso.make(iso.boxFaces(x0, y0, z0, x1, y1, z1, m), p, m.right); };
  const plateY = (p: number, x0: number, x1: number, z0: number, z1: number, y: number, fill: string, edge = fill) => { if (p > 0) iso.make([{ pts: [P(x0, y, z0), P(x1, y, z0), P(x1, y, z1), P(x0, y, z1)], fill }], p, edge); };
  const plateX = (p: number, y0: number, y1: number, z0: number, z1: number, x: number, fill: string, edge = fill) => { if (p > 0) iso.make([{ pts: [P(x, y0, z0), P(x, y1, z0), P(x, y1, z1), P(x, y0, z1)], fill }], p, edge); };
  const plateZ = (p: number, x0: number, x1: number, y0: number, y1: number, z: number, fill: string, edge = fill) => { if (p > 0) iso.make([{ pts: [P(x0, y0, z), P(x1, y0, z), P(x1, y1, z), P(x0, y1, z)], fill }], p, edge); };
  const seg = (a: P3, b: P3, p: number, color: string, w: number) => { if (p > 0) iso.line(P(...a), P(...L3(a, b, clamp01(p))), color, w); };

  // ---------- 1. the iso grid, ruled line by line across the paper
  const gp = at(E.grid); if (gp > 0) {
    const lines: [P3, P3][] = []; for (let k = -2; k <= 12; k++) lines.push([[-2, k, PL.z0], [16, k, PL.z0]]); for (let k = -2; k <= 16; k++) lines.push([[k, -2, PL.z0], [k, 12, PL.z0]]);
    const n = gp * lines.length; lines.forEach(([a, b], i) => seg(a, b, clamp01(n - i), GRIDLINE, 1));
  }

  // ---------- 2. the plinth: the engagement root everything stands on
  const pp = at(E.plinth);
  // the plinth's own shadow on the paper: one flat offset plane, swept in like any other plane
  if (pp > 0) iso.make([{ pts: ([[PL.x1 + 0.7, PL.y0 + 0.7, PL.z0], [PL.x1 + 0.7, PL.y1 + 0.7, PL.z0], [PL.x0 + 0.7, PL.y1 + 0.7, PL.z0], [PL.x0 + 0.7, PL.y1, PL.z0], [PL.x1, PL.y1, PL.z0], [PL.x1, PL.y0 + 0.7, PL.z0]] as P3[]).map((q) => P(...q)), fill: "#e2ddd1" }], pp, "#e2ddd1");
  box(pp, PL.x0, PL.y0, PL.z0, PL.x1, PL.y1, 0, M.plinth);
  // board joints on the plinth top: sparse hairlines, laid with the details
  const jp = at(E.keys); if (jp > 0) { for (let k = 1; k < 6; k++) seg([k * 2.5, PL.y0, 0.001], [k * 2.5, PL.y1, 0.001], clamp01(jp * 6 - k), "#e0dacb", 1); for (let k = 1; k < 4; k++) seg([PL.x0, k * 2.5, 0.001], [PL.x1, k * 2.5, 0.001], clamp01(jp * 6 - k - 1), "#e0dacb", 1); }

  // ---------- 3. flat shadows on the plinth top, thrown to +x by the upper-left light
  const sp = at(E.shadows);
  const shadow = (x0: number, y0: number, x1: number, y1: number, h: number) => { if (sp <= 0) return; const dx = h * 0.42, dy = h * 0.1; plateZ(sp, x0 + 0.05, x1 + dx, y0 + dy, y1 + dy * 0.3, 0.001, SHADOW); };
  shadow(TERM.x0, TERM.y0, TERM.x1, TERM.y1, TERM.top * 0.6); shadow(V.x0, V.y0, V.x1, V.y1, V.z1); shadow(BOARD.x0, BOARD.y0, BOARD.x1, BOARD.y1 + 0.3, BOARD.z1);
  shadow(TRAY.draft[0], TRAY.y0, TRAY.draft[1], TRAY.y1, TRAY.h); shadow(TRAY.final[0], TRAY.y0, TRAY.final[1], TRAY.y1, TRAY.h);

  // ---------- 4. massing boxes: translucent construction volumes, gone once the real piece is made
  const mp = at(E.mass);
  const mass = (done: number, x0: number, y0: number, z0: number, x1: number, y1: number, z1: number) => {
    if (mp <= 0 || done >= 1) return; const f = iso.boxFaces(x0, y0, z0, x1, y1, z1, { top: MASS, left: MASS, right: MASS });
    f.forEach((q) => { const L = q.pts.length; for (let i = 0; i < L; i++) { const a = q.pts[i], b = q.pts[(i + 1) % L], e = clamp01(mp * 1.4); iso.line(a, [lerp(a[0], b[0], e), lerp(a[1], b[1], e)], MASS, 1, 0.5); } });
  };
  mass(at(E.scope), SCOPE.x0, SCOPE.y0, 0, SCOPE.x1, SCOPE.y1, 0.04);
  mass(at(E.vault), V.x0, V.y0, 0, V.x1, V.y1, V.z1);
  mass(at(E.board), BOARD.x0, BOARD.y0, 0, BOARD.x1, BOARD.y1, BOARD.z1);
  mass(at(E.termPanel), TERM.x0, TERM.y0, 0, TERM.x1, TERM.y1, TERM.top);
  mass(at(E.trays), TRAY.draft[0], TRAY.y0, 0, TRAY.final[1], TRAY.y1, TRAY.h);

  // ---------- 5. scope.yaml: the sheet that marks the engagement root
  const scp = at(E.scope); box(scp, SCOPE.x0, SCOPE.y0, 0, SCOPE.x1, SCOPE.y1, 0.04, M.paper);
  const slp = at(E.scopeLines); if (slp > 0) [0.95, 0.7, 0.85, 0.55, 0.9].forEach((len, i) => seg([SCOPE.x0 + 0.3, SCOPE.y0 + 0.55 + i * 0.3, 0.041], [SCOPE.x0 + 0.3 + len * 1.6, SCOPE.y0 + 0.55 + i * 0.3, 0.041], clamp01(slp * 5 - i), "#b9b2a3", 1.4));
  box(clamp01(slp * 2), SCOPE.x0 + 0.75, SCOPE.y0 - 0.05, 0.04, SCOPE.x0 + 1.45, SCOPE.y0 + 0.17, 0.16, M.body);

  // ---------- 6. the findings board, at the back right (behind the thread, so drawn early)
  const bp = at(E.board);
  box(clamp01(bp * 2), BOARD.x0 + 0.35, BOARD.y0 - 0.25, 0, BOARD.x0 + 0.65, BOARD.y1 + 0.35, 0.18, M.body);
  box(clamp01(bp * 2), BOARD.x1 - 0.65, BOARD.y0 - 0.25, 0, BOARD.x1 - 0.35, BOARD.y1 + 0.35, 0.18, M.body);
  box(clamp01(bp * 1.5 - 0.4), BOARD.x0, BOARD.y0, 0.18, BOARD.x1, BOARD.y1, BOARD.z1, M.board);
  // the finding card: traced, filled, its severity stripe, its text, its pin
  const fc = at(E.fCard); plateY(fc, CARD.x0, CARD.x1, CARD.z0, CARD.z1, CARD.y, "#fffaf1", "#d8cfbf");
  plateY(at(E.fStripe), CARD.x0, CARD.x1, CARD.z1 - 0.26, CARD.z1, CARD.y + 0.001, SEV);
  const flp = at(E.fLines); if (flp > 0) [0.85, 0.65, 0.78, 0.5].forEach((len, i) => seg([CARD.x0 + 0.25, CARD.y + 0.002, CARD.z1 - 0.55 - i * 0.26], [CARD.x0 + 0.25 + len * 2.1, CARD.y + 0.002, CARD.z1 - 0.55 - i * 0.26], clamp01(flp * 4 - i), "#8c8577", 1.6));
  const pin: P3 = [CARD.x0 + 0.28, CARD.y + 0.02, CARD.z1 - 0.13];
  const ring = (q: P2, r: number, p: number, fill: boolean) => { if (p <= 0) return; ctx.strokeStyle = INK; ctx.lineWidth = 2; ctx.beginPath(); ctx.arc(q[0], q[1], r, -Math.PI / 2, -Math.PI / 2 + Math.PI * 2 * clamp01(p * 1.25)); ctx.stroke(); if (fill && p >= 1) { ctx.fillStyle = INK; ctx.beginPath(); ctx.arc(q[0], q[1], r, 0, Math.PI * 2); ctx.fill(); } };
  ring(P(...pin), 4.2, at(E.fPin), true);

  // ---------- 7. the evidence vault: cabinet, five drawers, pulls, label plates, tags
  const vp = at(E.vault); box(vp, V.x0, V.y0, 0, V.x1, V.y1, V.z1, M.steel);
  if (vp >= 1) { plateZ(1, V.x0 + 0.15, V.x1 - 0.15, V.y0 + 0.15, V.y1 - 0.15, V.z1 + 0.001, "#e3dfd5"); plateY(1, V.x0, V.x1, 0, 0.13, V.y1 + 0.001, "#7d776c"); plateX(1, V.y0, V.y1, 0, 0.13, V.x1 + 0.001, "#625d55"); }
  const drawersP = at(E.drawers), hw = at(E.hardware);
  const openAmt = (k: number) => k === 2 ? OPEN * (ease(at(E.d2Open)) - ease(at(E.d2Close))) : k === 0 ? OPEN * (ease(at(E.d0Open)) - ease(at(E.d0Close))) : 0;
  // tags on a drawer: [make range, hash range, badge range]
  const TAGS: Record<number, [R, R, R][]> = { 2: [[E.tag1, E.hash1, E.badge2]], 0: [[E.tag2, E.hash2, E.badge0]] };
  // the two captures. Each flies from the screen, drops into its drawer, and rides it shut.
  type Cap3 = { k: number; make: R; fly: R; drop: R; kind: "shot" | "file" };
  const CAPTURES: Cap3[] = [{ k: 2, make: E.card1Make, fly: E.card1Fly, drop: E.card1Drop, kind: "shot" }, { k: 0, make: E.card2Make, fly: E.card2Fly, drop: E.card2Drop, kind: "file" }];
  const CW = 1.5, CD = 1.0, CT = 0.05;
  const cardAt = (c: Cap3): { p: P3; made: number; inside: boolean } | null => {
    const made = at(c.make); if (made <= 0) return null;
    const start: P3 = [SCR.x + 0.25, 6.8, 1.15], hover: P3 = [VC - 0.75, V.y1 + OPEN - 1.15, dz1(c.k) + 1.0], rest: P3 = [VC - 0.75, V.y1 + OPEN - 1.15, dz0(c.k) + 0.12];
    const f = ease(at(c.fly)), d = ease(at(c.drop));
    if (d <= 0) { const q = L3(start, hover, f); q[2] += Math.sin(Math.PI * f) * 1.6; return { p: q, made, inside: false }; }
    const q = L3(hover, rest, d); return { p: q, made, inside: d >= 1 };
  };
  const drawCard = (c: Cap3, p: P3, made: number, clipY = -Infinity) => {
    const [x, y, z] = p, y0 = Math.max(y, clipY), y1 = y + CD; if (y1 <= y0) return;
    box(made, x, y0, z, x + CW, y1, z + CT, M.paper);
    if (made < 1) return; const zt = z + CT + 0.001;
    if (c.kind === "shot") { // a screenshot: window title bar, a viewport block, two text bars
      if (y + 0.24 > y0) plateZ(1, x + 0.08, x + CW - 0.08, Math.max(y0, y + 0.08), y + 0.24, zt, INK);
      if (y + 0.62 > y0) plateZ(1, x + 0.12, x + 0.8, Math.max(y0, y + 0.34), y + 0.62, zt, "#cfd8e3");
      [[0.72, 0.9], [0.84, 0.6]].forEach(([yy, l]) => { if (y + yy > y0) seg([x + 0.12, y + yy, zt], [x + 0.12 + l, y + yy, zt], 1, "#b9b2a3", 1.6); });
    } else { // an imported text file: lines of output and a folded corner
      [0.2, 0.36, 0.52, 0.68, 0.84].forEach((yy, i) => { if (y + yy > y0) seg([x + 0.14, y + yy, zt], [x + 0.14 + [1.0, 0.75, 1.1, 0.6, 0.9][i], y + yy, zt], 1, "#9d968a", 1.5); });
      if (y + 0.3 > y0) iso.poly([P(x + CW - 0.3, Math.max(y0, y), zt), P(x + CW, Math.max(y0, y), zt), P(x + CW, y + 0.3, zt)], M.paper.right);
    }
  };
  // closed drawers top to bottom, then any open drawer last: pulled out, it sits in front of the rest
  const order = [4, 3, 2, 1, 0].sort((a, b) => Number(openAmt(a) > 0.001) - Number(openAmt(b) > 0.001));
  for (const k of order) {
    const dp = clamp01(drawersP * 5 - (4 - k) * 0.8); if (dp <= 0) continue;
    const o = openAmt(k), fy = V.y1 + o;
    if (o > 0.001) { // the drawer out on its runners: slot, hollow, the card inside, side wall
      plateY(1, V.fx0 + 0.08, V.fx1 - 0.08, dz0(k) + 0.04, dz1(k) - 0.06, V.y1 + 0.001, SLOT);
      plateZ(1, V.fx0 + 0.14, V.fx1 - 0.14, V.y1, fy - 0.06, dz0(k) + 0.1, HOLLOW);
      plateX(1, V.y1, fy - 0.06, dz0(k) + 0.1, dz1(k) - 0.06, V.fx0 + 0.14, HOLLOW_WALL);
    }
    CAPTURES.filter((c) => c.k === k).forEach((c) => { const s = cardAt(c); if (s && s.p[2] < dz1(k) - 0.05) { const q: P3 = [s.p[0], s.p[1] - (OPEN - o) * (s.inside ? 1 : 0), s.p[2]]; drawCard(c, q, s.made, V.y1); } });
    if (o > 0.001) plateX(1, V.y1, fy, dz0(k) + 0.04, dz1(k) - 0.06, V.fx1 - 0.08, M.drawer.right);
    box(dp, V.fx0, fy, dz0(k), V.fx1, fy + V.depth, dz1(k), M.drawer);
    const yf = fy + V.depth + 0.002;
    if (hw > 0) { // label plate left, pull in the middle
      plateY(clamp01(hw * 2 - (4 - k) * 0.2), V.fx0 + 0.22, V.fx0 + 0.95, dzc(k) - 0.13, dzc(k) + 0.13, yf, "#f6f3ec");
      box(clamp01(hw * 2 - (4 - k) * 0.2 - 0.3), VC - 0.4, yf, dzc(k) - 0.05, VC + 0.4, yf + 0.1, dzc(k) + 0.05, M.body);
    }
    (TAGS[k] ?? []).forEach(([mk, hs, bg], i) => { // the SHA-256 sidecar: an orange tag; verify adds a green badge
      const x0 = TAGX + i * TAGW * 2; box(at(mk), x0, yf, dzc(k) - 0.17, x0 + TAGW, yf + 0.07, dzc(k) + 0.17, M.tag);
      const hp = at(hs), ty = yf + 0.071; if (hp > 0) { const z0 = dzc(k) - 0.1, z1 = dzc(k) + 0.1, xa = x0 + 0.13, xb = x0 + 0.27; seg([xa, ty, z0], [xa + 0.03, ty, z1], hp * 4, PAPER, 1.6); seg([xb, ty, z0], [xb + 0.03, ty, z1], hp * 4 - 1, PAPER, 1.6); seg([x0 + 0.06, ty, z0 + 0.13], [x0 + 0.37, ty, z0 + 0.13], hp * 4 - 2, PAPER, 1.6); seg([x0 + 0.06, ty, z0 + 0.06], [x0 + 0.37, ty, z0 + 0.06], hp * 4 - 3, PAPER, 1.6); }
      const bp2 = at(bg); if (bp2 > 0) { const b0 = x0 + TAGW + 0.04; box(bp2, b0, yf, dzc(k) - 0.17, b0 + 0.34, yf + 0.07, dzc(k) + 0.17, M.seal); if (bp2 >= 1) { seg([b0 + 0.07, ty, dzc(k)], [b0 + 0.14, ty, dzc(k) - 0.09], 1, PAPER, 2); seg([b0 + 0.14, ty, dzc(k) - 0.09], [b0 + 0.28, ty, dzc(k) + 0.1], 1, PAPER, 2); } }
    });
    CAPTURES.filter((c) => c.k === k).forEach((c) => { const s = cardAt(c); if (s && at(c.drop) > 0 && s.p[2] >= dz1(k) - 0.05) drawCard(c, s.p, s.made); }); // dropping, still above the rim
  }
  // captures in flight that have not reached their drawer column yet are drawn on top of everything below
  // (they are higher than the vault top for most of the arc; drawCard above covers the rest)

  // ---------- 8. the terminal: base, keys, screen panel hinged at the back edge, the screen
  box(at(E.termBase), TERM.x0, TERM.y0, 0, TERM.x1, TERM.y1, TERM.base, M.body);
  const kp = at(E.keys); if (kp > 0) for (let r = 0; r < 5; r++) for (let c = 0; c < 9; c++) { if (clamp01(kp * 6 - r) <= 0) continue; const x = TERM.x0 + 0.65 + r * 0.38, y = TERM.y0 + 0.35 + c * 0.24; plateZ(1, x, x + 0.28, y, y + 0.18, TERM.base + 0.001, M.key.top); }
  box(at(E.termPanel), TERM.x0, TERM.y0 + 0.12, TERM.base, TERM.px1, TERM.y1 - 0.12, TERM.top, M.body);
  plateX(at(E.termScreen), SCR.y0, SCR.y1, SCR.z0, SCR.z1, SCR.x, SCREEN);
  if (at(E.termScreen) >= 1) { // the screen: prompt lines, newest at the bottom, scrolling up when full
    ctx.save(); const cl = [P(SCR.x, SCR.y0, SCR.z0), P(SCR.x, SCR.y1, SCR.z0), P(SCR.x, SCR.y1, SCR.z1), P(SCR.x, SCR.y0, SCR.z1)]; ctx.beginPath(); cl.forEach(([x, y], i) => (i ? ctx.lineTo(x, y) : ctx.moveTo(x, y))); ctx.closePath(); ctx.clip();
    // This screen faces +x, so its reader's right is -y: lines start at the high-y edge (screen
    // lower left) and run toward -y (screen upper right), and the prompt points the same way.
    const shown = LINES.filter((l) => t >= l.at[0]), n = shown.length, scroll = Math.max(0, n - 6), x = SCR.x + 0.001, ys = SCR.y1 - 0.2;
    let last: P3 = [x, ys, SCR.z1 - 0.25];
    shown.forEach((l, i) => {
      const z = SCR.z1 - 0.25 - (i - scroll) * 0.26, lp = at(l.at);
      if (!l.ok) { seg([x, ys, z + 0.06], [x, ys - 0.09, z], 1, TEXT, 1.8); seg([x, ys - 0.09, z], [x, ys, z - 0.06], 1, TEXT, 1.8); }
      const b0 = ys - (l.ok ? 0 : 0.2), b1 = b0 - l.len * clamp01(lp); if (b1 < b0) plateX(1, b1, b0, z - 0.045, z + 0.045, x, l.ok ? OKTXT : TEXT);
      last = [x, b1 - 0.08, z];
    });
    const blink = Math.floor(local / 9) % 2 === 0 || t < CH.hold - 5;
    if (blink) plateX(1, last[1] - 0.1, last[1], last[2] - 0.07, last[2] + 0.07, x, "#e8efe9");
    ctx.restore();
  }

  // ---------- 9. the report trays: draft behind, final in front; the report stack rides between them
  const tp = at(E.trays);
  const trayBack = (x0: number, x1: number) => { if (tp < 1) return; const { y0, y1, h, t: w } = TRAY; plateZ(1, x0 + w, x1 - w, y0 + w, y1 - w, 0.07, "#b9c6bd"); plateX(1, y0 + w, y1 - w, 0.07, h, x0 + w, M.tray.right); plateY(1, x0 + w, x1 - w, 0.07, h, y0 + w, M.tray.left); };
  const trayFront = (x0: number, x1: number) => { if (tp <= 0) return; const { y0, y1, h, t: w } = TRAY; iso.make(iso.boxFaces(x0, y0, 0, x1, y1, h, M.tray).slice(0, 2), tp, M.tray.right); if (tp >= 1) { plateZ(1, x0, x1, y0, y0 + w, h, M.tray.top); plateZ(1, x0, x0 + w, y0, y1, h, M.tray.top); plateZ(1, x1 - w, x1, y0, y1, h, M.tray.top); plateZ(1, x0, x1, y1 - w, y1, h, M.tray.top); } };
  // the report: three sheets, a title bar and text on the top one. The draft keeps its checksum
  // tag; finalize COPIES it (README: draft copied to report/final), and only the copy is sealed.
  const sh = at(E.sheets), sl = ease(at(E.slide));
  const stack = (dx: number, dz: number, copy: boolean) => {
    if (sh <= 0) return; const { x0, x1, y0, y1 } = STACK;
    for (let j = 0; j < 3; j++) { const p = clamp01(sh * 3.2 - j); box(p, x0 + dx, y0, 0.1 + j * 0.075 + dz, x1 + dx, y1, 0.16 + j * 0.075 + dz, M.paper); }
    const zt = 0.311 + dz, st = at(E.sheetText);
    if (st > 0) { plateZ(clamp01(st * 2), x0 + 0.18 + dx, x0 + 1.15 + dx, y0 + 0.15, y0 + 0.36, zt, INK); [0.8, 1.45, 1.05, 1.3].forEach((len, i) => seg([x0 + 0.18 + dx, y0 + 0.6 + i * 0.22, zt], [x0 + 0.18 + len + dx, y0 + 0.6 + i * 0.22, zt], clamp01(st * 5 - i - 1), "#b9b2a3", 1.5)); }
    const tx0 = x1 - 0.55 + dx, ty = y1; if (!copy) box(at(E.csTag), tx0, ty - 0.06, zt - 0.02, tx0 + TAGW, ty, zt + 0.4, M.tag);
    const hp = copy ? 0 : at(E.csHash); if (hp > 0) { const yy = ty + 0.001, z0 = zt + 0.1, z1 = zt + 0.3, xa = tx0 + 0.13, xb = tx0 + 0.27; seg([xa, yy, z0], [xa + 0.03, yy, z1], hp * 4, PAPER, 1.6); seg([xb, yy, z0], [xb + 0.03, yy, z1], hp * 4 - 1, PAPER, 1.6); seg([tx0 + 0.06, yy, z0 + 0.13], [tx0 + 0.37, yy, z0 + 0.13], hp * 4 - 2, PAPER, 1.6); seg([tx0 + 0.06, yy, z0 + 0.06], [tx0 + 0.37, yy, z0 + 0.06], hp * 4 - 3, PAPER, 1.6); }
    // the seal: a short green disc dropped onto the promoted draft, then its check
    const se = copy ? at(E.seal) : 0; if (se > 0) {
      const cx = x0 + 1.15 + dx, cy = (y0 + y1) / 2 + 0.1, r = 0.42, zb = zt + 0.002 + (1 - ease(se)) * 1.2, zt2 = zb + 0.07, N = 28;
      const ring = (z: number) => Array.from({ length: N }, (_, i) => { const a = (i / N) * Math.PI * 2; return P(cx + Math.cos(a) * r, cy + Math.sin(a) * r, z); });
      const lo = ring(zb), hi = ring(zt2); const side: P2[] = []; for (let i = 0; i <= N / 2; i++) side.push(lo[(i + N - 4) % N]); for (let i = N / 2; i >= 0; i--) side.push(hi[(i + N - 4) % N]);
      iso.poly(side, M.seal.left); iso.poly(hi, M.seal.top);
      const tk = at(E.sealTick); seg([cx - 0.2, cy + 0.05, zt2 + 0.001], [cx - 0.05, cy + 0.2, zt2 + 0.001], tk * 2, PAPER, 2.6); seg([cx - 0.05, cy + 0.2, zt2 + 0.001], [cx + 0.22, cy - 0.18, zt2 + 0.001], tk * 2 - 1, PAPER, 2.6);
    }
  };
  // The copy is drawn where its depth puts it: inside the draft tray while it rises, above both
  // trays in flight, inside the final tray once it is lower than the walls.
  const cdx = SLIDE * sl, cdz = Math.sin(Math.PI * sl) * 0.7, low = 0.1 + cdz < TRAY.h, nearDraft = cdx < SLIDE / 2;
  trayBack(...TRAY.draft); stack(0, 0, false); if (sl > 0 && low && nearDraft) stack(cdx, cdz, true); trayFront(...TRAY.draft);
  trayBack(...TRAY.final); if (sl > 0 && low && !nearDraft) stack(cdx, cdz, true); trayFront(...TRAY.final);
  if (sl > 0 && !low) stack(cdx, cdz, true);

  // ---------- 10. the link: a thread from the finding's pin, round the cabinet's corner, to the evidence tag
  const thp = at(E.thread); if (thp > 0) {
    const tag: P3 = [TAGX + 0.22, V.y1 + V.depth + 0.08, dzc(2)], corner: P3 = [V.x1 + 0.03, V.y1 + V.depth + 0.04, dzc(2) + 0.25];
    const a = P(...pin), c = P(...corner), b = P(...tag), mid: P2 = [(a[0] + c[0]) / 2, (a[1] + c[1]) / 2 + 22];
    const pts: P2[] = []; for (let i = 0; i <= 20; i++) { const s = i / 20; pts.push([(1 - s) ** 2 * a[0] + 2 * (1 - s) * s * mid[0] + s * s * c[0], (1 - s) ** 2 * a[1] + 2 * (1 - s) * s * mid[1] + s * s * c[1]]); } pts.push(b);
    const L = pts.slice(1).reduce((acc, q, i) => acc + Math.hypot(q[0] - pts[i][0], q[1] - pts[i][1]), 0); let left = L * thp;
    ctx.strokeStyle = INK; ctx.lineWidth = 2; ctx.lineJoin = "round"; ctx.beginPath(); ctx.moveTo(a[0], a[1]);
    for (let i = 1; i < pts.length && left > 0; i++) { const s = Math.hypot(pts[i][0] - pts[i - 1][0], pts[i][1] - pts[i - 1][1]), k = Math.min(1, left / s); ctx.lineTo(lerp(pts[i - 1][0], pts[i][0], k), lerp(pts[i - 1][1], pts[i][1], k)); left -= s; } ctx.stroke();
    [a, b].forEach((q) => ring(q, 6, at(E.rings), false));
  }
  const lb = at(E.linkBadge); if (lb > 0) { const x0 = CARD.x0 + 0.55; box(lb, x0, CARD.y, CARD.z1 - 0.3, x0 + 0.34, CARD.y + 0.07, CARD.z1 + 0.04, M.seal); if (lb >= 1) { const yy = CARD.y + 0.071, zc = CARD.z1 - 0.13; seg([x0 + 0.07, yy, zc], [x0 + 0.14, yy, zc - 0.09], 1, PAPER, 2); seg([x0 + 0.14, yy, zc - 0.09], [x0 + 0.28, yy, zc + 0.1], 1, PAPER, 2); } }

  // ---------- 11. verify: a scanner bar sweeps the drawer fronts, then the finding; its light is flat bands
  const bin = at(E.barIn), bout = at(E.barOut), sw = at(E.sweep);
  if (bin > 0 && bout < 1) {
    const zs = lerp(V.z1 + 0.15, 0.12, sw), yb = V.y1 + V.depth + 0.12;
    if (sw > 0 && sw < 1) { ctx.save(); const pts = [P(V.x0, yb - 0.02, 0), P(V.x1, yb - 0.02, 0), P(V.x1, yb - 0.02, V.z1), P(V.x0, yb - 0.02, V.z1)]; ctx.beginPath(); pts.forEach(([x, y], i) => (i ? ctx.lineTo(x, y) : ctx.moveTo(x, y))); ctx.closePath(); ctx.clip();
      plateY(1, V.x0, V.x1, zs - 0.42, zs + 0.42, yb - 0.03, LIGHT); plateY(1, V.x0, V.x1, zs - 0.16, zs + 0.16, yb - 0.025, LIGHT_CORE); ctx.restore();
      ctx.save(); ctx.globalAlpha = 0.25; plateY(1, V.x0, V.x1, zs - 0.42, zs + 0.42, yb - 0.03, ACCENT); ctx.restore(); }
    box(bin * (1 - bout), V.x0 - 0.15, yb, zs - 0.05, V.x1 + 0.15, yb + 0.12, zs + 0.05, M.tag);
  }
  const bsw = at(E.boardSweep); if (bsw > 0 && bsw < 1) { const zs = lerp(CARD.z1 + 0.05, CARD.z0 - 0.05, bsw); ctx.save(); ctx.globalAlpha = 0.45; plateY(1, CARD.x0, CARD.x1, Math.max(CARD.z0, zs - 0.3), Math.min(CARD.z1, zs + 0.3), CARD.y + 0.003, LIGHT); ctx.restore(); box(1, CARD.x0 - 0.1, CARD.y + 0.02, zs - 0.04, CARD.x1 + 0.1, CARD.y + 0.12, zs + 0.04, M.tag); }

  // ---------- 12. captures still on their way (above every drawer): drawn last so nothing hides them
  CAPTURES.forEach((c) => { const s = cardAt(c); if (s && at(c.drop) <= 0) drawCard(c, s.p, s.made); });

  // ---------- 13. lettering: object labels, the caption block, the chapter rail
  const g = new Gfx(ctx, env, local, CRISP);
  const write = (s: string, x: number, y: number, cap: number, color: string, p: number, align: "left" | "center" | "right" = "left") => { if (p > 0) letter(g, s, x, y, { cap, color, slant: 0, progress: p, opacity: 1, track: 1.35, seed: 3, align, w: Math.max(1.6, cap * 0.11) }); };
  const lp = at(E.labels);
  if (lp > 0) {
    const lab = (s: string, q: P2, dx: number, dy: number, align: "left" | "center" | "right" = "left") => write(s, q[0] + dx, q[1] + dy, 15, MUTED, lp, align);
    lab("SCOPE.YAML", P((SCOPE.x0 + SCOPE.x1) / 2, SCOPE.y0, 0.04), -14, -40, "center");
    lab("EVIDENCE/", P((V.x0 + V.x1) / 2, V.y0, V.z1), 0, -34, "center");
    lab("FINDINGS/", P((BOARD.x0 + BOARD.x1) / 2, BOARD.y0, BOARD.z1), 0, -30, "center");
    lab("REPORT/DRAFT", P(TRAY.draft[0], TRAY.y1, 0), -14, -2, "right");
    lab("REPORT/FINAL", P(TRAY.final[1], TRAY.y1, 0), 14, -10);
  }
  CAPS.forEach((c) => {
    write(c.step.s, 60, 44, 16, ACCENT, prog(t, c.step.w, c.step.e));
    c.cmd.forEach((m) => { const p = prog(t, m.w, m.e); if (p <= 0) return; const x = 60, y = 76; g.pen([[x, y + 4], [x + 11, y + 15], [x, y + 26]], { w: 3, color: MUTED, wobble: 0, boil: 0, taper: 0, opacity: 1, progress: clamp01(p * 6) }); write(m.s, x + 26, y, 30, INK, p); });
    write(c.sub.s, 60, 124, 17, MUTED, prog(t, c.sub.w, c.sub.e));
  });
  // the rail: five chapters, top right. A segment is drawn in accent as its chapter opens and turns
  // ink when the next one opens: it changes only at chapter cuts, so it never pulls the eye mid-chapter.
  const starts = [CH.init, CH.capture, CH.finding, CH.verify, CH.report, Infinity];
  for (let i = 0; i < 5; i++) {
    const x0 = 936 + i * 60, y = 56, w = 48, f = span(t, starts[i], starts[i] + 10), done = t >= starts[i + 1];
    iso.line([x0, y], [x0 + w, y], SOFT, 4); if (f > 0) iso.line([x0, y], [x0 + w * f, y], done ? INK : ACCENT, 4);
    write(`0${i + 1}`, x0, y + 12, 12, done ? INK : f > 0 ? ACCENT : "#b5ae9f", 1);
  }
};

export const dorocapFlow: Film = {
  meta: { title: "How dorocap works", W: 1280, H: 720, fps: FPS, bpm: BPM, durationFrames: DURATION, kind: "explainer", raster: "cpu", holds: [[E.sealTick[1], CH.rewind]] }, // the reading hold: only the cursor blinks
  assets: { images: {} },
  shots: [{ id: "engagement", start: 0, end: DURATION, draw }],
};
