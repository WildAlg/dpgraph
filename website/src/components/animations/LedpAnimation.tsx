import { useEffect, useRef, useState } from "react";
import { motion, useInView } from "motion/react";
import { useReducedMotion } from "./useReducedMotion";

/**
 * LedpAnimation — local-edge DP threat-model loop.
 *
 * Mirrors Figure 1 of Mundra et al. 2025:
 *   each user holds a private adjacency list →
 *   applies an ε-local randomizer →
 *   sends a noisy output to an untrusted curator →
 *   curator aggregates and publishes LEDP-statistics.
 *
 * Six nodes on a hex; the "true" edges are drawn solid.
 * Each animation tick (~7s):
 *   t0   nodes pulse, edges trace in
 *   t1   adjacency bits at each node flicker — some flip (orange = noise)
 *   t2   small packets travel along straight lines to the curator
 *   t3   curator pulses; "LEDP-statistics" output emerges below
 */

const W = 460;
const H = 360;
const CX = W / 2;
const CY = 160;
const RADIUS = 116;

// six users on a hexagon
const NODES = Array.from({ length: 6 }, (_, i) => {
  const a = (Math.PI / 3) * i - Math.PI / 2;
  return { id: i, x: CX + RADIUS * Math.cos(a), y: CY + RADIUS * Math.sin(a) };
});

// edges (true graph) — small social-style topology
const EDGES: Array<[number, number]> = [
  [0, 1],
  [1, 2],
  [2, 3],
  [3, 4],
  [4, 5],
  [5, 0],
  [0, 2],
  [3, 5],
];

const CURATOR = { x: CX, y: 300, w: 88, h: 36 };

// per-node adjacency bits (4 bits each, just for the visual)
const BITS: number[][] = [
  [1, 0, 1, 1],
  [1, 1, 0, 1],
  [0, 1, 1, 0],
  [1, 0, 1, 1],
  [0, 1, 1, 0],
  [1, 1, 0, 1],
];
// which bits "flip" during the noise step
const FLIPS: boolean[][] = [
  [false, true, false, false],
  [true, false, false, false],
  [false, false, true, false],
  [false, true, false, false],
  [false, false, false, true],
  [true, false, false, false],
];

const TICK = 7000;

