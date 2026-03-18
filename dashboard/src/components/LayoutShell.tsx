'use client';

import { useState, useCallback } from 'react';
import Header from './Header';
import TabNav from './TabNav';
import KeyboardHelp from './KeyboardHelp';
import { useKeyboard } from '@/hooks/useKeyboard';
import TopologyCanvas from './TopologyCanvas';

export default function LayoutShell({ children }: { children: React.ReactNode }) {
  const [helpOpen, setHelpOpen] = useState(false);

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

  return (
    <div className="min-h-screen bg-zinc-950">
      <TopologyCanvas />
      <Header />
      {/* Push content below the fixed 48px header */}
      <div className="pt-12">
        <TabNav />
        <main className="pt-0">{children}</main>
      </div>
      <KeyboardHelp open={helpOpen} onClose={() => setHelpOpen(false)} />
    </div>
  );
}
