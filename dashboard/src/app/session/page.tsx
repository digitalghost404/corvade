'use client';

import { Suspense } from 'react';
import { useSearchParams } from 'next/navigation';
import TopologyGraph from '@/components/TopologyGraph';

function SessionContent() {
  const searchParams = useSearchParams();
  const id = searchParams.get('id') ?? '';

  return (
    <main className="min-h-screen bg-zinc-950 text-zinc-100 p-6">
      <div className="max-w-7xl mx-auto">
        <div className="flex items-center gap-3 mb-6">
          <a href="/" className="text-zinc-500 hover:text-zinc-300">
            ← Back
          </a>
          <h1 className="text-xl font-semibold">
            Session {id ? id.slice(0, 12) : '—'}
          </h1>
        </div>
        {id ? (
          <TopologyGraph sessionId={id} />
        ) : (
          <div className="flex items-center justify-center h-96 bg-zinc-950 border border-zinc-800 rounded-lg text-zinc-500">
            No session ID provided.
          </div>
        )}
      </div>
    </main>
  );
}

export default function SessionPage() {
  return (
    <Suspense
      fallback={
        <main className="min-h-screen bg-zinc-950 text-zinc-100 p-6">
          <div className="max-w-7xl mx-auto">
            <div className="flex items-center justify-center h-96 text-zinc-500">
              Loading...
            </div>
          </div>
        </main>
      }
    >
      <SessionContent />
    </Suspense>
  );
}
