'use client';

import { useEffect, useState } from 'react';
import { fetchSessions } from '@/lib/api';
import SessionCard from '@/components/SessionCard';
import EmptyState from '@/components/EmptyState';

interface Session {
  id: string;
  agent: string | null;
  trace_count: number;
  total_tokens: number;
  total_cost: number;
  status: 'active' | 'completed' | 'error';
  start_time: string;
  end_time: string | null;
}

export default function SessionsPage() {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchSessions()
      .then((data) => setSessions(Array.isArray(data) ? data : []))
      .catch(() => setSessions([]))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="p-6 max-w-7xl mx-auto">
      {loading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          {[0, 1, 2].map((i) => (
            <div key={i} className="skeleton h-36 rounded-lg" />
          ))}
        </div>
      ) : sessions.length === 0 ? (
        <EmptyState
          title="No sessions yet"
          description="Sessions are created when agents tag requests with X-Corvade-Session headers."
        />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          {sessions.map((session) => (
            <SessionCard key={session.id} session={session} />
          ))}
        </div>
      )}
    </div>
  );
}
