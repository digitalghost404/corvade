'use client';

import { useCallback, useRef } from 'react';

export function useTraceSound() {
  const ctxRef = useRef<AudioContext | null>(null);
  const enabledRef = useRef(false);

  const setEnabled = useCallback((on: boolean) => {
    enabledRef.current = on;
    if (on && !ctxRef.current) {
      ctxRef.current = new AudioContext();
    }
  }, []);

  const tick = useCallback(() => {
    if (!enabledRef.current || !ctxRef.current) return;
    const ctx = ctxRef.current;
    const osc = ctx.createOscillator();
    const gain = ctx.createGain();
    osc.connect(gain);
    gain.connect(ctx.destination);
    osc.frequency.setValueAtTime(800, ctx.currentTime);
    osc.frequency.exponentialRampToValueAtTime(400, ctx.currentTime + 0.05);
    gain.gain.setValueAtTime(0.03, ctx.currentTime); // Very quiet
    gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.08);
    osc.start(ctx.currentTime);
    osc.stop(ctx.currentTime + 0.08);
  }, []);

  return { tick, setEnabled, enabled: enabledRef };
}
