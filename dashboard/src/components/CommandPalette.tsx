'use client';

import { useEffect, useRef, useState, useCallback } from 'react';
import { fetchTraces } from '@/lib/api';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface Trace {
  id: string;
  model: string;
  agent: string | null;
}

type ResultKind = 'command' | 'trace';

interface CommandResult {
  kind: 'command';
  id: string;
  icon: string;
  title: string;
  subtitle: string;
  hint: string;
  action: () => void;
}

interface TraceResult {
  kind: 'trace';
  id: string;
  icon: string;
  title: string;
  subtitle: string;
  hint: string;
  action: () => void;
}

type PaletteResult = CommandResult | TraceResult;

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function truncateId(id: string): string {
  if (id.length <= 12) return id;
  return `${id.slice(0, 8)}…${id.slice(-4)}`;
}

function matchesQuery(query: string, ...fields: (string | null | undefined)[]): boolean {
  if (!query) return true;
  const q = query.toLowerCase();
  return fields.some((f) => f?.toLowerCase().includes(q));
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function ResultRow({
  result,
  selected,
  onMouseEnter,
  onClick,
}: {
  result: PaletteResult;
  selected: boolean;
  onMouseEnter: () => void;
  onClick: () => void;
}) {
  return (
    <li
      role="option"
      aria-selected={selected}
      onMouseEnter={onMouseEnter}
      onClick={onClick}
      className={[
        'flex items-center gap-3 px-4 py-2.5 cursor-pointer select-none rounded-lg mx-1 transition-colors',
        selected
          ? 'bg-violet-600/20 text-zinc-100'
          : 'text-zinc-300 hover:bg-zinc-800/60',
      ].join(' ')}
    >
      {/* Icon */}
      <span className="text-base w-5 text-center flex-shrink-0 text-violet-400" aria-hidden="true">
        {result.icon}
      </span>

      {/* Title + subtitle */}
      <span className="flex flex-col min-w-0 flex-1">
        <span className="text-sm font-medium truncate leading-tight">{result.title}</span>
        {result.subtitle && (
          <span className="text-xs text-zinc-500 truncate leading-tight mt-0.5">
            {result.subtitle}
          </span>
        )}
      </span>

      {/* Keyboard hint */}
      {result.hint && (
        <kbd className="flex-shrink-0 bg-zinc-800 text-zinc-400 border border-zinc-700 rounded px-1.5 py-0.5 text-xs font-mono leading-tight">
          {result.hint}
        </kbd>
      )}
    </li>
  );
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

interface CommandPaletteProps {
  open: boolean;
  onClose: () => void;
  onOpenHelp: () => void;
}

const MAX_RESULTS = 8;

export default function CommandPalette({ open, onClose, onOpenHelp }: CommandPaletteProps) {
  const [query, setQuery] = useState('');
  const [traces, setTraces] = useState<Trace[]>([]);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  // Fetch recent traces once on mount (not per-open — palette might open often).
  useEffect(() => {
    fetchTraces({ limit: '20' })
      .then((data: Trace[]) => {
        if (Array.isArray(data)) setTraces(data);
      })
      .catch(() => {
        // Silently ignore — traces are non-critical for the palette to work.
      });
  }, []);

  // Reset query and selection each time palette opens; auto-focus input.
  useEffect(() => {
    if (open) {
      setQuery('');
      setSelectedIndex(0);
      // Defer focus to next tick so the element is visible.
      requestAnimationFrame(() => inputRef.current?.focus());
    }
  }, [open]);

  // ---------------------------------------------------------------------------
  // Build the static commands list (memoised so onOpenHelp reference changes
  // don't rebuild unnecessarily unless the function identity changes).
  // ---------------------------------------------------------------------------
  const buildCommands = useCallback((): CommandResult[] => [
    {
      kind: 'command',
      id: 'goto-traces',
      icon: '⚡',
      title: 'Go to Traces',
      subtitle: 'View all LLM traces',
      hint: '1',
      action: () => { window.location.href = '/'; },
    },
    {
      kind: 'command',
      id: 'goto-sessions',
      icon: '🗂',
      title: 'Go to Sessions',
      subtitle: 'Browse agent sessions',
      hint: '2',
      action: () => { window.location.href = '/sessions'; },
    },
    {
      kind: 'command',
      id: 'goto-diff',
      icon: '⚖',
      title: 'Go to Diff',
      subtitle: 'Compare two sessions',
      hint: '3',
      action: () => { window.location.href = '/diff'; },
    },
    {
      kind: 'command',
      id: 'goto-stats',
      icon: '📊',
      title: 'Go to Stats',
      subtitle: 'Token usage and cost analytics',
      hint: '4',
      action: () => { window.location.href = '/stats'; },
    },
    {
      kind: 'command',
      id: 'keyboard-shortcuts',
      icon: '⌨',
      title: 'Keyboard Shortcuts',
      subtitle: 'Show all available shortcuts',
      hint: '?',
      action: () => { onOpenHelp(); },
    },
  ], [onOpenHelp]);

  // ---------------------------------------------------------------------------
  // Filter results
  // ---------------------------------------------------------------------------
  const results: PaletteResult[] = (() => {
    const commands = buildCommands();
    const matchedCommands = commands.filter((c) =>
      matchesQuery(query, c.title, c.subtitle)
    );

    const matchedTraces: TraceResult[] = traces
      .filter((t) => matchesQuery(query, t.model, t.agent, t.id))
      .map((t) => ({
        kind: 'trace',
        id: t.id,
        icon: '◈',
        title: t.model,
        subtitle: `${t.agent ?? '—'} · ${truncateId(t.id)}`,
        hint: '',
        action: () => { window.location.href = '/'; },
      }));

    return [...matchedCommands, ...matchedTraces].slice(0, MAX_RESULTS);
  })();

  // Clamp selection after filter changes.
  const clampedIndex = Math.min(selectedIndex, Math.max(results.length - 1, 0));

  // ---------------------------------------------------------------------------
  // Keyboard navigation within palette
  // ---------------------------------------------------------------------------
  useEffect(() => {
    if (!open) return;

    function handleKeyDown(e: KeyboardEvent) {
      switch (e.key) {
        case 'ArrowDown':
          e.preventDefault();
          setSelectedIndex((i) => Math.min(i + 1, results.length - 1));
          break;
        case 'ArrowUp':
          e.preventDefault();
          setSelectedIndex((i) => Math.max(i - 1, 0));
          break;
        case 'Enter':
          e.preventDefault();
          if (results[clampedIndex]) {
            results[clampedIndex].action();
            onClose();
          }
          break;
        case 'Escape':
          e.preventDefault();
          onClose();
          break;
      }
    }

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, results, clampedIndex, onClose]);

  // Reset selection to 0 whenever the filtered list changes.
  useEffect(() => {
    setSelectedIndex(0);
  }, [query]);

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 bg-zinc-950/80 backdrop-blur-sm flex items-start justify-center z-50 pt-[15vh]"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-label="Command palette"
    >
      <div
        className="glass neon-edge rounded-xl max-w-lg w-full mx-4 overflow-hidden"
        style={{
          boxShadow:
            '0 0 0 1px rgba(139,92,246,0.15), 0 8px 32px rgba(139,92,246,0.12), 0 2px 8px rgba(0,0,0,0.6)',
        }}
        onClick={(e) => e.stopPropagation()}
      >
        {/* Search input */}
        <div className="flex items-center gap-3 px-4 py-3 border-b border-zinc-800/60">
          <svg
            className="w-4 h-4 text-violet-400 flex-shrink-0"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M21 21l-4.35-4.35M17 11A6 6 0 1 1 5 11a6 6 0 0 1 12 0z"
            />
          </svg>
          <input
            ref={inputRef}
            type="text"
            role="combobox"
            aria-autocomplete="list"
            aria-expanded={results.length > 0}
            aria-controls="command-palette-results"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search traces, agents, models..."
            className="flex-1 bg-transparent text-zinc-100 placeholder-zinc-500 text-sm outline-none"
          />
          <kbd className="flex-shrink-0 bg-zinc-800 text-zinc-500 border border-zinc-700 rounded px-1.5 py-0.5 text-xs font-mono">
            esc
          </kbd>
        </div>

        {/* Results list */}
        {results.length > 0 ? (
          <ul
            id="command-palette-results"
            role="listbox"
            aria-label="Command results"
            className="py-1.5 max-h-80 overflow-y-auto"
          >
            {results.map((result, i) => (
              <ResultRow
                key={result.id}
                result={result}
                selected={i === clampedIndex}
                onMouseEnter={() => setSelectedIndex(i)}
                onClick={() => {
                  result.action();
                  onClose();
                }}
              />
            ))}
          </ul>
        ) : (
          <div className="px-4 py-8 text-center text-zinc-500 text-sm">
            No results for &ldquo;{query}&rdquo;
          </div>
        )}

        {/* Footer hint */}
        <div className="px-4 py-2 border-t border-zinc-800/60 flex items-center gap-3 text-xs text-zinc-600">
          <span>
            <kbd className="bg-zinc-800 border border-zinc-700 rounded px-1 py-0.5 font-mono">↑</kbd>
            <kbd className="bg-zinc-800 border border-zinc-700 rounded px-1 py-0.5 font-mono ml-1">↓</kbd>
            {' '}navigate
          </span>
          <span>
            <kbd className="bg-zinc-800 border border-zinc-700 rounded px-1 py-0.5 font-mono">↵</kbd>
            {' '}select
          </span>
          <span>
            <kbd className="bg-zinc-800 border border-zinc-700 rounded px-1 py-0.5 font-mono">esc</kbd>
            {' '}close
          </span>
        </div>
      </div>
    </div>
  );
}
