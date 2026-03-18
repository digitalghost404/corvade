'use client';

import { useEffect, useRef } from 'react';

// ─── Types ────────────────────────────────────────────────────────────────────

interface Node {
  x: number;
  y: number;
  z: number;       // depth 0.5–1.0  (1.0 = closest)
  vx: number;
  vy: number;
  baseOpacity: number;
  radius: number;
  pulseT: number;  // remaining pulse frames (0 = inactive)
}

// ─── Constants ────────────────────────────────────────────────────────────────

const NODE_COUNT        = 50;
const EDGE_DISTANCE     = 150;
const TARGET_FPS        = 30;
const FRAME_INTERVAL_MS = 1000 / TARGET_FPS;
const VIOLET            = '139, 92, 246'; // #8b5cf6 as RGB components
const PULSE_DURATION_F  = 24;             // frames at 30fps ≈ 0.8s
const PULSE_INTERVAL_MS_MIN = 3000;
const PULSE_INTERVAL_MS_MAX = 5000;

// ─── Helpers ──────────────────────────────────────────────────────────────────

function rand(min: number, max: number): number {
  return Math.random() * (max - min) + min;
}

function makeNode(width: number, height: number): Node {
  const z           = rand(0.5, 1.0);
  const speed       = rand(0.1, 0.3) * z;   // farther → slower
  const angle       = rand(0, Math.PI * 2);

  return {
    x:           rand(0, width),
    y:           rand(0, height),
    z,
    vx:          Math.cos(angle) * speed,
    vy:          Math.sin(angle) * speed,
    baseOpacity: rand(0.1, 0.3) * z,          // farther → dimmer
    radius:      rand(1.5, 3) * z,             // farther → smaller
    pulseT:      0,
  };
}

function initNodes(width: number, height: number): Node[] {
  const nodes: Node[] = [];
  for (let i = 0; i < NODE_COUNT; i++) {
    nodes.push(makeNode(width, height));
  }
  return nodes;
}

// ─── Component ────────────────────────────────────────────────────────────────

export default function TopologyCanvas() {
  const canvasRef  = useRef<HTMLCanvasElement>(null);
  // Stable refs so closures inside rAF always see current values
  const nodesRef   = useRef<Node[]>([]);
  const rafRef     = useRef<number>(0);
  const lastTsRef  = useRef<number>(0);
  const pulseTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // ── Schedule next pulse ──────────────────────────────────────────────────
  function schedulePulse() {
    const delay = rand(PULSE_INTERVAL_MS_MIN, PULSE_INTERVAL_MS_MAX);
    pulseTimerRef.current = setTimeout(() => {
      const nodes = nodesRef.current;
      if (nodes.length > 0) {
        const idx = Math.floor(Math.random() * nodes.length);
        nodes[idx].pulseT = PULSE_DURATION_F;
      }
      schedulePulse();
    }, delay);
  }

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    // ── Size canvas to viewport ──────────────────────────────────────────
    function resize() {
      if (!canvas) return;
      canvas.width  = window.innerWidth;
      canvas.height = window.innerHeight;
    }

    resize();
    nodesRef.current = initNodes(canvas.width, canvas.height);
    schedulePulse();

    // ── Main draw ────────────────────────────────────────────────────────
    function draw(ts: number) {
      rafRef.current = requestAnimationFrame(draw);

      // Frame-rate cap
      const elapsed = ts - lastTsRef.current;
      if (elapsed < FRAME_INTERVAL_MS) return;
      lastTsRef.current = ts - (elapsed % FRAME_INTERVAL_MS);

      if (!canvas || !ctx) return;

      const W = canvas.width;
      const H = canvas.height;

      ctx.clearRect(0, 0, W, H);

      const nodes = nodesRef.current;

      // ── Update positions ─────────────────────────────────────────────
      for (const node of nodes) {
        node.x += node.vx;
        node.y += node.vy;

        // Wrap to opposite edge
        if (node.x < -node.radius)  node.x = W + node.radius;
        if (node.x > W + node.radius) node.x = -node.radius;
        if (node.y < -node.radius)  node.y = H + node.radius;
        if (node.y > H + node.radius) node.y = -node.radius;

        if (node.pulseT > 0) node.pulseT--;
      }

      // ── Draw edges ───────────────────────────────────────────────────
      for (let i = 0; i < nodes.length; i++) {
        const a = nodes[i];
        // Cull nodes entirely off-screen
        if (a.x < 0 || a.x > W || a.y < 0 || a.y > H) continue;

        for (let j = i + 1; j < nodes.length; j++) {
          const b = nodes[j];
          if (b.x < 0 || b.x > W || b.y < 0 || b.y > H) continue;

          const dx   = a.x - b.x;
          const dy   = a.y - b.y;
          const dist = Math.sqrt(dx * dx + dy * dy);
          if (dist > EDGE_DISTANCE) continue;

          // Fade edge with distance
          const distFade = 1 - dist / EDGE_DISTANCE;

          // Flash if either endpoint is pulsing
          const pulsing  = a.pulseT > 0 || b.pulseT > 0;
          const flashAmt = pulsing
            ? Math.max(a.pulseT, b.pulseT) / PULSE_DURATION_F
            : 0;
          const baseAlpha  = rand(0.03, 0.06) * distFade;
          const flashAlpha = 0.18 * flashAmt * distFade;

          ctx.beginPath();
          ctx.strokeStyle = `rgba(${VIOLET}, ${baseAlpha + flashAlpha})`;
          ctx.lineWidth   = 0.8;
          ctx.moveTo(a.x, a.y);
          ctx.lineTo(b.x, b.y);
          ctx.stroke();
        }
      }

      // ── Draw nodes ───────────────────────────────────────────────────
      for (const node of nodes) {
        if (node.x < 0 || node.x > W || node.y < 0 || node.y > H) continue;

        const pulseProgress = node.pulseT / PULSE_DURATION_F;
        const pulseOpacity  = 0.6 * pulseProgress;
        const opacity       = node.pulseT > 0
          ? Math.max(node.baseOpacity, pulseOpacity)
          : node.baseOpacity;

        ctx.beginPath();
        ctx.arc(node.x, node.y, node.radius, 0, Math.PI * 2);
        ctx.fillStyle = `rgba(${VIOLET}, ${opacity})`;
        ctx.fill();

        // Subtle halo during pulse
        if (node.pulseT > 0) {
          const haloRadius = node.radius + 4 * pulseProgress;
          const haloAlpha  = 0.15 * pulseProgress;
          ctx.beginPath();
          ctx.arc(node.x, node.y, haloRadius, 0, Math.PI * 2);
          ctx.fillStyle = `rgba(${VIOLET}, ${haloAlpha})`;
          ctx.fill();
        }
      }
    }

    rafRef.current = requestAnimationFrame(draw);

    // ── Resize handler ───────────────────────────────────────────────────
    const handleResize = () => {
      resize();
      // Re-seed nodes for new dimensions; existing nodes keep positions
      // and will wrap naturally — just add any missing ones if needed
    };

    window.addEventListener('resize', handleResize);

    // ── Cleanup ──────────────────────────────────────────────────────────
    return () => {
      cancelAnimationFrame(rafRef.current);
      window.removeEventListener('resize', handleResize);
      if (pulseTimerRef.current !== null) {
        clearTimeout(pulseTimerRef.current);
      }
    };
  }, []);

  return (
    <canvas
      ref={canvasRef}
      aria-hidden="true"
      style={{
        position:      'fixed',
        inset:         0,
        width:         '100%',
        height:        '100%',
        zIndex:        0,
        pointerEvents: 'none',
        display:       'block',
      }}
    />
  );
}
