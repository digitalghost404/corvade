'use client';

import { useEffect, useRef, useState, useCallback } from 'react';
import { fetchTraces } from '@/lib/api';
import DetailInspector from '@/components/DetailInspector';
import SkeletonRows from '@/components/SkeletonRows';
import EmptyState from '@/components/EmptyState';
import { useKeyboard } from '@/hooks/useKeyboard';
import { getWS } from '@/lib/wsClient';

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

function urgencyClass(status: number): string {
  if (status === 429) return 'row-rate-limit';
  if (status >= 400) return 'row-error';
  return '';
}

function timeAgo(iso: string): string {
  try {
    const d = new Date(iso);
    if (isNaN(d.getTime())) return '—';
    const diffMs = Date.now() - d.getTime();
    const diffS = Math.floor(diffMs / 1000);
    if (diffS < 60) return `${diffS}s ago`;
    const diffM = Math.floor(diffS / 60);
    if (diffM < 60) return `${diffM}m ago`;
    const diffH = Math.floor(diffM / 60);
    if (diffH < 24) return `${diffH}h ago`;
    const diffD = Math.floor(diffH / 24);
    return `${diffD}d ago`;
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
  const [selectedIndex, setSelectedIndex] = useState(-1);
  // ID of the most recently arrived trace — drives the scanline animation
  const [scanTraceId, setScanTraceId] = useState<string | null>(null);
  const scanTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const searchRef = useRef<HTMLInputElement>(null);
  const rowRefs = useRef<(HTMLTableRowElement | null)[]>([]);

  // Debounced fetch: filters are applied 300ms after the last change
  const agentFilterRef = useRef(agentFilter);
  const modelFilterRef = useRef(modelFilter);
  const searchFilterRef = useRef(searchFilter);
  agentFilterRef.current = agentFilter;
  modelFilterRef.current = modelFilter;
  searchFilterRef.current = searchFilter;

  const load = useCallback(async () => {
    try {
      const params: Record<string, string> = {};
      if (agentFilterRef.current) params.agent = agentFilterRef.current;
      if (modelFilterRef.current) params.model = modelFilterRef.current;
      if (searchFilterRef.current) params.search = searchFilterRef.current;
      const data = await fetchTraces(Object.keys(params).length ? params : undefined);
      setTraces(Array.isArray(data) ? data : []);
    } catch {
      setTraces([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    setLoading(true);
    const timer = setTimeout(() => {
      load();
    }, 300);
    return () => clearTimeout(timer);
  }, [agentFilter, modelFilter, searchFilter, load]);

  // Subscribe to trace:new to trigger reload + scanline on the new row
  useEffect(() => {
    const handleTraceNew = (data: { id?: string }) => {
      load();
      if (data?.id) {
        setScanTraceId(data.id);
        if (scanTimer.current) clearTimeout(scanTimer.current);
        scanTimer.current = setTimeout(() => setScanTraceId(null), 700);
      }
    };

    const wsClient = getWS();
    wsClient.on('trace:new', handleTraceNew);

    return () => {
      wsClient.off('trace:new', handleTraceNew);
      if (scanTimer.current) clearTimeout(scanTimer.current);
    };
  }, [load]);

  // Scroll selected row into view
  useEffect(() => {
    if (selectedIndex >= 0 && rowRefs.current[selectedIndex]) {
      rowRefs.current[selectedIndex]?.scrollIntoView({ block: 'nearest' });
    }
  }, [selectedIndex]);

  const hasFilters = agentFilter || modelFilter || searchFilter;

  function clearFilters() {
    setAgentFilter('');
    setModelFilter('');
    setSearchFilter('');
  }

  const openInspector = useCallback(() => {
    if (selectedIndex >= 0 && selectedIndex < traces.length) {
      const trace = traces[selectedIndex];
      setSelectedTraceId((prev) => (prev === trace.id ? null : trace.id));
    }
  }, [selectedIndex, traces]);

  useKeyboard({
    j: (e) => {
      e.stopPropagation();
      setSelectedIndex((i) => Math.min(i + 1, traces.length - 1));
    },
    ArrowDown: (e) => {
      e.stopPropagation();
      setSelectedIndex((i) => Math.min(i + 1, traces.length - 1));
    },
    k: (e) => {
      e.stopPropagation();
      setSelectedIndex((i) => Math.max(i - 1, 0));
    },
    ArrowUp: (e) => {
      e.stopPropagation();
      setSelectedIndex((i) => Math.max(i - 1, 0));
    },
    Enter: (e) => {
      e.stopPropagation();
      openInspector();
    },
    ' ': (e) => {
      e.stopPropagation();
      openInspector();
    },
    Escape: (e) => {
      e.stopPropagation();
      setSelectedTraceId(null);
      setSelectedIndex(-1);
    },
    '/': (e) => {
      e.stopPropagation();
      e.preventDefault();
      searchRef.current?.focus();
    },
  });

  const inputClass =
    'bg-zinc-800 border border-zinc-700 text-zinc-100 placeholder-zinc-500 rounded px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent';

  return (
    <div>
      {/* Filters */}
      <div className="flex flex-wrap gap-3 mb-4 items-center">
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
          ref={searchRef}
          type="text"
          placeholder="Search..."
          value={searchFilter}
          onChange={(e) => setSearchFilter(e.target.value)}
          className={inputClass}
        />
        {hasFilters && (
          <button
            type="button"
            onClick={clearFilters}
            className="text-zinc-500 hover:text-zinc-300 text-sm cursor-pointer"
          >
            Clear
          </button>
        )}
      </div>

      {/* Empty state rendered outside the table */}
      {!loading && traces.length === 0 ? (
        <EmptyState
          title="No traces yet"
          description="Point your agents at the proxy to get started"
          code="OPENAI_BASE_URL=http://localhost:4400/v1"
        />
      ) : (
        /* Table */
        <div className="glass neon-edge rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-zinc-900 text-zinc-400 text-left">
                <th className="px-4 py-3 font-medium">Time</th>
                <th className="px-4 py-3 font-medium">Model</th>
                <th className="px-4 py-3 font-medium">Agent</th>
                <th className="px-4 py-3 font-medium text-right hidden sm:table-cell">Tokens</th>
                <th className="px-4 py-3 font-medium text-right">Cost</th>
                <th className="px-4 py-3 font-medium text-right hidden sm:table-cell">Latency</th>
                <th className="px-4 py-3 font-medium text-center">Status</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <SkeletonRows rows={5} />
              ) : (
                traces.map((trace, index) => {
                  const isSelected = selectedTraceId === trace.id;
                  const isKeyboardSelected = selectedIndex === index;
                  const isEven = index % 2 === 0;
                  const isScanning = scanTraceId === trace.id;
                  const errorClass = urgencyClass(trace.status_code);

                  let rowClass =
                    'border-t border-zinc-800 cursor-pointer transition-all duration-100 border-l-2 ';

                  if (isSelected) {
                    rowClass += 'bg-zinc-800/70 border-l-violet-500';
                  } else {
                    rowClass +=
                      (isEven ? 'bg-zinc-950 ' : 'bg-zinc-900/30 ') +
                      (isKeyboardSelected
                        ? 'border-l-violet-500 bg-zinc-800/50'
                        : 'border-l-transparent hover:bg-zinc-800/50 hover:border-l-violet-500');
                  }

                  if (isScanning) rowClass += ' trace-scan';
                  if (errorClass) rowClass += ` ${errorClass}`;

                  return (
                    <tr
                      key={trace.id}
                      ref={(el) => { rowRefs.current[index] = el; }}
                      onClick={() => {
                        setSelectedIndex(index);
                        setSelectedTraceId((prev) => (prev === trace.id ? null : trace.id));
                      }}
                      className={rowClass}
                    >
                      <td
                        className="px-4 py-2 font-mono text-zinc-400 text-xs"
                        title={trace.created_at}
                      >
                        {timeAgo(trace.created_at)}
                      </td>
                      <td className="px-4 py-2 text-zinc-200">{trace.model}</td>
                      <td className="px-4 py-2 text-zinc-300">
                        {trace.agent ?? <span className="text-zinc-500">—</span>}
                      </td>
                      <td className="px-4 py-2 text-right font-mono text-zinc-300 hidden sm:table-cell">
                        {formatTokens(trace.tokens_prompt, trace.tokens_completion)}
                      </td>
                      <td className="px-4 py-2 text-right font-mono text-zinc-300">
                        {formatCost(trace.cost)}
                      </td>
                      <td className="px-4 py-2 text-right font-mono text-zinc-300 hidden sm:table-cell">
                        {formatLatency(trace.latency_ms)}
                      </td>
                      <td className="px-4 py-2 text-center">
                        <span className={`font-mono font-medium ${statusColor(trace.status_code)}`}>
                          {trace.status_code}
                        </span>
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
      )}

      {/* Detail Inspector */}
      {selectedTraceId && (
        <DetailInspector
          traceId={selectedTraceId}
          onClose={() => {
            setSelectedTraceId(null);
            setSelectedIndex(-1);
          }}
        />
      )}
    </div>
  );
}
