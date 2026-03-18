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

function matchScoreColor(score: number): string {
  if (score >= 0.8) return 'text-emerald-400';
  if (score >= 0.5) return 'text-yellow-400';
  return 'text-red-400';
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

  const totalNodes =
    result.aligned_nodes.length + result.left_only.length + result.right_only.length;

  return (
    <div className="space-y-6">
      {/* Summary bar */}
      <div className="flex flex-wrap gap-4 text-sm text-zinc-400 border border-zinc-800 rounded-lg px-4 py-3 bg-zinc-950">
        <span>
          <span className="text-zinc-200 font-semibold">{totalNodes}</span> total nodes
        </span>
        <span>
          <span className="text-emerald-400 font-semibold">{result.aligned_nodes.length}</span> matched
        </span>
        <span>
          <span className="text-red-400 font-semibold">{result.left_only.length}</span> left-only
        </span>
        <span>
          <span className="text-green-400 font-semibold">{result.right_only.length}</span> right-only
        </span>
        <span>
          <span className="text-yellow-400 font-semibold">{result.divergence_points.length}</span> divergence points
        </span>
      </div>

      {/* Aligned pairs */}
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
                <div className={`text-xs font-mono font-semibold whitespace-nowrap ${matchScoreColor(pair.match_score)}`}>
                  {(pair.match_score * 100).toFixed(0)}%
                </div>
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
              <NodePill
                key={node.id}
                node={node}
                className="border-yellow-700/60 bg-yellow-950/20"
              />
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
