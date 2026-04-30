import { useEffect, useRef, useState } from "react";
import { motion, useInView } from "motion/react";
import { useReducedMotion } from "./useReducedMotion";

/**
 * KCoreAnimation — Levels Data Structure (LDS) walkthrough.
 *
 * Mirrors Figure 3 of Mundra et al. 2025: nodes start at level 0 and rise
 * level-by-level when their noisy neighbour count clears the per-level
 * threshold. Failed checks flash orange (the noise term blocked them).
 *
 * Two synchronized panels:
 *   left  — graph with nodes coloured by current LDS level
 *   right — horizontal level bands with node tokens that shift up
 *
 * Auto-advances when on screen, pauses otherwise. A play/scrub control
 * also lets users step manually.
 */

type NodeId = "a" | "b" | "c" | "d" | "e" | "f" | "g";

const NODES: { id: NodeId; x: number; y: number }[] = [
  { id: "a", x: 60, y: 60 },
  { id: "b", x: 140, y: 40 },
  { id: "c", x: 200, y: 110 },
  { id: "d", x: 145, y: 170 },
  { id: "e", x: 60, y: 160 },
  { id: "f", x: 105, y: 110 },
  { id: "g", x: 220, y: 50 },
];

const EDGES: [NodeId, NodeId][] = [
  ["a", "b"],
  ["a", "f"],
  ["a", "e"],
  ["b", "f"],
  ["b", "c"],
  ["c", "f"],
  ["c", "d"],
  ["d", "f"],
  ["d", "e"],
  ["e", "f"],
  ["b", "g"],
];

// rounds: per-node level over time. failures (level didn't change despite a check) flagged separately.
type Round = {
  levels: Record<NodeId, number>;
  failed: Partial<Record<NodeId, true>>;
  caption: string;
};

const ROUNDS: Round[] = [
  {
    levels: { a: 0, b: 0, c: 0, d: 0, e: 0, f: 0, g: 0 },
    failed: {},
    caption: "Round 0 · all nodes initialized at level 0",
  },
  {
    levels: { a: 1, b: 1, c: 1, d: 1, e: 1, f: 1, g: 0 },
    failed: { g: true },
    caption: "Round 1 · degree threshold clears for everyone except g",
  },
  {
    levels: { a: 2, b: 2, c: 2, d: 2, e: 2, f: 2, g: 0 },
    failed: { g: true },
    caption: "Round 2 · the dense cluster moves up; g still fails the noisy check",
  },
  {
    levels: { a: 3, b: 3, c: 3, d: 3, e: 2, f: 3, g: 0 },
    failed: { e: true, g: true },
    caption: "Round 3 · 4-clique {a,b,c,f} settles at level 3; e and g halt",
  },
  {
    levels: { a: 4, b: 4, c: 4, d: 3, e: 2, f: 4, g: 0 },
    failed: { d: true, e: true, g: true },
    caption: "Round 4 · final levels — core estimates derived from levels via post-processing",
  },
];

const LEVEL_COLOR = (lvl: number) => {
  // soft cool-to-warm scale on accent
  const stops = ["#dcd9d0", "#a8b3d3", "#7d8fbe", "#4f63a5", "#2c4994"];
  return stops[Math.min(lvl, stops.length - 1)];
};

const TICK_MS = 1900;

