'use client';

import { useState, useCallback, useEffect, useRef } from 'react';
import Header from './Header';
import TabNav from './TabNav';
import KeyboardHelp from './KeyboardHelp';
import CommandPalette from './CommandPalette';
import { useKeyboard } from '@/hooks/useKeyboard';
import TopologyCanvas from './TopologyCanvas';

// Boot stages:
//  0 → black screen (topology only)
//  1 → header slides down
//  2 → eye-open flash  (class applied via data-boot-eye on the Header wrapper)
//  3 → tab nav fades in
//  4 → content fades in  (sequence complete)

type BootStage = 0 | 1 | 2 | 3 | 4;

const BOOT_KEY = 'corvade-booted';

export default function LayoutShell({ children }: { children: React.ReactNode }) {
  const [helpOpen, setHelpOpen] = useState(false);
  const [paletteOpen, setPaletteOpen] = useState(false);

  // If sessionStorage already has the flag, skip straight to stage 4 (fully visible).
  const [stage, setStage] = useState<BootStage>(() => {
    if (typeof window !== 'undefined' && sessionStorage.getItem(BOOT_KEY)) {
      return 4;
    }
    return 0;
  });

  const timers = useRef<ReturnType<typeof setTimeout>[]>([]);

  useEffect(() => {
    // Already booted this session — nothing to do.
    if (stage === 4) return;

    const schedule = (fn: () => void, delay: number) => {
      const id = setTimeout(fn, delay);
      timers.current.push(id);
    };

    schedule(() => setStage(1), 400);   // header slides down at 400 ms
    schedule(() => setStage(2), 700);   // eye-open flash at 700 ms
    schedule(() => setStage(3), 900);   // tab nav fades in at 900 ms
    schedule(() => {                    // content fades in at 1200 ms, mark done
      setStage(4);
      sessionStorage.setItem(BOOT_KEY, '1');
    }, 1200);

    return () => {
      timers.current.forEach(clearTimeout);
      timers.current = [];
    };
    // Intentionally empty deps — runs once on first mount only.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

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

  // Cmd+K / Ctrl+K — needs a separate listener because useKeyboard explicitly
  // ignores modifier key combos (it bails on ctrlKey/metaKey).
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

  // Derive per-element class names from the current stage.
  // Stage 4 means the animation is done; apply no animation classes so
  // subsequent renders (e.g. hot-reload) don't re-fire them.
  const booted = stage === 4;

  const headerClass = (() => {
    if (booted) return '';
    if (stage === 0) return 'boot-hidden';
    if (stage === 1) return 'boot-header-enter';
    // stages 2-3: header is visible and static
    return '';
  })();

  // The eye-open class is applied to the Header's wrapper for one stage only.
  // Header internally renders the SVG logo; we expose a data attribute so the
  // CSS selector can scope the animation to the iris element.  Because we can't
  // reach inside Header's DOM easily, we wrap it and let the .boot-eye-open
  // class on the wrapper act as a trigger via CSS inheritance / drop-shadow
  // on the <svg> itself.
  const eyeClass = stage === 2 ? 'boot-eye-open' : '';

  const tabClass = (() => {
    if (booted) return '';
    if (stage < 3) return 'boot-hidden';
    if (stage === 3) return 'boot-content-enter';
    return '';
  })();

  const contentClass = (() => {
    if (booted) return '';
    if (stage < 4) return 'boot-hidden';
    return '';
  })();

  return (
    <div className="min-h-screen bg-zinc-950">
      {/* Topology canvas is always visible — it renders behind everything */}
      <TopologyCanvas />

      {/* Header: slides down from above during stage 1, eye flashes at stage 2 */}
      <div className={`${headerClass} ${eyeClass}`.trim()}>
        <Header />
      </div>

      {/* Push content below the fixed 48px header */}
      <div className="pt-12">
        {/* Tab nav: fades in at stage 3 */}
        <div className={tabClass}>
          <TabNav />
        </div>

        {/* Main content: fades in at stage 4 */}
        <main className={`pt-0 ${contentClass}`.trim()}>{children}</main>
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
