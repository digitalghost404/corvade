'use client';

import { useEffect, useState } from 'react';
import { useScramble } from '@/hooks/useScramble';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:4401';

interface StatsResponse {
  trace_count: number;
  total_cost: number;
  total_tokens: number;
  by_model: Record<string, unknown>;
  by_agent: Record<string, unknown>;
}

interface Metric {
  label: string;
  value: string;
}

function MetricCard({ label, value }: Metric) {
  const display = useScramble(value);
  return (
    <div className="glass neon-edge rounded-lg px-4 py-3 flex-1 min-w-0">
      <div className="tabular-nums text-violet-400 text-lg font-semibold">{display}</div>
      <div className="text-zinc-400 text-xs uppercase tracking-wider mt-1">{label}</div>
    </div>
  );
}

export default function StatsBar() {
  const [stats, setStats] = useState<StatsResponse | null>(null);

  useEffect(() => {
    fetch(`${API_BASE}/api/stats`)
      .then((r) => r.json())
      .then((data: StatsResponse) => setStats(data))
      .catch(() => {
        setStats({
          trace_count: 0,
          total_cost: 0,
          total_tokens: 0,
          by_model: {},
          by_agent: {},
        });
      });
  }, []);

  const metrics: Metric[] = stats
    ? [
        { label: 'Traces', value: stats.trace_count.toLocaleString() },
        { label: 'Tokens', value: stats.total_tokens.toLocaleString() },
        { label: 'Cost', value: `$${stats.total_cost.toFixed(4)}` },
        { label: 'Agents', value: Object.keys(stats.by_agent ?? {}).length.toLocaleString() },
        { label: 'Models', value: Object.keys(stats.by_model ?? {}).length.toLocaleString() },
      ]
    : [
        { label: 'Traces', value: '—' },
        { label: 'Tokens', value: '—' },
        { label: 'Cost', value: '—' },
        { label: 'Agents', value: '—' },
        { label: 'Models', value: '—' },
      ];

  return (
    <div className="flex gap-3">
      {metrics.map((m) => (
        <MetricCard key={m.label} label={m.label} value={m.value} />
      ))}
    </div>
  );
}
