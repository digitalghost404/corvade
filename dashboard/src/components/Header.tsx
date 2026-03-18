'use client';

import { useEffect, useRef, useState } from 'react';
import { getWS } from '@/lib/wsClient';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:4401';

export default function Header() {
  const [traceCount, setTraceCount] = useState<number | null>(null);
  const [cost, setCost] = useState<number | null>(null);
  const [flashing, setFlashing] = useState(false);
  const flashTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    fetch(`${API_BASE}/api/stats`)
      .then((r) => r.json())
      .then((data) => {
        setTraceCount(data.trace_count ?? 0);
        setCost(data.total_cost ?? 0);
      })
      .catch(() => {
        setTraceCount(0);
        setCost(0);
      });

    const handleTraceNew = (data: { cost?: number }) => {
      setTraceCount((n) => (n ?? 0) + 1);
      if (data?.cost != null) {
        setCost((c) => (c ?? 0) + data.cost!);
      }
      setFlashing(true);
      if (flashTimer.current) clearTimeout(flashTimer.current);
      flashTimer.current = setTimeout(() => setFlashing(false), 200);
    };

    const wsClient = getWS();
    wsClient.on('trace:new', handleTraceNew);
    wsClient.connect();

    return () => {
      wsClient.off('trace:new', handleTraceNew);
      wsClient.disconnect();
      if (flashTimer.current) clearTimeout(flashTimer.current);
    };
  }, []);

  const statsText =
    traceCount === null
      ? '— traces · $—'
      : `${traceCount} traces · $${(cost ?? 0).toFixed(4)}`;

  return (
    <header className="fixed top-0 left-0 right-0 z-50 h-12 flex items-center justify-between px-4 bg-zinc-900 border-b border-zinc-800">
      <div className="flex items-center gap-2">
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img src="/logo.svg" alt="Corvade logo" width={24} height={24} />
        <span className="font-semibold text-sm text-zinc-50">Corvade</span>
      </div>
      <div
        className={`text-xs tabular-nums text-zinc-400 stat-flash${flashing ? ' updating' : ''}`}
      >
        {statsText}
      </div>
    </header>
  );
}
