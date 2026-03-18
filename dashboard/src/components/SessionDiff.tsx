'use client';

import { useEffect, useState } from 'react';
import { fetchSessionDiff } from '@/lib/api';

interface GraphNode {
  id: string;
  type: string;
  agent?: string;
  step?: string;
  model?: string;
  tokens?: number;
  cost?: number;
  confidence?: number;
}

interface AlignedPair {
  left: GraphNode;
  right: GraphNode;
  match_score: number;
}

interface DiffResult {
  aligned_nodes: AlignedPair[];
  left_only: GraphNode[];
  right_only: GraphNode[];
  divergence_points: GraphNode[];
}

interface Props {
  sessionA: string;
  sessionB: string;
}

function NodePill({ node, className }: { node: GraphNode; className?: string }) {
  return (
    <div className={`rounded-md border px-3 py-2 text-sm ${className ?? ''}`}>
      <div className="font-medium text-zinc-100 truncate">{node.step ?? node.type}</div>
      {node.agent && (
        <div className="text-zinc-400 text-xs truncate">agent: {node.agent}</div>
      )}
      {node.model && (
        <div className="text-zinc-500 text-xs truncate">{node.model}</div>
      )}
    </div>
  );
}

function MatchBar({ score }: { score: number }) {
  const pct = Math.round(score * 100);
  const barColor = score >= 0.8 ? 'bg-green-500' : 'bg-yellow-500';
  const textColor = score >= 0.8 ? 'text-emerald-400' : 'text-yellow-400';
  return (
    <div className="flex flex-col items-center gap-1 min-w-[48px]">
      <span className={`text-xs font-mono font-semibold ${textColor}`}>{pct}%</span>
      <div className="w-10 h-1.5 bg-zinc-700 rounded-full overflow-hidden">
        <div
          className={`h-full rounded-full ${barColor}`}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  );
}

export default function SessionDiff({ sessionA, sessionB }: Props) {
  const [result, setResult] = useState<DiffResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    setResult(null);

    fetchSessionDiff(sessionA, sessionB)
      .then((data: DiffResult) => {
        if (!cancelled) setResult(data);
      })
      .catch((err: unknown) => {
        if (!cancelled)
          setError(err instanceof Error ? err.message : 'Failed to load diff');
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [sessionA, sessionB]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-40 bg-zinc-950 border border-zinc-800 rounded-lg text-zinc-500">
        Loading diff…
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-40 bg-zinc-950 border border-zinc-800 rounded-lg text-red-400">
        {error}
      </div>
    );
  }

  if (!result) return null;

  const matchedCount = result.aligned_nodes.length;
  const divergedCount = result.divergence_points.length;
  const leftOnlyCount = result.left_only.length;
  const rightOnlyCount = result.right_only.length;
  const totalNodes = matchedCount + leftOnlyCount + rightOnlyCount;

  return (
    <div className="space-y-6">
      {/* Summary bar */}
      <p className="text-zinc-400 text-sm">
        <span className="text-emerald-400 font-semibold">{matchedCount} matched</span>
        {' · '}
        <span className="text-yellow-400 font-semibold">{divergedCount} diverged</span>
        {' · '}
        <span className="text-red-400 font-semibold">{leftOnlyCount} left-only</span>
        {' · '}
        <span className="text-green-400 font-semibold">{rightOnlyCount} right-only</span>
      </p>

      {/* Aligned / matched pairs */}
      {result.aligned_nodes.length > 0 && (
        <section>
          <h3 className="text-zinc-300 font-semibold text-sm mb-3">Matched Pairs</h3>
          <div className="space-y-2">
            {result.aligned_nodes.map((pair, i) => (
              <div
                key={i}
                className="grid grid-cols-[1fr_auto_1fr] gap-3 items-center bg-zinc-900 border border-zinc-800 rounded-lg px-4 py-3"
              >
                <NodePill node={pair.left} className="border-zinc-700 bg-zinc-950" />
                <MatchBar score={pair.match_score} />
                <NodePill node={pair.right} className="border-zinc-700 bg-zinc-950" />
              </div>
            ))}
          </div>
        </section>
      )}

      {/* Divergence points */}
      {result.divergence_points.length > 0 && (
        <section>
          <h3 className="text-yellow-400 font-semibold text-sm mb-3">Divergence Points</h3>
          <div className="space-y-2">
            {result.divergence_points.map((node) => (
              <div
                key={node.id}
                className="flex items-start gap-2 rounded-md border border-yellow-700/60 bg-yellow-950/20 px-3 py-2"
              >
                <span
                  className="inline-block w-2 h-2 rounded-full bg-violet-500 pulse-dot mt-1.5 shrink-0"
                  aria-hidden="true"
                />
                <NodePill node={node} className="border-0 bg-transparent p-0 flex-1" />
              </div>
            ))}
          </div>
        </section>
      )}

      {/* Left-only */}
      {result.left_only.length > 0 && (
        <section>
          <h3 className="text-red-400 font-semibold text-sm mb-3">
            Left Only — <span className="font-mono text-zinc-500">{sessionA}</span>
          </h3>
          <div className="space-y-2">
            {result.left_only.map((node) => (
              <NodePill
                key={node.id}
                node={node}
                className="border-red-700/60 bg-red-950/20"
              />
            ))}
          </div>
        </section>
      )}

      {/* Right-only */}
      {result.right_only.length > 0 && (
        <section>
          <h3 className="text-green-400 font-semibold text-sm mb-3">
            Right Only — <span className="font-mono text-zinc-500">{sessionB}</span>
          </h3>
          <div className="space-y-2">
            {result.right_only.map((node) => (
              <NodePill
                key={node.id}
                node={node}
                className="border-green-700/60 bg-green-950/20"
              />
            ))}
          </div>
        </section>
      )}

      {totalNodes === 0 && (
        <div className="flex items-center justify-center h-32 text-zinc-500 text-sm">
          Both sessions have no graph nodes.
        </div>
      )}
    </div>
  );
}
