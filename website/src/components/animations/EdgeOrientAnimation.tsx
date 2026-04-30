import { useEffect, useRef, useState } from "react";
import { motion, useInView } from "motion/react";
import { useReducedMotion } from "./useReducedMotion";

/**
 * EdgeOrientAnimation — three-frame walkthrough of Figure 4.
 *
 *   1) Original graph: triangles_blue = 2
 *   2) Randomized response: edges flip with prob 1/(e^eps + 1)
 *   3) Low out-degree orientation on the noisy graph; oriented triangle = 1
 */

type Pt = { id: string; x: number; y: number; label: string };

const NODES: Pt[] = [
  { id: "a", x: 70, y: 60, label: "a" },
  { id: "b", x: 200, y: 60, label: "b" },
  { id: "c", x: 135, y: 135, label: "c" },
  { id: "d", x: 60, y: 175, label: "d" },
  { id: "e", x: 215, y: 165, label: "e" },
];

type Edge = { a: string; b: string; kind?: "true" | "noisy" | "removed" | "added"; oriented?: "ab" | "ba" };

const FRAMES: { title: string; caption: string; edges: Edge[]; highlight?: string[] }[] = [
  {
    title: "Original graph",
    caption: "Two true triangles incident to node c — {a,b,c} and {b,c,e}.",
    edges: [
      { a: "a", b: "b", kind: "true" },
      { a: "a", b: "c", kind: "true" },
      { a: "b", b: "c", kind: "true" },
      { a: "b", b: "e", kind: "true" },
      { a: "c", b: "e", kind: "true" },
      { a: "a", b: "d", kind: "true" },
      { a: "c", b: "d", kind: "true" },
    ],
    highlight: ["a-b", "a-c", "b-c", "b-e", "c-e"],
  },
  {
    title: "Randomized response",
    caption: "Each edge bit flips with probability 1/(eᵉ+1). One true edge drops out, one spurious edge appears.",
    edges: [
      { a: "a", b: "b", kind: "true" },
      { a: "a", b: "c", kind: "true" },
      { a: "b", b: "c", kind: "removed" }, // dropped
      { a: "b", b: "e", kind: "true" },
      { a: "c", b: "e", kind: "true" },
      { a: "a", b: "d", kind: "true" },
      { a: "c", b: "d", kind: "true" },
      { a: "a", b: "e", kind: "added" }, // spurious
    ],
  },
  {
    title: "Low out-degree orientation",
    caption: "Edges oriented from low to high level by k-CoreD. Only oriented triangles count — estimate is 1.",
    edges: [
      { a: "a", b: "b", kind: "true", oriented: "ab" },
      { a: "a", b: "c", kind: "true", oriented: "ab" },
      { a: "b", b: "e", kind: "true", oriented: "ab" },
      { a: "c", b: "e", kind: "true", oriented: "ab" },
      { a: "a", b: "d", kind: "true", oriented: "ba" },
      { a: "c", b: "d", kind: "true", oriented: "ba" },
      { a: "a", b: "e", kind: "added", oriented: "ab" },
    ],
    highlight: ["a-c", "c-e", "a-e"], // the surviving oriented triangle
  },
];

const TICK_MS = 2700;

