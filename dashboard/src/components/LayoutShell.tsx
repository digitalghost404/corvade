'use client';

import { useState, useCallback, useEffect } from 'react';
import Header from './Header';
import TabNav from './TabNav';
import KeyboardHelp from './KeyboardHelp';
import CommandPalette from './CommandPalette';
import { useKeyboard } from '@/hooks/useKeyboard';
import TopologyCanvas from './TopologyCanvas';
import CursorGlow from './CursorGlow';

export default function LayoutShell({ children }: { children: React.ReactNode }) {
  const [helpOpen, setHelpOpen] = useState(false);
  const [paletteOpen, setPaletteOpen] = useState(false);

  const bindings = useCallback(
    () => ({
      '?': () => setHelpOpen((prev) => !prev),
      '1': () => { window.location.href = '/'; },
      '2': () => { window.location.href = '/sessions'; },
      '3': () => { window.location.href = '/diff'; },
      '4': () => { window.location.href = '/stats'; },
    }),
    []
  );

  useKeyboard(bindings());

  // Cmd+K / Ctrl+K — separate listener because useKeyboard ignores modifier combos
  useEffect(() => {
    function handlePaletteShortcut(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setPaletteOpen((prev) => !prev);
      }
    }
    document.addEventListener('keydown', handlePaletteShortcut);
    return () => document.removeEventListener('keydown', handlePaletteShortcut);
  }, []);

  return (
    <div className="min-h-screen bg-zinc-950">
      <TopologyCanvas />
      <CursorGlow />

      <Header />

      <div className="pt-12">
        <TabNav />
        <main>{children}</main>
      </div>

      <KeyboardHelp open={helpOpen} onClose={() => setHelpOpen(false)} />

      <CommandPalette
        open={paletteOpen}
        onClose={() => setPaletteOpen(false)}
        onOpenHelp={() => {
          setPaletteOpen(false);
          setHelpOpen(true);
        }}
      />
    </div>
  );
}
