'use client';

type EventHandler = (data: any) => void;

class CorvadeWS {
  private ws: WebSocket | null = null;
  private handlers: Map<string, Set<EventHandler>> = new Map();
  private reconnectTimer: NodeJS.Timeout | null = null;
  private url: string;

  constructor(url?: string) {
    this.url = url || process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:4401/ws';
  }

  connect() {
    this.ws = new WebSocket(this.url);
    this.ws.onmessage = (e) => {
      const event = JSON.parse(e.data);
      this.handlers.get(event.event)?.forEach(h => h(event.data));
    };
    this.ws.onclose = () => {
      this.reconnectTimer = setTimeout(() => this.connect(), 2000);
    };
  }

  disconnect() {
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    this.ws?.close();
  }

  on(event: string, handler: EventHandler) {
    if (!this.handlers.has(event)) this.handlers.set(event, new Set());
    this.handlers.get(event)!.add(handler);
  }

  off(event: string, handler: EventHandler) {
    this.handlers.get(event)?.delete(handler);
  }
}

export { CorvadeWS };
