'use client';

import { useEffect } from 'react';

const FOCUS_BLOCKED_TAGS = new Set(['INPUT', 'TEXTAREA', 'SELECT']);

export function useKeyboard(
  bindings: Record<string, (e: KeyboardEvent) => void>
): void {
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      // Ignore when focus is inside a form element
      const tag = document.activeElement?.tagName ?? '';
      if (FOCUS_BLOCKED_TAGS.has(tag)) return;

      // Ignore modified key presses
      if (e.ctrlKey || e.altKey || e.metaKey) return;

      const handler = bindings[e.key];
      if (handler) handler(e);
    }

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
    // Stringify bindings keys for a stable dep — handlers are recreated each
    // render but we only re-register when the key set changes. Callers should
    // memoize handlers if they care about re-registration frequency.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [JSON.stringify(Object.keys(bindings).sort()), bindings]);
}