export default function KCoreAnimation() {
  const ref = useRef<HTMLDivElement>(null);
  const inView = useInView(ref, { amount: 0.3 });
  const reduced = useReducedMotion();
  const [round, setRound] = useState(0);
  const [paused, setPaused] = useState(false);

  useEffect(() => {
    if (!inView || reduced || paused) return;
    const id = setInterval(() => setRound((r) => (r + 1) % ROUNDS.length), TICK_MS);
    return () => clearInterval(id);
  }, [inView, reduced, paused]);

  const r = ROUNDS[round]!;
  const maxLevel = 4;

  return (
    <div
      ref={ref}
      className="rounded-md border p-4 sm:p-6"
      style={{ borderColor: "var(--rule)" }}
      role="img"
      aria-label="Animation of the Levels Data Structure: nodes incrementally move to higher levels when their noisy neighbour count clears a threshold."
    >
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <div className="text-[11px] uppercase tracking-[0.14em]" style={{ color: "var(--muted)" }}>
          Figure 3 · LDS level moves
        </div>
        <div className="flex items-center gap-1">
          <Step label="Prev" onClick={() => setRound((r) => (r - 1 + ROUNDS.length) % ROUNDS.length)} />
          <Step label={paused ? "Play" : "Pause"} onClick={() => setPaused((p) => !p)} primary />
          <Step label="Next" onClick={() => setRound((r) => (r + 1) % ROUNDS.length)} />
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        {/* graph panel */}
        <svg viewBox="0 0 280 220" className="w-full h-auto">
          <g stroke="currentColor" strokeOpacity={0.22} strokeWidth={1}>
            {EDGES.map(([a, b], i) => {
              const A = NODES.find((n) => n.id === a)!;
              const B = NODES.find((n) => n.id === b)!;
              return <line key={i} x1={A.x} y1={A.y} x2={B.x} y2={B.y} />;
            })}
          </g>
          {NODES.map((n) => {
            const lvl = r.levels[n.id];
            const failed = r.failed[n.id];
            return (
              <g key={n.id}>
                <motion.circle
                  cx={n.x}
                  cy={n.y}
                  r={11}
                  fill={LEVEL_COLOR(lvl)}
                  stroke={failed ? "var(--noise)" : "currentColor"}
                  strokeWidth={failed ? 1.6 : 1}
                  animate={
                    failed
                      ? { strokeOpacity: [0.2, 1, 0.7], scale: [1, 1.06, 1] }
                      : { strokeOpacity: 0.85, scale: 1 }
                  }
                  transition={{ duration: 0.7, ease: [0.22, 1, 0.36, 1] }}
                  style={{ transformOrigin: `${n.x}px ${n.y}px`, transformBox: "fill-box" }}
                />
                <text x={n.x} y={n.y + 3} textAnchor="middle" fontSize={9} className="font-mono" fill="#0E0E10" style={{ pointerEvents: "none" }}>
                  {n.id}
                </text>
              </g>
            );
          })}
          <text x={10} y={210} fontSize={9} className="font-mono" fill="currentColor" opacity={0.55}>
            graph view
          </text>
        </svg>

        {/* LDS bands */}
        <svg viewBox="0 0 280 220" className="w-full h-auto">
          <g>
            {Array.from({ length: maxLevel + 1 }, (_, lvl) => {
              const y = 200 - lvl * 40;
              return (
                <g key={lvl}>
                  <line x1={20} y1={y} x2={260} y2={y} stroke="currentColor" strokeOpacity={0.18} strokeDasharray="2 3" />
                  <text x={6} y={y - 2} fontSize={9} className="font-mono" fill="currentColor" opacity={0.55}>
                    L{lvl}
                  </text>
                </g>
              );
            })}
          </g>
          {NODES.map((n, i) => {
            const lvl = r.levels[n.id];
            const targetY = 200 - lvl * 40 - 12;
            const x = 50 + i * 30;
            const failed = r.failed[n.id];
            return (
              <motion.g
                key={n.id}
                initial={false}
                animate={{ x, y: targetY }}
                transition={{ duration: 0.6, ease: [0.22, 1, 0.36, 1] }}
              >
                <circle r={10} fill={LEVEL_COLOR(lvl)} stroke={failed ? "var(--noise)" : "currentColor"} strokeWidth={failed ? 1.6 : 1} strokeOpacity={0.85} />
                <text textAnchor="middle" y={3} fontSize={9} className="font-mono" fill="#0E0E10">
                  {n.id}
                </text>
              </motion.g>
            );
          })}
          <text x={10} y={213} fontSize={9} className="font-mono" fill="currentColor" opacity={0.55}>
            level data structure
          </text>
        </svg>
      </div>

      <div className="mt-3 flex items-center justify-between gap-3 text-[12.5px]" style={{ color: "var(--muted)" }}>
        <div>{r.caption}</div>
        <div className="font-mono text-[11px]">
          round {round + 1} / {ROUNDS.length}
        </div>
      </div>
    </div>
  );
}

function Step({ label, onClick, primary }: { label: string; onClick: () => void; primary?: boolean }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="rounded-full border px-2.5 py-1 font-mono text-[11px] transition-colors"
      style={{
        borderColor: "var(--rule)",
        background: primary ? "var(--accent)" : "transparent",
        color: primary ? "#fff" : "var(--fg)",
      }}
    >
      {label}
    </button>
  );
}
