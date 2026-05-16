const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

async function request(path: string, opts: RequestInit = {}) {
  const token = typeof window !== "undefined" ? localStorage.getItem("aev_token") : null;
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(opts.headers as Record<string, string>),
  };
  if (token) headers["Authorization"] = `Bearer ${token}`;

  const res = await fetch(`${API}${path}`, { ...opts, headers });
  if (!res.ok) throw new Error(`API error: ${res.status}`);
  return res.json();
}

export const api = {
  // Auth
  oauth: (provider: string, code: string, redirect?: string) =>
    request("/api/auth/oauth", { method: "POST", body: JSON.stringify({ provider, code, redirect_uri: redirect }) }),

  siwe: (message: string, signature: string) =>
    request("/api/auth/siwe", { method: "POST", body: JSON.stringify({ message, signature }) }),

  worldID: (nullifierHash: string, proof: string) =>
    request("/api/auth/world-id", { method: "POST", body: JSON.stringify({ nullifier_hash: nullifierHash, proof }) }),

  me: () => request("/api/auth/me"),

  // Events
  listEvents: (category?: string) =>
    request(`/api/events${category ? `?category=${category}` : ""}`),

  getEvent: (id: string) => request(`/api/events/${id}`),

  // Tickets
  myTickets: () => request("/api/my/tickets"),

  issueTicket: (eventId: string, zkProof: string, ticketType = "standard", nullifierHash?: string) =>
    request(`/api/events/${eventId}/tickets`, {
      method: "POST",
      body: JSON.stringify({ zk_proof_commitment: zkProof, nullifier_hash: nullifierHash, ticket_type: ticketType }),
    }),

  // ZK
  generateAgeProof: (birthdate: string, minAge: number) =>
    request("/api/zk/generate-age-proof", {
      method: "POST",
      body: JSON.stringify({ birthdate, min_age: minAge }),
    }),

  // Analytics
  getAnalytics: () => request("/api/analytics/overview"),
  getSybilScore: (userId: string) => request(`/api/analytics/sybil/${userId}`),

  // Admin
  getAdminStats: () => request("/api/admin/stats"),
};