export default function LedpAnimation() {
  const ref = useRef<HTMLDivElement>(null);
  const inView = useInView(ref, { amount: 0.3 });
  const reduced = useReducedMotion();
  const [phase, setPhase] = useState(0); // 0..3

  useEffect(() => {
    if (!inView || reduced) return;
    const id = setInterval(
      () => setPhase((p) => (p + 1) % 4),
      TICK / 4
    );
    return () => clearInterval(id);
  }, [inView, reduced]);

  // reduced-motion: hold final aggregated state
  const p = reduced ? 3 : phase;

  return (
    <div
      ref={ref}
      className="relative w-full"
      role="img"
      aria-label="Six users each privately perturb their adjacency list and send a noisy output to an untrusted curator, who aggregates and publishes LEDP statistics."
    >
      <svg
        viewBox={`0 0 ${W} ${H}`}
        className="block w-full h-auto"
        preserveAspectRatio="xMidYMid meet"
      >
        {/* corner ticks — research-paper figure feel */}
        <Ticks />

        {/* edges (true graph) */}
        <g>
          {EDGES.map(([a, b], i) => {
            const A = NODES[a]!;
            const B = NODES[b]!;
            return (
              <motion.line
                key={i}
                x1={A.x}
                y1={A.y}
                x2={B.x}
                y2={B.y}
                stroke="currentColor"
                strokeOpacity={0.2}
                strokeWidth={1}
                initial={{ pathLength: 0 }}
                animate={{ pathLength: p >= 0 ? 1 : 0 }}
                transition={{ duration: 0.6, delay: 0.05 * i, ease: [0.22, 1, 0.36, 1] }}
              />
            );
          })}
        </g>

        {/* packets — travel from each node to the curator during phase 2 */}
        <g>
          {NODES.map((n, i) => (
            <motion.circle
              key={`pkt-${i}`}
              r={3}
              fill="var(--noise)"
              cx={n.x}
              cy={n.y}
              animate={
                p === 2
                  ? { cx: [n.x, CURATOR.x], cy: [n.y, CURATOR.y - 4], opacity: [0, 1, 1, 0] }
                  : { cx: n.x, cy: n.y, opacity: 0 }
              }
              transition={{ duration: 1.4, delay: 0.06 * i, ease: [0.22, 1, 0.36, 1] }}
            />
          ))}
        </g>

        {/* nodes + adjacency bits */}
        <g>
          {NODES.map((n, i) => (
            <Node
              key={i}
              x={n.x}
              y={n.y}
              bits={BITS[i]!}
              flips={FLIPS[i]!}
              phase={p}
              index={i}
            />
          ))}
        </g>

        {/* curator box */}
        <g>
          <motion.rect
            x={CURATOR.x - CURATOR.w / 2}
            y={CURATOR.y - CURATOR.h / 2}
            width={CURATOR.w}
            height={CURATOR.h}
            rx={3}
            fill="none"
            stroke="currentColor"
            strokeOpacity={0.5}
            strokeWidth={1}
            strokeDasharray="3 3"
            animate={
              p === 2 || p === 3
                ? { strokeOpacity: [0.5, 1, 0.5], stroke: ["var(--accent)", "var(--accent)", "var(--accent)"] }
                : { strokeOpacity: 0.4 }
            }
            transition={{ duration: 1, ease: [0.22, 1, 0.36, 1] }}
          />
          <text
            x={CURATOR.x}
            y={CURATOR.y - 2}
            textAnchor="middle"
            className="font-mono"
            fontSize={9}
            fill="currentColor"
            opacity={0.85}
          >
            CURATOR
          </text>
          <text
            x={CURATOR.x}
            y={CURATOR.y + 10}
            textAnchor="middle"
            className="font-mono"
            fontSize={7.5}
            fill="currentColor"
            opacity={0.45}
          >
            untrusted
          </text>
        </g>

        {/* publish pulse — appears in phase 3 */}
        <motion.circle
          cx={CURATOR.x}
          cy={CURATOR.y}
          r={CURATOR.w / 2 + 4}
          fill="none"
          stroke="var(--accent)"
          strokeWidth={1}
          initial={{ opacity: 0, scale: 0.8 }}
          animate={p === 3 ? { opacity: [0, 0.7, 0], scale: [0.85, 1.6, 1.9] } : { opacity: 0 }}
          transition={{ duration: 1.6, ease: [0.22, 1, 0.36, 1] }}
          style={{ transformOrigin: `${CURATOR.x}px ${CURATOR.y}px`, transformBox: "fill-box" }}
        />

        {/* LEDP-statistics output — small bar chart that "publishes" */}
        <g transform={`translate(${CURATOR.x - 30}, ${CURATOR.y + 28})`}>
          <motion.g
            initial={{ opacity: 0, y: 4 }}
            animate={p === 3 ? { opacity: 1, y: 0 } : { opacity: 0.18, y: 0 }}
            transition={{ duration: 0.7, ease: [0.22, 1, 0.36, 1] }}
          >
            <rect x={0} y={6} width={6} height={8} fill="currentColor" opacity={0.8} />
            <rect x={9} y={2} width={6} height={12} fill="currentColor" opacity={0.8} />
            <rect x={18} y={9} width={6} height={5} fill="currentColor" opacity={0.8} />
            <rect x={27} y={4} width={6} height={10} fill="currentColor" opacity={0.8} />
            <rect x={36} y={7} width={6} height={7} fill="currentColor" opacity={0.8} />
            <rect x={45} y={3} width={6} height={11} fill="currentColor" opacity={0.8} />
            <line x1={0} y1={16} x2={51} y2={16} stroke="currentColor" strokeWidth={0.5} opacity={0.4} />
          </motion.g>
          <text
            x={26}
            y={28}
            textAnchor="middle"
            className="font-mono"
            fontSize={7.5}
            fill="currentColor"
            opacity={0.55}
          >
            LEDP-statistics
          </text>
        </g>

        {/* phase legend */}
        <g transform="translate(16, 332)">
          <Legend phase={p} />
        </g>
      </svg>
    </div>
  );
}

