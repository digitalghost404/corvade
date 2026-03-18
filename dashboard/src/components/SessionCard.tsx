'use client';

import AgentAvatar from '@/components/AgentAvatar';

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

interface Props {
  session: Session;
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  const hh = d.getHours().toString().padStart(2, '0');
  const mm = d.getMinutes().toString().padStart(2, '0');
  return `${hh}:${mm}`;
}

const STATUS_STYLES: Record<Session['status'], string> = {
  active: 'bg-violet-500/20 text-violet-400',
  completed: 'bg-green-500/20 text-green-400',
  error: 'bg-red-500/20 text-red-400',
};

export default function SessionCard({ session }: Props) {
  const timeRange = session.end_time
    ? `${formatTime(session.start_time)} → ${formatTime(session.end_time)}`
    : `${formatTime(session.start_time)} → ongoing`;

  return (
    <a
      href={`/session?id=${session.id}`}
      className="block glass neon-edge rounded-lg p-4 hover:border-violet-500 transition-colors duration-100"
    >
      <div className="flex items-start justify-between gap-2 mb-2">
        <span
          className={`flex items-center gap-2 min-w-0 ${
            session.agent
              ? 'text-lg font-semibold text-zinc-50'
              : 'text-lg font-semibold text-zinc-500 italic'
          }`}
        >
          {session.agent && <AgentAvatar name={session.agent} size={20} />}
          <span className="truncate">{session.agent ?? 'Unknown Agent'}</span>
        </span>
        <span
          className={`shrink-0 text-xs px-2 py-0.5 rounded-full ${STATUS_STYLES[session.status] ?? STATUS_STYLES.error}`}
        >
          {session.status}
        </span>
      </div>

      <p className="text-sm text-zinc-400 mb-3">
        {session.trace_count} traces &middot; {session.total_tokens.toLocaleString()} tok &middot; ${session.total_cost.toFixed(4)}
      </p>

      <p className="text-xs text-zinc-500">{timeRange}</p>
    </a>
  );
}
