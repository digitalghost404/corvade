'use client';

import { useState } from 'react';
import SessionDiff from '@/components/SessionDiff';

export default function DiffPage() {
  const [inputA, setInputA] = useState('');
  const [inputB, setInputB] = useState('');
  const [sessionA, setSessionA] = useState('');
  const [sessionB, setSessionB] = useState('');

  function handleCompare() {
    const a = inputA.trim();
    const b = inputB.trim();
    if (a && b) {
      setSessionA(a);
      setSessionB(b);
    }
  }

  return (
    <main className="min-h-screen bg-zinc-950 text-zinc-100 p-8">
      <div className="max-w-4xl mx-auto space-y-8">
        <div>
          <h1 className="text-2xl font-bold text-zinc-100">Session Diff</h1>
          <p className="text-zinc-500 text-sm mt-1">
            Compare two session graphs to identify structural differences.
          </p>
        </div>

        {/* Inputs */}
        <div className="flex flex-col sm:flex-row gap-3 items-end">
          <div className="flex-1">
            <label className="block text-xs text-zinc-400 mb-1">Session A</label>
            <input
              type="text"
              value={inputA}
              onChange={(e) => setInputA(e.target.value)}
              placeholder="Session ID…"
              className="w-full bg-zinc-900 border border-zinc-700 rounded-md px-3 py-2 text-sm text-zinc-100 placeholder-zinc-600 focus:outline-none focus:ring-1 focus:ring-zinc-500"
            />
          </div>
          <div className="flex-1">
            <label className="block text-xs text-zinc-400 mb-1">Session B</label>
            <input
              type="text"
              value={inputB}
              onChange={(e) => setInputB(e.target.value)}
              placeholder="Session ID…"
              className="w-full bg-zinc-900 border border-zinc-700 rounded-md px-3 py-2 text-sm text-zinc-100 placeholder-zinc-600 focus:outline-none focus:ring-1 focus:ring-zinc-500"
            />
          </div>
          <button
            onClick={handleCompare}
            disabled={!inputA.trim() || !inputB.trim()}
            className="px-5 py-2 rounded-md bg-zinc-100 text-zinc-900 text-sm font-semibold hover:bg-white transition disabled:opacity-40 disabled:cursor-not-allowed"
          >
            Compare
          </button>
        </div>

        {/* Result */}
        {sessionA && sessionB && (
          <SessionDiff sessionA={sessionA} sessionB={sessionB} />
        )}
      </div>
    </main>
  );
}