function Node({
  x,
  y,
  bits,
  flips,
  phase,
  index,
}: {
  x: number;
  y: number;
  bits: number[];
  flips: boolean[];
  phase: number;
  index: number;
}) {
  // adjacency card sits to the side of each node (rotated by hex angle)
  const a = (Math.PI / 3) * index - Math.PI / 2;
  const cardX = x + Math.cos(a) * 28;
  const cardY = y + Math.sin(a) * 28;

  return (
    <g>
      <motion.circle
        cx={x}
        cy={y}
        r={6}
        fill="var(--bg)"
        stroke="currentColor"
        strokeWidth={1.4}
        animate={
          phase === 0
            ? { scale: [1, 1.18, 1] }
            : phase === 1
            ? { stroke: ["currentColor", "var(--noise)", "currentColor"] }
            : { scale: 1 }
        }
        transition={{ duration: 0.9, delay: index * 0.06, ease: [0.22, 1, 0.36, 1] }}
        style={{ transformOrigin: `${x}px ${y}px`, transformBox: "fill-box" }}
      />
      {/* adjacency bit card */}
      <g transform={`translate(${cardX - 13}, ${cardY - 6})`} opacity={0.95}>
        <rect width={26} height={12} rx={1.5} fill="var(--bg)" stroke="currentColor" strokeOpacity={0.35} strokeWidth={0.6} />
        {bits.map((b, j) => {
          const flipped = phase >= 1 && flips[j];
          const shown = flipped ? 1 - b : b;
          return (
            <motion.text
              key={j}
              x={3 + j * 6.2}
              y={9}
              fontSize={7}
              className="font-mono"
              fill={flipped && phase === 1 ? "var(--noise)" : "currentColor"}
              opacity={flipped && phase === 1 ? 1 : 0.85}
              animate={
                phase === 1 && flips[j]
                  ? { opacity: [0.4, 1, 0.85] }
                  : { opacity: 0.85 }
              }
              transition={{ duration: 0.7, delay: index * 0.05 + j * 0.04 }}
            >
              {shown}
            </motion.text>
          );
        })}
      </g>
    </g>
  );
}

function Legend({ phase }: { phase: number }) {
  const labels = ["1 · True graph", "2 · Local randomizer", "3 · Send to curator", "4 · Aggregate & publish"];
  return (
    <g className="font-mono" fontSize={8.5} fill="currentColor" opacity={0.72}>
      {labels.map((l, i) => (
        <g key={i} transform={`translate(${i * 110}, 0)`}>
          <rect x={0} y={-7} width={4} height={4} fill={i === phase ? "var(--accent)" : "currentColor"} opacity={i === phase ? 1 : 0.25} />
          <text x={9} y={-3} fill="currentColor" opacity={i === phase ? 1 : 0.5}>{l}</text>
        </g>
      ))}
    </g>
  );
}

function Ticks() {
  const t = (x: number, y: number) => (
    <g key={`${x}-${y}`} stroke="currentColor" strokeOpacity={0.3} strokeWidth={0.7}>
      <line x1={x - 5} y1={y} x2={x + 5} y2={y} />
      <line x1={x} y1={y - 5} x2={x} y2={y + 5} />
    </g>
  );
  return (
    <g>
      {t(8, 8)}
      {t(W - 8, 8)}
      {t(8, H - 8)}
      {t(W - 8, H - 8)}
    </g>
  );
}
