'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { fetchStats, fetchTraces } from '@/lib/api';
import EmptyState from '@/components/EmptyState';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface StatsData {
  trace_count: number;
  total_cost: number;
  total_tokens: number;
  by_model: Record<string, number>;
  by_agent: Record<string, number>;
}

interface Trace {
  id: string;
  created_at: string;
  cost?: number | null;
}

// ---------------------------------------------------------------------------
// Time-range helpers
// ---------------------------------------------------------------------------

type Range = 'Last hour' | 'Last 24h' | 'Last 7d' | 'Last 30d' | 'All time';

const RANGES: Range[] = ['Last hour', 'Last 24h', 'Last 7d', 'Last 30d', 'All time'];

function fromForRange(range: Range): string | undefined {
  if (range === 'All time') return undefined;
  const now = new Date();
  if (range === 'Last hour') {
    return new Date(now.getTime() - 60 * 60 * 1000).toISOString();
  }
  if (range === 'Last 24h') {
    return new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
  }
  if (range === 'Last 7d') {
    return new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000).toISOString();
  }
  // Last 30d
  return new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000).toISOString();
}

// ---------------------------------------------------------------------------
// Metric card
// ---------------------------------------------------------------------------

function MetricCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="glass neon-edge rounded-lg p-4">
      <div className="text-2xl font-semibold text-violet-400 tabular-nums">{value}</div>
      <div className="text-zinc-400 text-xs uppercase tracking-wider mt-1">{label}</div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Breakdown bar row
// ---------------------------------------------------------------------------

