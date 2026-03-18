'use client';

import { generateNarrative, type GraphNode, type NarrativeSegment } from '@/lib/narrative';

const TYPE_ICONS: Record<string, string> = {
  llm_call: '●',
  tool_call: '⬡',
  tool_result: '◆',
  decision_point: '◇',
};

const TYPE_COLORS: Record<string, string> = {
  llm_call: 'text-blue-400',
  tool_call: 'text-amber-400',
  tool_result: 'text-emerald-400',
  decision_point: 'text-red-400',
};

interface Props {
  nodes: GraphNode[];
}

export default function NarrativeView({ nodes }: Props) {
  const segments: NarrativeSegment[] = generateNarrative(nodes);

  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded-lg overflow-hidden">
      <div className="px-4 py-3 border-b border-zinc-800">
        <span className="text-xs font-semibold tracking-widest uppercase text-zinc-400">
          Narrative
        </span>
      </div>

      {segments.length === 0 ? (
        <div className="px-4 py-8 text-center text-zinc-500 text-sm">
          No narrative to display.
        </div>
      ) : (
        <ol className="divide-y divide-zinc-800">
          {segments.map((seg, i) => {
            const icon = TYPE_ICONS[seg.nodeType] ?? '○';
            const color = TYPE_COLORS[seg.nodeType] ?? 'text-zinc-400';

            return (
              <li key={seg.nodeId} className="flex items-start gap-3 px-4 py-3">
                <span className="mt-0.5 text-xs text-zinc-600 font-mono w-5 shrink-0 text-right">
                  {i + 1}
                </span>
                <span className={`mt-0.5 text-sm shrink-0 ${color}`} aria-hidden="true">
                  {icon}
                </span>
                <span className="text-sm text-zinc-300 leading-relaxed">{seg.text}</span>
              </li>
            );
          })}
        </ol>
      )}
    </div>
  );
}
