'use client';

import { useEffect } from 'react';

interface ShortcutRow {
  keys: string[];
  description: string;
}

interface ShortcutGroup {
  heading: string;
  rows: ShortcutRow[];
}

const SHORTCUT_GROUPS: ShortcutGroup[] = [
  {
    heading: 'Global',
    rows: [
      { keys: ['1', '2', '3', '4'], description: 'Switch tabs' },
      { keys: ['?'], description: 'Toggle this help overlay' },
    ],
  },
  {
    heading: 'Traces',
    rows: [
      { keys: ['j', 'k'], description: 'Navigate trace list' },
      { keys: ['Enter', 'Space'], description: 'Toggle inspector' },
      { keys: ['Esc'], description: 'Close inspector' },
      { keys: ['/'], description: 'Focus search' },
    ],
  },
  {
    heading: 'Inspector',
    rows: [{ keys: ['—'], description: 'Reserved for future shortcuts' }],
  },
];

function Kbd({ children }: { children: string }) {
  return (
    <kbd className="bg-zinc-700 text-zinc-200 rounded px-2 py-0.5 text-xs font-mono">
      {children}
    </kbd>
  );
}

interface KeyboardHelpProps {
  open: boolean;
  onClose: () => void;
}

export default function KeyboardHelp({ open, onClose }: KeyboardHelpProps) {
  useEffect(() => {
    if (!open) return;
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose();
    }
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 bg-zinc-950/80 flex items-center justify-center z-50"
      style={{ transition: 'opacity 100ms', opacity: open ? 1 : 0 }}
      onClick={onClose}
      aria-modal="true"
      role="dialog"
      aria-label="Keyboard shortcuts"
    >
      <div
        className="bg-zinc-900 border border-zinc-800 rounded-lg p-6 max-w-md w-full mx-4"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="text-zinc-100 text-base font-semibold mb-4">
          Keyboard Shortcuts
        </h2>

        <div className="space-y-5">
          {SHORTCUT_GROUPS.map((group) => (
            <section key={group.heading}>
              <h3 className="text-zinc-500 text-xs font-medium uppercase tracking-wider mb-2">
                {group.heading}
              </h3>
              <ul className="space-y-2">
                {group.rows.map((row) => (
                  <li
                    key={row.description}
                    className="flex items-center justify-between gap-4"
                  >
                    <span className="flex items-center gap-1 flex-shrink-0">
                      {row.keys.map((k) => (
                        <Kbd key={k}>{k}</Kbd>
                      ))}
                    </span>
                    <span className="text-zinc-400 text-sm text-right">
                      {row.description}
                    </span>
                  </li>
                ))}
              </ul>
            </section>
          ))}
        </div>
      </div>
    </div>
  );
}
