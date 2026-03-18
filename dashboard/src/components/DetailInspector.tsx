'use client';

import { useEffect, useState } from 'react';
import { fetchTrace } from '@/lib/api';

interface TraceDetail {
  id: string;
  timestamp: string;
  model: string;
  provider?: string;
  agent?: string;
  status: number;
  latency?: number;
  ttft?: number;
  cost?: number;
  tokens_in?: number;
  tokens_out?: number;
  tokens_cached?: number;
  request?: unknown;
  response?: unknown;
}

interface Props {
  traceId: string;
  onClose: () => void;
}

type Tab = 'request' | 'response';

function statusColor(status: number): string {
  if (status === 429) return 'text-yellow-400 bg-yellow-400/10';
  if (status >= 400) return 'text-red-400 bg-red-400/10';
  if (status >= 200 && status < 300) return 'text-green-400 bg-green-400/10';
  return 'text-zinc-400 bg-zinc-800';
}

function MetricPill({ label, value }: { label: string; value: string | number | undefined }) {
  return (
    <div className="flex flex-col gap-0.5">
      <span className="text-xs text-zinc-500 uppercase tracking-wide">{label}</span>
      <span className="text-sm font-mono text-zinc-200">
        {value ?? <span className="text-zinc-600">—</span>}
      </span>
    </div>
  );
}

function JsonViewer({ data }: { data: unknown }) {
  if (data === undefined || data === null) {
    return (
      <div className="px-4 py-8 text-center text-zinc-600 text-sm">No data available.</div>
    );
  }

  let formatted: string;
  try {
    formatted = JSON.stringify(data, null, 2);
  } catch {
    formatted = String(data);
  }

  // Simple syntax highlighting using CSS classes applied to tokens
  // We parse the JSON string into spans for color, using a replace approach
  // This avoids any external dependency.
  const highlighted = formatted
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    // strings (keys and values)
    .replace(
      /("(\\u[a-fA-F0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?)/g,
      (match) => {
        if (/:$/.test(match.trimEnd())) {
          return `<span class="text-sky-400">${match}</span>`;
        }
        return `<span class="text-amber-300">${match}</span>`;
      },
    )
    // numbers
    .replace(/\b(-?\d+\.?\d*([eE][+-]?\d+)?)\b/g, '<span class="text-violet-400">$1</span>')
    // booleans and null
    .replace(/\b(true|false|null)\b/g, '<span class="text-emerald-400">$1</span>');

  return (
    <pre
      className="text-xs font-mono leading-relaxed text-zinc-300 whitespace-pre-wrap break-all p-4 overflow-auto max-h-[420px]"
      // eslint-disable-next-line react/no-danger
      dangerouslySetInnerHTML={{ __html: highlighted }}
    />
  );
}

export default function DetailInspector({ traceId, onClose }: Props) {
  const [trace, setTrace] = useState<TraceDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [tab, setTab] = useState<Tab>('request');

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    setTrace(null);

    fetchTrace(traceId)
      .then((data: TraceDetail) => {
        if (!cancelled) setTrace(data);
      })
      .catch((err: unknown) => {
        if (!cancelled)
          setError(err instanceof Error ? err.message : 'Failed to load trace');
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [traceId]);

  // Close on Escape key
  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose();
    }
    window.addEventListener('keydown', handleKey);
    return () => window.removeEventListener('keydown', handleKey);
  }, [onClose]);

  const truncatedId = traceId.length > 12 ? `${traceId.slice(0, 8)}…` : traceId;

  const tabClass = (active: boolean) =>
    `px-4 py-2 text-sm font-medium transition-colors border-b-2 ${
      active
        ? 'text-zinc-100 border-sky-500'
        : 'text-zinc-400 border-transparent hover:text-zinc-200 hover:border-zinc-600'
    }`;

  return (
    <div className="mt-4 rounded-lg border border-zinc-700 bg-zinc-950 overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between gap-3 px-4 py-3 bg-zinc-900 border-b border-zinc-800">
        <div className="flex items-center gap-3 min-w-0">
          <span className="text-zinc-100 font-medium truncate">
            {loading || !trace ? '\u00a0' : trace.model}
          </span>
          {trace?.provider && (
            <span className="text-xs text-zinc-500 bg-zinc-800 border border-zinc-700 rounded px-2 py-0.5 shrink-0">
              {trace.provider}
            </span>
          )}
          <span
            className="text-xs font-mono text-zinc-600 shrink-0"
            title={traceId}
          >
            #{truncatedId}
          </span>
        </div>
        <button
          onClick={onClose}
          aria-label="Close detail inspector"
          className="shrink-0 text-zinc-500 hover:text-zinc-200 transition-colors text-lg leading-none px-1"
        >
          &times;
        </button>
      </div>

      {/* Loading / Error states */}
      {loading && (
        <div className="px-4 py-10 text-center text-zinc-500 text-sm">Loading trace...</div>
      )}

      {!loading && error && (
        <div className="px-4 py-10 text-center text-red-400 text-sm">{error}</div>
      )}

      {!loading && !error && trace && (
        <>
          {/* Metrics bar */}
          <div className="flex flex-wrap gap-x-6 gap-y-3 px-4 py-3 border-b border-zinc-800 bg-zinc-900/40">
            <MetricPill label="Tokens in" value={trace.tokens_in?.toLocaleString()} />
            <MetricPill label="Tokens out" value={trace.tokens_out?.toLocaleString()} />
            <MetricPill label="Cached" value={trace.tokens_cached?.toLocaleString()} />
            <MetricPill
              label="Cost"
              value={
                trace.cost !== undefined
                  ? trace.cost < 0.01
                    ? `$${(trace.cost * 1000).toFixed(3)}m`
                    : `$${trace.cost.toFixed(4)}`
                  : undefined
              }
            />
            <MetricPill
              label="Latency"
              value={trace.latency !== undefined ? `${trace.latency.toFixed(2)}s` : undefined}
            />
            <MetricPill
              label="TTFT"
              value={trace.ttft !== undefined ? `${(trace.ttft * 1000).toFixed(0)}ms` : undefined}
            />
            <div className="flex flex-col gap-0.5">
              <span className="text-xs text-zinc-500 uppercase tracking-wide">Status</span>
              <span
                className={`text-sm font-mono font-semibold px-2 py-0.5 rounded w-fit ${statusColor(trace.status)}`}
              >
                {trace.status}
              </span>
            </div>
          </div>

          {/* Tabs */}
          <div className="flex border-b border-zinc-800 bg-zinc-900/20">
            <button
              className={tabClass(tab === 'request')}
              onClick={() => setTab('request')}
            >
              Request
            </button>
            <button
              className={tabClass(tab === 'response')}
              onClick={() => setTab('response')}
            >
              Response
            </button>
          </div>

          {/* JSON viewer */}
          <div className="bg-zinc-950">
            <JsonViewer data={tab === 'request' ? trace.request : trace.response} />
          </div>
        </>
      )}
    </div>
  );
}