export default function EdgeOrientAnimation() {
  const ref = useRef<HTMLDivElement>(null);
  const inView = useInView(ref, { amount: 0.3 });
  const reduced = useReducedMotion();
  const [frame, setFrame] = useState(0);
  const [paused, setPaused] = useState(false);

  useEffect(() => {
    if (!inView || reduced || paused) return;
    const id = setInterval(() => setFrame((f) => (f + 1) % FRAMES.length), TICK_MS);
    return () => clearInterval(id);
  }, [inView, reduced, paused]);

  const F = FRAMES[frame]!;

  return (
    <div
      ref={ref}
      className="rounded-md border p-4 sm:p-6"
      style={{ borderColor: "var(--rule)" }}
      role="img"
      aria-label="EdgeOrient: three-step walkthrough of randomized response and low out-degree orientation for triangle counting."
    >
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <div className="text-[11px] uppercase tracking-[0.14em]" style={{ color: "var(--muted)" }}>
          Figure 4 · EdgeOrient pipeline
        </div>
        <div className="flex items-center gap-1">
          {FRAMES.map((_, i) => (
            <button
              key={i}
              type="button"
              onClick={() => {
                setFrame(i);
                setPaused(true);
              }}
              className="rounded-full border px-2.5 py-1 font-mono text-[11px] transition-colors"
              style={{
                borderColor: i === frame ? "var(--accent)" : "var(--rule)",
                background: i === frame ? "var(--accent)" : "transparent",
                color: i === frame ? "#fff" : "var(--fg)",
              }}
            >
              {i + 1}
            </button>
          ))}
          <button
            type="button"
            onClick={() => setPaused((p) => !p)}
            className="ml-1 rounded-full border px-2.5 py-1 font-mono text-[11px]"
            style={{ borderColor: "var(--rule)" }}
          >
            {paused ? "Play" : "Pause"}
          </button>
        </div>
      </div>

      <svg viewBox="0 0 280 220" className="w-full h-auto">
        {/* edges */}
        <g>
          {F.edges.map((e, i) => {
            const A = NODES.find((n) => n.id === e.a)!;
            const B = NODES.find((n) => n.id === e.b)!;
            const isTrue = e.kind === "true";
            const isRemoved = e.kind === "removed";
            const isAdded = e.kind === "added";
            const stroke = isAdded ? "var(--noise)" : isRemoved ? "var(--noise)" : "currentColor";
            const dash = isRemoved ? "3 4" : isAdded ? "4 3" : undefined;
            const opacity = isRemoved ? 0.35 : isAdded ? 0.95 : 0.6;
            return (
              <motion.line
                key={`${e.a}-${e.b}-${i}`}
                x1={A.x}
                y1={A.y}
                x2={B.x}
                y2={B.y}
                stroke={stroke}
                strokeOpacity={opacity}
                strokeWidth={isAdded || isRemoved ? 1.3 : 1}
                strokeDasharray={dash}
                initial={{ pathLength: 0 }}
                animate={{ pathLength: 1 }}
                transition={{ duration: 0.45, delay: i * 0.04, ease: [0.22, 1, 0.36, 1] }}
              />
            );
          })}
        </g>

        {/* arrowheads on oriented edges (frame 3) */}
        {frame === 2 && (
          <g>
            {F.edges
              .filter((e) => e.oriented)
              .map((e, i) => {
                const A = NODES.find((n) => n.id === e.a)!;
                const B = NODES.find((n) => n.id === e.b)!;
                const from = e.oriented === "ab" ? A : B;
                const to = e.oriented === "ab" ? B : A;
                const dx = to.x - from.x;
                const dy = to.y - from.y;
                const len = Math.sqrt(dx * dx + dy * dy);
                const ux = dx / len;
                const uy = dy / len;
                // place arrowhead near 60% of the way to keep it off the node
                const px = from.x + ux * (len - 18);
                const py = from.y + uy * (len - 18);
                const ang = (Math.atan2(dy, dx) * 180) / Math.PI;
                const isAdded = e.kind === "added";
                return (
                  <motion.g
                    key={`arr-${i}`}
                    transform={`translate(${px}, ${py}) rotate(${ang})`}
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    transition={{ duration: 0.4, delay: 0.4 + i * 0.04 }}
                  >
                    <path d="M 0 0 L -5 -3 L -5 3 Z" fill={isAdded ? "var(--noise)" : "currentColor"} opacity={isAdded ? 0.9 : 0.65} />
                  </motion.g>
                );
              })}
          </g>
        )}

        {/* highlight ring on the focal triangle */}
        {F.highlight && frame === 0 && <TriangleHighlight ids={["a", "b", "c"]} stroke="var(--accent)" />}
        {F.highlight && frame === 0 && <TriangleHighlight ids={["b", "c", "e"]} stroke="var(--accent)" />}
        {F.highlight && frame === 2 && <TriangleHighlight ids={["a", "c", "e"]} stroke="var(--accent)" filled />}

        {/* nodes */}
        {NODES.map((n) => (
          <g key={n.id}>
            <circle cx={n.x} cy={n.y} r={9} fill="var(--bg)" stroke="currentColor" strokeWidth={1} />
            <text x={n.x} y={n.y + 3} textAnchor="middle" fontSize={9} className="font-mono" fill="currentColor">
              {n.label}
            </text>
          </g>
        ))}
      </svg>

      <div className="mt-3 flex flex-col gap-1 text-[12.5px]" style={{ color: "var(--muted)" }}>
        <div className="flex items-center justify-between gap-3">
          <span style={{ color: "var(--fg)" }} className="font-medium">{frame + 1}. {F.title}</span>
          <span className="font-mono text-[11px]">step {frame + 1} / {FRAMES.length}</span>
        </div>
        <div>{F.caption}</div>
      </div>
    </div>
  );
}

function TriangleHighlight({ ids, stroke, filled }: { ids: string[]; stroke: string; filled?: boolean }) {
  const pts = ids.map((id) => NODES.find((n) => n.id === id)!);
  const d = `M ${pts[0]!.x} ${pts[0]!.y} L ${pts[1]!.x} ${pts[1]!.y} L ${pts[2]!.x} ${pts[2]!.y} Z`;
  return (
    <motion.path
      d={d}
      fill={filled ? "var(--accent)" : "none"}
      fillOpacity={filled ? 0.07 : 0}
      stroke={stroke}
      strokeWidth={1}
      strokeOpacity={0.6}
      strokeDasharray="3 3"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.5 }}
    />
  );
}
