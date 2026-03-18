'use client';

import { useEffect, useState, useCallback } from 'react';
import { fetchTraces } from '@/lib/api';

interface Trace {
  id: string;
  timestamp: string;
  model: string;
  agent: string;
  tokens: number;
  cost: number;
  latency: number;
  status: number;
}

function statusColor(status: number): string {
  if (status === 429) return 'text-yellow-400';
  if (status >= 400) return 'text-red-400';
  if (status >= 200 && status < 300) return 'text-green-400';
  return 'text-zinc-400';
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

function formatCost(cost: number): string {
  if (cost < 0.01) return `$${(cost * 1000).toFixed(3)}m`;
  return `$${cost.toFixed(4)}`;
}

export default function Timeline() {
  const [traces, setTraces] = useState<Trace[]>([]);
  const [loading, setLoading] = useState(true);
  const [agentFilter, setAgentFilter] = useState('');
  const [modelFilter, setModelFilter] = useState('');
  const [searchFilter, setSearchFilter] = useState('');

  const load = useCallback(async () => {
    try {
      const params: Record<string, string> = {};
      if (agentFilter) params.agent = agentFilter;
      if (modelFilter) params.model = modelFilter;
      const data = await fetchTraces(Object.keys(params).length ? params : undefined);
      setTraces(Array.isArray(data) ? data : []);
    } catch {
      setTraces([]);
    } finally {
      setLoading(false);
    }
  }, [agentFilter, modelFilter]);

  useEffect(() => {
    setLoading(true);
    load();
  }, [load]);

  const filtered = traces.filter((t) => {
    if (!searchFilter) return true;
    const q = searchFilter.toLowerCase();
    return (
      t.model?.toLowerCase().includes(q) ||
      t.agent?.toLowerCase().includes(q) ||
      t.id?.toLowerCase().includes(q)
    );
  });

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
            ) : filtered.length === 0 ? (
              <tr>
                <td colSpan={7} className="px-4 py-8 text-center text-zinc-500">
                  No traces captured yet...
                </td>
              </tr>
            ) : (
              filtered.map((trace) => (
                <tr
                  key={trace.id}
                  className="border-t border-zinc-800 hover:bg-zinc-900/50 transition-colors"
                >
                  <td className="px-4 py-3 font-mono text-zinc-400 text-xs">
                    {formatTime(trace.timestamp)}
                  </td>
                  <td className="px-4 py-3 text-zinc-200">{trace.model}</td>
                  <td className="px-4 py-3 text-zinc-300">{trace.agent}</td>
                  <td className="px-4 py-3 text-right font-mono text-zinc-300">
                    {trace.tokens?.toLocaleString()}
                  </td>
                  <td className="px-4 py-3 text-right font-mono text-zinc-300">
                    {formatCost(trace.cost)}
                  </td>
                  <td className="px-4 py-3 text-right font-mono text-zinc-300">
                    {trace.latency?.toFixed(2)}s
                  </td>
                  <td className="px-4 py-3 text-center">
                    <span className={`font-mono font-medium ${statusColor(trace.status)}`}>
                      {trace.status}
                    </span>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
