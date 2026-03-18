'use client';

// Hash a string to a number
function hashStr(s: string): number {
  let hash = 0;
  for (let i = 0; i < s.length; i++) {
    hash = ((hash << 5) - hash + s.charCodeAt(i)) | 0;
  }
  return Math.abs(hash);
}

// Palette of muted colors that work on dark backgrounds
const COLORS = ['#8b5cf6', '#06b6d4', '#f59e0b', '#10b981', '#ef4444', '#ec4899', '#3b82f6', '#f97316'];

// Shapes: circle, diamond, hexagon, triangle
const SHAPES = ['circle', 'diamond', 'hexagon', 'triangle'] as const;

interface Props {
  name: string;
  size?: number; // default 20
}

export default function AgentAvatar({ name, size = 20 }: Props) {
  const hash = hashStr(name);
  const color = COLORS[hash % COLORS.length];
  const shape = SHAPES[(hash >> 4) % SHAPES.length];
  const half = size / 2;

  return (
    <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} aria-hidden="true" className="shrink-0">
      {shape === 'circle' && <circle cx={half} cy={half} r={half * 0.7} fill={color} opacity={0.8} />}
      {shape === 'diamond' && <path d={`M${half} ${size*0.1} L${size*0.9} ${half} L${half} ${size*0.9} L${size*0.1} ${half} Z`} fill={color} opacity={0.8} />}
      {shape === 'hexagon' && <path d={`M${half} ${size*0.1} L${size*0.85} ${size*0.3} L${size*0.85} ${size*0.7} L${half} ${size*0.9} L${size*0.15} ${size*0.7} L${size*0.15} ${size*0.3} Z`} fill={color} opacity={0.8} />}
      {shape === 'triangle' && <path d={`M${half} ${size*0.1} L${size*0.9} ${size*0.85} L${size*0.1} ${size*0.85} Z`} fill={color} opacity={0.8} />}
    </svg>
  );
}
