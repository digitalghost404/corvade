'use client';

import { useEffect, useState, useCallback } from 'react';
import {
  ReactFlow,
  Background,
  Controls,
  type Node,
  type Edge,
  Position,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import dagre from 'dagre';
import { fetchSessionGraph } from '@/lib/api';

const NODE_WIDTH = 220;
const NODE_HEIGHT = 80;

interface GraphNode {
  id: string;
  type: string;
  model?: string;
  agent?: string;
  step?: string;
  cost?: number;
  confidence?: number;
}

interface GraphEdge {
  id: string;
  source: string;
  target: string;
  confidence?: number;
}

interface GraphData {
  nodes: GraphNode[];
  edges: GraphEdge[];
}

const NODE_TYPE_STYLES: Record<string, { border: string; rounded: boolean }> = {
  llm_call: { border: '#3b82f6', rounded: true },
  tool_call: { border: '#f59e0b', rounded: true },
  tool_result: { border: '#10b981', rounded: true },
  decision_point: { border: '#ef4444', rounded: false },
};

function getNodeStyle(node: GraphNode) {
  const config = NODE_TYPE_STYLES[node.type] ?? { border: '#71717a', rounded: true };
  const isLowConfidence = (node.confidence ?? 1) < 0.8;
  return {
    background: '#09090b',
    border: `1.5px ${isLowConfidence ? 'dashed' : 'solid'} ${config.border}`,
    borderRadius: config.rounded ? '8px' : '2px',
    color: '#f4f4f5',
    width: NODE_WIDTH,
    height: NODE_HEIGHT,
    fontSize: '12px',
    padding: '8px 12px',
    display: 'flex',
    flexDirection: 'column' as const,
    justifyContent: 'center',
    gap: '2px',
  };
}

function buildFlowNodes(rawNodes: GraphNode[]): Node[] {
  return rawNodes.map((n) => ({
    id: n.id,
    type: 'default',
    position: { x: 0, y: 0 },
    sourcePosition: Position.Bottom,
    targetPosition: Position.Top,
    style: getNodeStyle(n),
    data: {
      label: (
        <div>
          <div className="font-medium text-zinc-100 truncate">{n.step ?? n.type}</div>
          {n.model && (
            <div className="text-zinc-400 text-[11px] truncate">{n.model}</div>
          )}
          {n.cost !== undefined && (
            <div className="text-zinc-500 text-[11px]">
              ${n.cost.toFixed(4)}
            </div>
          )}
        </div>
      ),
    },
  }));
}

function buildFlowEdges(rawEdges: GraphEdge[]): Edge[] {
  return rawEdges.map((e) => {
    const isLowConfidence = (e.confidence ?? 1) < 0.8;
    return {
      id: e.id,
      source: e.source,
      target: e.target,
      style: {
        stroke: isLowConfidence ? '#52525b' : '#3f3f46',
        strokeDasharray: isLowConfidence ? '5 3' : undefined,
        strokeWidth: 1.5,
      },
      animated: false,
    };
  });
}

function applyDagreLayout(nodes: Node[], edges: Edge[]): Node[] {
  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({ rankdir: 'TB', nodesep: 40, ranksep: 60 });

  nodes.forEach((n) => g.setNode(n.id, { width: NODE_WIDTH, height: NODE_HEIGHT }));
  edges.forEach((e) => g.setEdge(e.source, e.target));

  dagre.layout(g);

  return nodes.map((n) => {
    const pos = g.node(n.id);
    return {
      ...n,
      position: {
        x: pos.x - NODE_WIDTH / 2,
        y: pos.y - NODE_HEIGHT / 2,
      },
    };
  });
}

interface Props {
  sessionId: string;
}

export default function TopologyGraph({ sessionId }: Props) {
  const [nodes, setNodes] = useState<Node[]>([]);
  const [edges, setEdges] = useState<Edge[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data: GraphData = await fetchSessionGraph(sessionId);
      const rawNodes: GraphNode[] = data.nodes ?? [];
      const rawEdges: GraphEdge[] = data.edges ?? [];

      const flowEdges = buildFlowEdges(rawEdges);
      const flowNodes = applyDagreLayout(buildFlowNodes(rawNodes), flowEdges);

      setNodes(flowNodes);
      setEdges(flowEdges);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load graph');
    } finally {
      setLoading(false);
    }
  }, [sessionId]);

  useEffect(() => {
    load();
  }, [load]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96 bg-zinc-950 border border-zinc-800 rounded-lg text-zinc-500">
        Loading graph...
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-96 bg-zinc-950 border border-zinc-800 rounded-lg text-red-400">
        {error}
      </div>
    );
  }

  if (nodes.length === 0) {
    return (
      <div className="flex items-center justify-center h-96 bg-zinc-950 border border-zinc-800 rounded-lg text-zinc-500">
        No graph data for this session.
      </div>
    );
  }

  return (
    <div className="h-[600px] bg-zinc-950 border border-zinc-800 rounded-lg overflow-hidden">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        fitView
        fitViewOptions={{ padding: 0.2 }}
        nodesDraggable={false}
        nodesConnectable={false}
        elementsSelectable={false}
        colorMode="dark"
      >
        <Background color="#27272a" gap={24} />
        <Controls
          style={{
            background: '#18181b',
            border: '1px solid #3f3f46',
            borderRadius: '6px',
          }}
        />
      </ReactFlow>
    </div>
  );
}
