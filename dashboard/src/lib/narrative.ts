export interface GraphNode {
  id: string;
  type: string;
  model?: string;
  agent?: string;
  step?: string;
  latency?: number;
  cost?: number;
  confidence?: number;
}

export interface NarrativeSegment {
  nodeId: string;
  text: string;
  nodeType: string;
}

function fmt(value: number | undefined, decimals = 2): string {
  return value !== undefined ? value.toFixed(decimals) : '?';
}

export function generateNarrative(nodes: GraphNode[]): NarrativeSegment[] {
  return nodes.map((node): NarrativeSegment => {
    const agent = node.agent ?? 'Agent';
    const model = node.model ?? 'unknown model';
    const step = node.step ?? node.type;
    const latency = fmt(node.latency);
    const cost = fmt(node.cost, 4);

    let text: string;

    switch (node.type) {
      case 'llm_call':
        text = `${agent} asked ${model} to ${step} in ${latency}s (cost: $${cost})`;
        break;
      case 'tool_call':
        text = `It called ${step} in ${latency}s`;
        break;
      case 'tool_result':
        text = `It received the result from ${step}`;
        break;
      case 'decision_point':
        text = `It decided to ${step}`;
        break;
      default:
        text = `${step} (${node.type}) in ${latency}s`;
    }

    return { nodeId: node.id, text, nodeType: node.type };
  });
}
