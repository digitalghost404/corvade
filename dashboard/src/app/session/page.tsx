'use client';

import { Suspense, useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import TopologyGraph from '@/components/TopologyGraph';
import NarrativeView from '@/components/NarrativeView';
import { fetchSession, fetchSessionGraph } from '@/lib/api';
import type { GraphNode } from '@/lib/narrative';

interface SessionMeta {
  id: string;
  agent: string | null;
  status: 'active' | 'completed' | 'error';
  start_time: string;
  end_time: string | null;
  total_cost: number;
}

type RightPanel = 'narrative' | 'traces';

const STATUS_STYLES: Record<string, string> = {
  active: 'bg-violet-500/20 text-violet-400',
  completed: 'bg-green-500/20 text-green-400',
  error: 'bg-red-500/20 text-red-400',
};

function formatTime(iso: string): string {
  const d = new Date(iso);
  const hh = d.getHours().toString().padStart(2, '0');
  const mm = d.getMinutes().toString().padStart(2, '0');
  return `${hh}:${mm}`;
}

function SessionContent() {
  const searchParams = useSearchParams();
  const id = searchParams.get('id') ?? '';

  const [session, setSession] = useState<SessionMeta | null>(null);
  const [graphNodes, setGraphNodes] = useState<GraphNode[]>([]);
  const [rightPanel, setRightPanel] = useState<RightPanel>('narrative');

  useEffect(() => {
    if (!id) return;
    fetchSession(id)
      .then((data: SessionMeta) => setSession(data))
      .catch(() => setSession(null));
    fetchSessionGraph(id)
      .then((data: { nodes?: GraphNode[] }) => setGraphNodes(data.nodes ?? []))
      .catch(() => setGraphNodes([]));
  }, [id]);

  if (!id) {
    return (
      <div className="p-6 max-w-7xl mx-auto">
        <div className="flex items-center justify-center h-96 bg-zinc-950 border border-zinc-800 rounded-lg text-zinc-500">
          No session ID provided.
        </div>
      </div>
    );
  }

  const timeRange = session
    ? session.end_time
      ? `${formatTime(session.start_time)} → ${formatTime(session.end_time)}`
      : `${formatTime(session.start_time)} → ongoing`
    : null;

  return (
    <div className="p-6 max-w-7xl mx-auto">
      {/* Session header bar */}
      <div className="flex flex-wrap items-center gap-3 mb-6">
        <span
          className={
            session?.agent
              ? 'text-xl font-semibold text-zinc-50'
              : 'text-xl font-semibold text-zinc-500 italic'
          }
        >
          {session?.agent ?? 'Unknown Agent'}
        </span>

        <span className="text-zinc-500 font-mono text-xs truncate max-w-[160px]">
          {id.slice(0, 12)}
        </span>

        {session && (
          <span
            className={`text-xs px-2 py-0.5 rounded-full ${STATUS_STYLES[session.status] ?? STATUS_STYLES.error}`}
          >
            {session.status}
          </span>
        )}

        {timeRange && (
          <span className="text-xs text-zinc-500">{timeRange}</span>
        )}

        {session && (
          <span className="text-xs text-zinc-400 ml-auto">
            ${session.total_cost.toFixed(4)}
          </span>
        )}
      </div>

      {/* Two-panel layout */}
      <div className="flex flex-col lg:flex-row gap-4">
        {/* Left panel: topology graph */}
        <div className="min-w-0" style={{ flex: 3 }}>
          <div
            className="bg-zinc-950 border border-zinc-800 rounded-lg overflow-hidden"
            style={{ height: 'calc(100vh - 200px)' }}
          >
            <TopologyGraph sessionId={id} />
          </div>
        </div>

        {/* Right panel: segmented control + content */}
        <div className="min-w-0 flex flex-col gap-3" style={{ flex: 2 }}>
          {/* Segmented control */}
          <div className="bg-zinc-800 rounded-lg overflow-hidden flex shrink-0">
            <button
              type="button"
              onClick={() => setRightPanel('narrative')}
              className={`px-4 py-2 text-sm transition-colors cursor-pointer ${
                rightPanel === 'narrative'
                  ? 'bg-violet-500 text-white'
                  : 'text-zinc-400 hover:text-zinc-200'
              }`}
            >
              Narrative
            </button>
            <button
              type="button"
              onClick={() => setRightPanel('traces')}
              className={`px-4 py-2 text-sm transition-colors cursor-pointer ${
                rightPanel === 'traces'
                  ? 'bg-violet-500 text-white'
                  : 'text-zinc-400 hover:text-zinc-200'
              }`}
            >
              Traces
            </button>
          </div>

          {/* Panel content */}
          <div className="overflow-y-auto" style={{ height: 'calc(100vh - 260px)' }}>
            {rightPanel === 'narrative' ? (
              <NarrativeView nodes={graphNodes} />
            ) : (
              <div className="bg-zinc-900 border border-zinc-800 rounded-lg overflow-hidden">
                <div className="px-4 py-3 border-b border-zinc-800">
                  <span className="text-xs font-semibold tracking-widest uppercase text-zinc-400">
                    Traces
                  </span>
                </div>
                {graphNodes.length === 0 ? (
                  <div className="px-4 py-8 text-center text-zinc-500 text-sm">
                    No traces to display.
                  </div>
                ) : (
                  <ol className="divide-y divide-zinc-800">
                    {graphNodes.map((node, i) => (
                      <li
                        key={node.id}
                        className="flex items-center gap-3 px-4 py-3"
                      >
                        <span className="text-xs text-zinc-600 font-mono w-5 shrink-0 text-right">
                          {i + 1}
                        </span>
                        <span className="text-sm text-zinc-300 truncate">
                          {node.step ?? node.type}
                        </span>
                        {node.model && (
                          <span className="text-xs text-zinc-500 truncate ml-auto">
                            {node.model}
                          </span>
                        )}
                      </li>
                    ))}
                  </ol>
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

export default function SessionPage() {
  return (
    <Suspense
      fallback={
        <div className="p-6 max-w-7xl mx-auto">
          <div className="flex items-center justify-center h-96 text-zinc-500">
            Loading...
          </div>
        </div>
      }
    >
      <SessionContent />
    </Suspense>
  );
}
