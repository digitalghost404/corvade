const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:4401';

export async function fetchTraces(params?: Record<string, string>) {
  const url = new URL(`${API_BASE}/api/traces`);
  if (params) Object.entries(params).forEach(([k, v]) => url.searchParams.set(k, v));
  const res = await fetch(url.toString());
  return res.json();
}

export async function fetchTrace(id: string) {
  const res = await fetch(`${API_BASE}/api/traces/${id}`);
  return res.json();
}

export async function fetchSessions(params?: Record<string, string>) {
  const url = new URL(`${API_BASE}/api/sessions`);
  if (params) Object.entries(params).forEach(([k, v]) => url.searchParams.set(k, v));
  const res = await fetch(url.toString());
  return res.json();
}

export async function fetchSession(id: string) {
  const res = await fetch(`${API_BASE}/api/sessions/${id}`);
  return res.json();
}

export async function fetchSessionGraph(id: string) {
  const res = await fetch(`${API_BASE}/api/sessions/${id}/graph`);
  return res.json();
}

export async function fetchSessionDiff(id1: string, id2: string) {
  const res = await fetch(`${API_BASE}/api/sessions/${id1}/diff/${id2}`);
  return res.json();
}

export async function fetchStats(params?: Record<string, string>) {
  const url = new URL(`${API_BASE}/api/stats`);
  if (params) Object.entries(params).forEach(([k, v]) => url.searchParams.set(k, v));
  const res = await fetch(url.toString());
  return res.json();
}
