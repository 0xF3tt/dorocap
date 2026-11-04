import type { Ctx, Env } from "./core";
import type { Film } from "./film";
import { Iso, type Mat, clamp01, ease, span } from "./isometricKit";

// Explainer, silent. One desk: a terminal, a vault, a report.
// Palette matches docs/dorocap-data-flow.html: paper #f3f1ec, ink #2a312e,
// verify orange #d86a2a, sealed report #2f6b45.
// Isometric 2:1. One light, upper left, carried only by the three face tones.
// Reference, from knowledge: a desktop terminal is a box with a darker screen
// on the face toward the viewer; a file is a thin slab; a tray is a low box.

const FPS = 30;
const BPM = 120;
const DURATION = 240;

const PAPER = "#f3f1ec";
const ORANGE = "#d86a2a";

const desk: Mat = { top: "#e7e2d6", left: "#cfc8b8", right: "#b3ab9a" };
const terminal: Mat = { top: "#3e4944", left: "#2a312e", right: "#1c221f" };
const screen: Mat = { top: "#24302b", left: "#1a211e", right: "#121715" };
const vault: Mat = { top: "#d4cfc3", left: "#b7b1a4", right: "#978f82" };
const card: Mat = { top: "#fffdf8", left: "#e6dfd2", right: "#cfc6b4" };
const seal: Mat = { top: "#7eae92", left: "#2f6b45", right: "#1d4a2e" };
const lamp: Mat = { top: "#f2b183", left: ORANGE, right: "#a34c16" };

const draw = (ctx: Ctx, frame: number, env: Env) => {
  const { W, H } = env;
  ctx.setTransform(env.scale, 0, 0, env.scale, 0, 0);
  ctx.fillStyle = PAPER;
  ctx.fillRect(0, 0, W, H);

  const iso = new Iso(ctx, 48, 640, 270);
  const box = (p: number, x0: number, y0: number, z0: number, x1: number, y1: number, z1: number, m: Mat) => {
    iso.make(iso.boxFaces(x0, y0, z0, x1, y1, z1, m), p, m.right);
  };

  const floorP = ease(span(frame, 0, 36));
  const termP = ease(span(frame, 18, 70));
  const vaultP = ease(span(frame, 48, 100));
  const drop = ease(span(frame, 100, 150));
  const verify = ease(span(frame, 150, 190));
  const report = ease(span(frame, 190, 230));

  // The desk is a low plinth. Its shadow is a flat plane, not a blur.
  if (floorP > 0) {
    iso.face(
      [[0.2, 0.4, 0], [14.4, 0.4, 0], [14.4, 10.2, 0], [0.2, 10.2, 0]],
      "#c9c3b4",
      0.45 * Math.min(1, floorP * 2),
    );
  }
  box(floorP, 0, 0, 0, 14, 9.6, 0.35, desk);

  // Terminal, back-left. The screen is a darker slab on the face toward us.
  box(termP, 0.8, 0.8, 0.35, 4.6, 3.4, 2.5, terminal);
  if (termP > 0.4) {
    const s = clamp01((termP - 0.4) / 0.6);
    box(s, 1.15, 3.15, 0.9, 4.25, 3.45, 2.15, screen);
    const bars = Math.ceil(3 * s);
    for (let i = 0; i < bars; i++) {
      const z0 = 1.15 + i * 0.32;
      box(1, 1.45, 3.46, z0, 4.0, 3.55, z0 + 0.1, lamp);
    }
  }

  // Vault: a low crate. During verify its top tone shifts toward orange.
  const vaultMat: Mat = verify <= 0
    ? vault
    : {
        top: mix(vault.top, "#f0c2a0", verify),
        left: mix(vault.left, "#e09860", verify * 0.65),
        right: mix(vault.right, "#c4743a", verify * 0.65),
      };
  box(vaultP, 5.4, 3.2, 0.35, 10.6, 8.2, 1.7, vaultMat);

  // Three evidence cards drop onto the vault, then sit still.
  if (drop > 0) {
    for (let i = 0; i < 3; i++) {
      const x0 = 1.5 + i * 0.7 + (5.7 + i * 1.5 - (1.5 + i * 0.7)) * drop;
      const y0 = 3.7 + (3.9 - 3.7) * drop;
      const z = 3.4 + (1.72 - 3.4) * drop;
      box(1, x0, y0, z, x0 + 1.2, y0 + 2.5, z + 0.14, card);
    }
  }

  // Hash ticks appear on the vault's near edge while verify runs.
  const ticks = Math.floor(verify * 4);
  for (let i = 0; i < ticks; i++) {
    const y0 = 4.0 + i * 0.95;
    box(1, 9.7, y0, 1.7, 10.15, y0 + 0.38, 1.86, lamp);
  }

  // The report grows out of the desk only after verify has finished.
  if (report > 0) {
    const h = 0.2 + 1.15 * report;
    box(1, 11.3, 4.4, 0.35, 13.6, 7.6, 0.35 + h, seal);
    if (report > 0.72) {
      const q = clamp01((report - 0.72) / 0.28);
      const zc = 0.4 + h;
      iso.line(iso.p(12.05, 6.35, zc), iso.p(12.4, 6.7, zc), PAPER, 5 * q);
      iso.line(iso.p(12.4, 6.7, zc), iso.p(13.05, 5.7, zc), PAPER, 5 * q);
    }
  }
};

const mix = (a: string, b: string, t: number) => {
  const p = (h: string) => [1, 3, 5].map((i) => parseInt(h.slice(i, i + 2), 16));
  const A = p(a), B = p(b);
  return "#" + A.map((v, i) => Math.round(v + (B[i] - v) * clamp01(t)).toString(16).padStart(2, "0")).join("");
};

export const dorocapFlow: Film = {
  meta: {
    title: "How dorocap works",
    W: 1280,
    H: 720,
    fps: FPS,
    bpm: BPM,
    durationFrames: DURATION,
    kind: "explainer",
    raster: "cpu",
  },
  assets: { images: {} },
  shots: [{ id: "desk", start: 0, end: DURATION, draw }],
};
