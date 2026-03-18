'use client';

import { useEffect, useState, useCallback } from 'react';
import { fetchTraces } from '@/lib/api';
import DetailInspector from '@/components/DetailInspector';

interface Trace {
  id: string;
  created_at: string;
  model: string;
  agent: string | null;
  tokens_prompt: number | null;
  tokens_completion: number | null;
  cost: number | null;
  latency_ms: number | null;
  status_code: number;
}

function statusColor(status: number): string {
  if (status === 429) return 'text-yellow-400';
  if (status >= 400) return 'text-red-400';
  if (status >= 200 && status < 300) return 'text-green-400';
  return 'text-zinc-400';
}

function formatTime(iso: string): string {
  try {
    const d = new Date(iso);
    if (isNaN(d.getTime())) return '—';
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  } catch {
    return '—';
  }
}

function formatCost(cost: number | null): string {
  if (cost === null || cost === undefined) return '—';
  if (cost === 0) return '$0.00';
  return `$${cost.toFixed(6)}`;
}

function formatTokens(prompt: number | null, completion: number | null): string {
  const total = (prompt || 0) + (completion || 0);
  if (total === 0 && prompt === null && completion === null) return '—';
  return total.toLocaleString();
}

function formatLatency(ms: number | null): string {
  if (ms === null || ms === undefined) return '—';
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

export default function Timeline() {
  const [traces, setTraces] = useState<Trace[]>([]);
  const [loading, setLoading] = useState(true);
  const [agentFilter, setAgentFilter] = useState('');
  const [modelFilter, setModelFilter] = useState('');
  const [searchFilter, setSearchFilter] = useState('');
  const [selectedTraceId, setSelectedTraceId] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const params: Record<string, string> = {};
      if (agentFilter) params.agent = agentFilter;
      if (modelFilter) params.model = modelFilter;
      if (searchFilter) params.search = searchFilter;
      const data = await fetchTraces(Object.keys(params).length ? params : undefined);
      setTraces(Array.isArray(data) ? data : []);
    } catch {
      setTraces([]);
    } finally {
      setLoading(false);
    }
  }, [agentFilter, modelFilter, searchFilter]);

  useEffect(() => {
    setLoading(true);
    load();
  }, [load]);

  const inputClass =
    'bg-zinc-800 border border-zinc-700 text-zinc-100 placeholder-zinc-500 rounded px-3 py-1.5 text-sm focus:outline-none focus:border-zinc-500';

  return (
    <div>
      {/* Filters */}
      <div className="flex flex-wrap gap-3 mb-4">
        <input
          type="text"
          placeholder="Filter by agent..."
          value={agentFilter}
          onChange={(e) => setAgentFilter(e.target.value)}
          className={inputClass}
        />
        <input
          type="text"
          placeholder="Filter by model..."
          value={modelFilter}
          onChange={(e) => setModelFilter(e.target.value)}
          className={inputClass}
        />
        <input
          type="text"
          placeholder="Search..."
          value={searchFilter}
          onChange={(e) => setSearchFilter(e.target.value)}
          className={inputClass}
        />
      </div>

      {/* Table */}
      <div className="rounded-lg border border-zinc-800 overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="bg-zinc-900 text-zinc-400 text-left">
              <th className="px-4 py-3 font-medium">Time</th>
              <th className="px-4 py-3 font-medium">Model</th>
              <th className="px-4 py-3 font-medium">Agent</th>
              <th className="px-4 py-3 font-medium text-right">Tokens</th>
              <th className="px-4 py-3 font-medium text-right">Cost</th>
              <th className="px-4 py-3 font-medium text-right">Latency</th>
              <th className="px-4 py-3 font-medium text-center">Status</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={7} className="px-4 py-8 text-center text-zinc-500">
                  Loading...
                </td>
              </tr>
            ) : traces.length === 0 ? (
              <tr>
                <td colSpan={7} className="px-4 py-8 text-center text-zinc-500">
                  No traces captured yet. Point your agents at the proxy to get started.
                </td>
              </tr>
            ) : (
              traces.map((trace) => (
                <tr
                  key={trace.id}
                  onClick={() =>
                    setSelectedTraceId((prev) => (prev === trace.id ? null : trace.id))
                  }
                  className={`border-t border-zinc-800 cursor-pointer transition-colors ${
                    selectedTraceId === trace.id
                      ? 'bg-zinc-800/70'
                      : 'hover:bg-zinc-900/50'
                  }`}
                >
                  <td className="px-4 py-3 font-mono text-zinc-400 text-xs">
                    {formatTime(trace.created_at)}
                  </td>
                  <td className="px-4 py-3 text-zinc-200">{trace.model}</td>
                  <td className="px-4 py-3 text-zinc-300">{trace.agent || '—'}</td>
                  <td className="px-4 py-3 text-right font-mono text-zinc-300">
                    {formatTokens(trace.tokens_prompt, trace.tokens_completion)}
                  </td>
                  <td className="px-4 py-3 text-right font-mono text-zinc-300">
                    {formatCost(trace.cost)}
                  </td>
                  <td className="px-4 py-3 text-right font-mono text-zinc-300">
                    {formatLatency(trace.latency_ms)}
                  </td>
                  <td className="px-4 py-3 text-center">
                    <span className={`font-mono font-medium ${statusColor(trace.status_code)}`}>
                      {trace.status_code}
                    </span>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Detail Inspector */}
      {selectedTraceId && (
        <DetailInspector
          traceId={selectedTraceId}
          onClose={() => setSelectedTraceId(null)}
        />
      )}
    </div>
  );
}
