'use client';

import { useEffect, useRef, useState } from 'react';
import { getWS } from '@/lib/wsClient';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:4401';

function CorvadeLogo({ noticing }: { noticing: boolean }) {
  return (
    <svg
      width="24"
      height="24"
      viewBox="0 0 80 80"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
      style={{ filter: 'drop-shadow(0 0 4px rgba(139, 92, 246, 0.4))' }}
    >
      {/* Circuit traces: left corner */}
      <path d="M22 40 L10 40 L10 28" stroke="#8b5cf6" strokeWidth="1.5" strokeLinecap="round" fill="none" opacity="0.4" />
      <circle cx="10" cy="26" r="2" fill="#8b5cf6" opacity="0.5" />

      {/* Circuit traces: right corner */}
      <path d="M58 40 L70 40 L70 28" stroke="#8b5cf6" strokeWidth="1.5" strokeLinecap="round" fill="none" opacity="0.4" />
      <circle cx="70" cy="26" r="2" fill="#8b5cf6" opacity="0.5" />

      {/* Circuit traces: upper-left branch */}
      <path d="M33 30 L33 20 L18 20" stroke="#8b5cf6" strokeWidth="1.5" strokeLinecap="round" fill="none" opacity="0.4" />
      <circle cx="16" cy="20" r="2" fill="#8b5cf6" opacity="0.5" />

      {/* Circuit traces: upper-right branch */}
      <path d="M47 30 L47 20 L62 20" stroke="#8b5cf6" strokeWidth="1.5" strokeLinecap="round" fill="none" opacity="0.4" />
      <circle cx="64" cy="20" r="2" fill="#8b5cf6" opacity="0.5" />

      {/* Lower-left trace */}
      <path d="M28 46 L14 46 L14 56" stroke="#8b5cf6" strokeWidth="1.5" strokeLinecap="round" fill="none" opacity="0.4" />
      <circle cx="14" cy="58" r="2" fill="#8b5cf6" opacity="0.5" />

      {/* Lower-right trace */}
      <path d="M52 46 L66 46 L66 56" stroke="#8b5cf6" strokeWidth="1.5" strokeLinecap="round" fill="none" opacity="0.4" />
      <circle cx="66" cy="58" r="2" fill="#8b5cf6" opacity="0.5" />

      {/* Eye outline */}
      <path d="M22 40 C28 28 52 28 58 40 C52 52 28 52 22 40 Z" stroke="#8b5cf6" strokeWidth="2" fill="none" />

      {/* Iris — breathing animation, notice animation overrides on trace:new */}
      <circle
        cx="40"
        cy="40"
        r="9"
        fill="#8b5cf6"
        className={noticing ? 'eye-noticing' : ''}
        style={
          noticing
            ? undefined
            : { animation: 'eye-breathe 4s ease-in-out infinite', transformOrigin: '40px 40px' }
        }
      />

      {/* Pupil: diamond cutout */}
      <path d="M40 33 L46 40 L40 47 L34 40 Z" fill="#09090b" />
    </svg>
  );
}

export default function Header() {
  const [traceCount, setTraceCount] = useState<number | null>(null);
  const [cost, setCost] = useState<number | null>(null);
  const [noticing, setNoticing] = useState(false);
  const noticeTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

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
      setNoticing(true);
      if (noticeTimer.current) clearTimeout(noticeTimer.current);
      noticeTimer.current = setTimeout(() => setNoticing(false), 300);
    };

    const wsClient = getWS();
    wsClient.on('trace:new', handleTraceNew);
    wsClient.connect();

    return () => {
      wsClient.off('trace:new', handleTraceNew);
      wsClient.disconnect();
      if (noticeTimer.current) clearTimeout(noticeTimer.current);
    };
  }, []);

  const statsText =
    traceCount === null
      ? '— traces · $—'
      : `${traceCount} traces · $${(cost ?? 0).toFixed(4)}`;

  const hasTraces = traceCount !== null && traceCount > 0;

  return (
    <header className="fixed top-0 left-0 right-0 z-50 h-12 flex items-center justify-between px-4 bg-zinc-900 border-b border-zinc-800">
      <div className="flex items-center gap-2">
        <CorvadeLogo noticing={noticing} />
        <span className="font-semibold text-sm text-zinc-50">Corvade</span>
      </div>
      <div className="flex items-center gap-2">
        {/* Heartbeat indicator */}
        <span
          className={hasTraces ? 'heartbeat' : ''}
          style={{
            display: 'inline-block',
            width: 8,
            height: 8,
            borderRadius: '50%',
            backgroundColor: hasTraces ? '#22c55e' : '#52525b',
            flexShrink: 0,
          }}
          aria-hidden="true"
        />
        <span className="text-xs tabular-nums text-zinc-400">
          {statsText}
        </span>
      </div>
    </header>
  );
}