function BreakdownRow({
  name,
  count,
  maxCount,
}: {
  name: string;
  count: number;
  maxCount: number;
}) {
  const widthPct = maxCount > 0 ? (count / maxCount) * 100 : 0;
  return (
    <div className="flex items-center gap-3">
      <span className="text-zinc-300 text-sm w-40 truncate shrink-0" title={name}>
        {name}
      </span>
      <div className="flex-1 bg-zinc-800 rounded h-2 overflow-hidden">
        <div
          className="h-full bg-violet-500 rounded"
          style={{ width: `${widthPct}%` }}
        />
      </div>
      <span className="text-zinc-500 text-xs tabular-nums w-10 text-right shrink-0">
        {count}
      </span>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Sparkline (SVG cost-over-time)
// ---------------------------------------------------------------------------

interface SparkPoint {
  hour: number; // ms timestamp bucket
  cost: number;
}

function CostSparkline({ points }: { points: SparkPoint[] }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(600);

  useEffect(() => {
    if (!containerRef.current) return;
    const obs = new ResizeObserver((entries) => {
      const rect = entries[0]?.contentRect;
      if (rect) setWidth(rect.width);
    });
    obs.observe(containerRef.current);
    return () => obs.disconnect();
  }, []);

  const height = 60;
  const padX = 8;
  const padY = 8;
  const innerW = Math.max(width - padX * 2, 1);
  const innerH = height - padY * 2;

  const pathD = useMemo(() => {
    if (points.length < 2) {
      // Flat line in the middle
      return `M ${padX} ${height / 2} L ${width - padX} ${height / 2}`;
    }
    const minX = points[0].hour;
    const maxX = points[points.length - 1].hour;
    const xRange = maxX - minX || 1;
    const maxCost = Math.max(...points.map((p) => p.cost), 0.000001);

    const coords = points.map((p) => {
      const x = padX + ((p.hour - minX) / xRange) * innerW;
      const y = padY + (1 - p.cost / maxCost) * innerH;
      return `${x.toFixed(1)},${y.toFixed(1)}`;
    });

    return `M ${coords.join(' L ')}`;
  }, [points, width, innerW, innerH]);

  return (
    <div ref={containerRef} className="bg-zinc-900 border border-zinc-800 rounded-lg p-4">
      <p className="text-zinc-300 font-semibold text-sm uppercase tracking-wider mb-3">
        Cost Over Time
      </p>
      <svg
        width="100%"
        height={height}
        viewBox={`0 0 ${width} ${height}`}
        preserveAspectRatio="none"
        aria-hidden="true"
      >
        <path
          d={pathD}
          stroke="#8b5cf6"
          strokeWidth="2"
          fill="none"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main page
// ---------------------------------------------------------------------------

export default function StatsPage() {
  const [range, setRange] = useState<Range>('All time');
  const [stats, setStats] = useState<StatsData | null>(null);
  const [sparkPoints, setSparkPoints] = useState<SparkPoint[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async (r: Range) => {
    setLoading(true);
    try {
      const from = fromForRange(r);
      const params: Record<string, string> = {};
      if (from) params.from = from;

      const [statsData, tracesData] = await Promise.all([
        fetchStats(params),
        fetchTraces({ limit: '200', ...(from ? { from } : {}) }),
      ]);

      setStats(statsData);

      // Build hourly cost buckets from traces
      const traces: Trace[] = Array.isArray(tracesData) ? tracesData : [];
      const buckets = new Map<number, number>();
      for (const t of traces) {
        if (!t.created_at) continue;
        const d = new Date(t.created_at);
        if (isNaN(d.getTime())) continue;
        const bucket = Math.floor(d.getTime() / (3600 * 1000)) * (3600 * 1000);
        buckets.set(bucket, (buckets.get(bucket) ?? 0) + (t.cost ?? 0));
      }
      const sorted = Array.from(buckets.entries())
        .sort((a, b) => a[0] - b[0])
        .map(([hour, cost]) => ({ hour, cost }));
      setSparkPoints(sorted);
    } catch {
      // leave existing data intact on error
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load(range);
  }, [range, load]);

  const isEmpty =
    !loading && stats !== null && stats.trace_count === 0;

  // Breakdown sorted lists
  const byModelEntries = stats
    ? Object.entries(stats.by_model ?? {}).sort((a, b) => b[1] - a[1])
    : [];
  const byAgentEntries = stats
    ? Object.entries(stats.by_agent ?? {}).sort((a, b) => b[1] - a[1])
    : [];
  const maxModel = byModelEntries[0]?.[1] ?? 1;
  const maxAgent = byAgentEntries[0]?.[1] ?? 1;

  const avgCost =
    stats && stats.trace_count > 0
      ? stats.total_cost / stats.trace_count
      : 0;

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Time range selector */}
      <div className="flex gap-1 bg-zinc-800 rounded-lg p-1 w-fit">
        {RANGES.map((r) => (
          <button
            key={r}
            onClick={() => setRange(r)}
            className={`px-3 py-1.5 text-sm rounded transition-colors ${
              range === r
                ? 'bg-violet-500 text-white'
                : 'text-zinc-400 hover:text-zinc-200'
            }`}
          >
            {r}
          </button>
        ))}
      </div>

      {isEmpty ? (
        <EmptyState
          title="No data yet"
          description="Start capturing traces to see usage analytics."
        />
      ) : (
        <>
          {/* Metric cards */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <MetricCard
              label="Total Traces"
              value={stats ? stats.trace_count.toLocaleString() : '—'}
            />
            <MetricCard
              label="Total Cost"
              value={stats ? `$${stats.total_cost.toFixed(4)}` : '—'}
            />
            <MetricCard
              label="Total Tokens"
              value={stats ? stats.total_tokens.toLocaleString() : '—'}
            />
            <MetricCard
              label="Avg Cost / Request"
              value={stats ? `$${avgCost.toFixed(6)}` : '—'}
            />
          </div>

          {/* By Model */}
          {byModelEntries.length > 0 && (
            <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4">
              <p className="text-zinc-300 font-semibold text-sm uppercase tracking-wider mb-3">
                By Model
              </p>
              <div className="space-y-2">
                {byModelEntries.map(([model, count]) => (
                  <BreakdownRow
                    key={model}
                    name={model}
                    count={count}
                    maxCount={maxModel}
                  />
                ))}
              </div>
            </div>
          )}

          {/* By Agent */}
          {byAgentEntries.length > 0 && (
            <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4">
              <p className="text-zinc-300 font-semibold text-sm uppercase tracking-wider mb-3">
                By Agent
              </p>
              <div className="space-y-2">
                {byAgentEntries.map(([agent, count]) => (
                  <BreakdownRow
                    key={agent}
                    name={agent}
                    count={count}
                    maxCount={maxAgent}
                  />
                ))}
              </div>
            </div>
          )}

          {/* Cost sparkline */}
          <CostSparkline points={sparkPoints} />
        </>
      )}
    </div>
  );
}
