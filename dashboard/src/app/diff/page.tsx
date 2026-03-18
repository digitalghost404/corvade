'use client';

import { useEffect, useRef, useState } from 'react';
import { fetchSessions } from '@/lib/api';
import SessionDiff from '@/components/SessionDiff';
import EmptyState from '@/components/EmptyState';

interface Session {
  id: string;
  agent?: string | null;
  trace_count?: number;
  created_at?: string;
  started_at?: string;
}

function timeAgo(iso: string | undefined | null): string {
  if (!iso) return '—';
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

interface SessionComboboxProps {
  label: string;
  value: string;
  onChange: (val: string) => void;
  sessions: Session[];
}

function SessionCombobox({ label, value, onChange, sessions }: SessionComboboxProps) {
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const filtered = sessions.filter((s) => {
    if (!value.trim()) return true;
    const q = value.trim().toLowerCase();
    return (
      s.id.toLowerCase().startsWith(q) ||
      (s.agent ?? '').toLowerCase().includes(q)
    );
  });

  // Close on outside click
  useEffect(() => {
    function handleMouseDown(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener('mousedown', handleMouseDown);
    return () => document.removeEventListener('mousedown', handleMouseDown);
  }, []);

  function handleSelect(session: Session) {
    onChange(session.id);
    setOpen(false);
  }

  return (
    <div ref={containerRef} className="flex-1 relative">
      <label className="block text-xs text-zinc-400 mb-1">{label}</label>
      <input
        type="text"
        value={value}
        onChange={(e) => {
          onChange(e.target.value);
          setOpen(true);
        }}
        onFocus={() => setOpen(true)}
        placeholder="Search by agent name or paste session ID…"
        className="w-full bg-zinc-900 border border-zinc-700 rounded-md px-3 py-2 text-sm text-zinc-100 placeholder-zinc-600 focus:outline-none focus:ring-1 focus:ring-zinc-500"
      />
      {open && filtered.length > 0 && (
        <ul className="absolute z-20 top-full mt-1 w-full bg-zinc-900 border border-zinc-700 rounded-md shadow-lg max-h-56 overflow-y-auto">
          {filtered.map((s) => (
            <li key={s.id}>
              <button
                type="button"
                onMouseDown={() => handleSelect(s)}
                className="w-full text-left px-3 py-2 hover:bg-zinc-800 transition-colors"
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="text-zinc-200 text-sm font-medium truncate">
                    {s.agent ?? 'Unknown agent'}
                  </span>
                  <span className="text-zinc-500 text-xs shrink-0">
                    {s.trace_count ?? 0} traces · {timeAgo(s.created_at ?? s.started_at)}
                  </span>
                </div>
                <div className="text-zinc-500 text-xs font-mono truncate mt-0.5">{s.id}</div>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

export default function DiffPage() {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [inputA, setInputA] = useState('');
  const [inputB, setInputB] = useState('');
  const [sessionA, setSessionA] = useState('');
  const [sessionB, setSessionB] = useState('');

  useEffect(() => {
    fetchSessions().then((data: Session[]) => {
      if (Array.isArray(data)) setSessions(data);
    }).catch(() => {/* silently ignore — dropdown is best-effort */});
  }, []);

  function handleCompare() {
    const a = inputA.trim();
    const b = inputB.trim();
    if (a && b) {
      setSessionA(a);
      setSessionB(b);
    }
  }

  const hasResult = sessionA && sessionB;

  return (
    <div className="p-6 max-w-7xl mx-auto">
      <div className="space-y-6">
        {/* Pickers row */}
        <div className="glass rounded-lg p-4 flex flex-col sm:flex-row gap-3 items-end">
          <SessionCombobox
            label="Session A"
            value={inputA}
            onChange={setInputA}
            sessions={sessions}
          />

          <span className="text-zinc-500 text-sm pb-2 shrink-0">vs</span>

          <SessionCombobox
            label="Session B"
            value={inputB}
            onChange={setInputB}
            sessions={sessions}
          />

          <button
            onClick={handleCompare}
            disabled={!inputA.trim() || !inputB.trim()}
            className="bg-violet-500 hover:bg-violet-600 text-white px-4 py-1.5 rounded text-sm disabled:opacity-50 transition-colors shrink-0 self-end mb-0.5"
          >
            Compare
          </button>
        </div>

        {/* Result or empty state */}
        {hasResult ? (
          <div className="glass rounded-lg overflow-hidden">
            <SessionDiff sessionA={sessionA} sessionB={sessionB} />
          </div>
        ) : (
          <EmptyState
            title="Compare agent sessions"
            description="Compare two agent sessions to see where they diverged. Run the same agent twice with different models or prompts."
          />
        )}
      </div>
    </div>
  );
}
