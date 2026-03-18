import PolicyBadge from '@/components/PolicyBadge';

interface Violation {
  rule: string;
  type: string;
  mode: string;
  message: string;
}

interface Props {
  violations: Violation[];
  blocked: boolean;
}

export default function PolicyTab({ violations, blocked }: Props) {
  return (
    <div className="p-4 space-y-3">
      {blocked && (
        <div className="bg-red-500/10 border border-red-500/30 text-red-400 rounded-lg px-4 py-2 text-sm">
          This request was blocked by policy enforcement
        </div>
      )}

      {violations.length === 0 ? (
        <p className="text-zinc-500 text-sm">No policy violations</p>
      ) : (
        <div className="space-y-2">
          {violations.map((v, i) => (
            <div
              key={i}
              className="flex items-start gap-3 py-2 border-b border-zinc-800 last:border-0"
            >
              <div className="mt-0.5 shrink-0">
                <PolicyBadge mode={v.mode === 'enforce' ? 'enforce' : 'observe'} />
              </div>
              <div className="min-w-0">
                <div className="flex items-center gap-2 flex-wrap">
                  <span className="text-zinc-200 text-sm font-medium">{v.rule}</span>
                  <span className="text-zinc-500 text-xs">{v.type}</span>
                </div>
                <p className="text-zinc-400 text-xs mt-0.5">{v.message}</p>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
