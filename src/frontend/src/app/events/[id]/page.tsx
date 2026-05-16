"use client";

import { use, useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useRouter } from "next/navigation";

interface Event {
  id: string;
  title: string;
  description?: string;
  venue_name?: string;
  start_time: string;
  end_time: string;
  capacity: number;
  min_age: number;
  status: string;
  category?: string;
}

export default function EventPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const [event, setEvent] = useState<Event | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [ticket, setTicket] = useState<any>(null);
  const [ageProof, setAgeProof] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    api.getEvent(id).then(setEvent).catch(console.error);
    setToken(localStorage.getItem("aev_token"));
  }, [id]);

  const handleGetTicket = async () => {
    setLoading(true);
    try {
      // 1. Generate ZK age proof
      const proof = await api.generateAgeProof("1990-01-01", event?.min_age || 18);
      setAgeProof(proof.proof);

      // 2. Issue ticket
      const t = await api.issueTicket(id, proof.proof);
      setTicket(t);
    } catch (e: any) {
      alert(e.message);
    } finally {
      setLoading(false);
    }
  };

  if (!event) return <div className="p-8 text-zinc-500">Loading...</div>;

  return (
    <main className="max-w-3xl mx-auto px-4 py-8">
      <button onClick={() => router.back()} className="text-sm text-zinc-500 mb-6">← Back</button>

      <div className="border border-zinc-800 rounded-2xl p-8">
        <h1 className="text-4xl font-bold mb-2">{event.title}</h1>
        {event.category && (
          <span className="inline-block text-xs bg-zinc-800 px-2 py-1 rounded mb-4">{event.category}</span>
        )}

        <div className="grid grid-cols-2 gap-4 my-6 text-sm">
          <div>
            <p className="text-zinc-500">Venue</p>
            <p>{event.venue_name || "TBD"}</p>
          </div>
          <div>
            <p className="text-zinc-500">Date</p>
            <p>{new Date(event.start_time).toLocaleString()}</p>
          </div>
          <div>
            <p className="text-zinc-500">Capacity</p>
            <p>{event.capacity}</p>
          </div>
          <div>
            <p className="text-zinc-500">Min Age</p>
            <p>{event.min_age}+</p>
          </div>
        </div>

        {event.description && <p className="text-zinc-300 mb-6">{event.description}</p>}

        {!token ? (
          <a href="/auth" className="block text-center bg-purple-600 rounded-xl py-3 font-semibold hover:bg-purple-500 transition">
            Sign In to Get Ticket
          </a>
        ) : ticket ? (
          <div className="bg-zinc-900 rounded-xl p-4 border border-green-800">
            <p className="text-green-400 font-semibold">✓ Ticket Issued</p>
            <p className="text-xs text-zinc-500 mt-1">QR: {ticket.qr_secret?.slice(0, 8)}...</p>
          </div>
        ) : (
          <button
            onClick={handleGetTicket}
            disabled={loading}
            className="w-full bg-purple-600 rounded-xl py-3 font-semibold hover:bg-purple-500 disabled:opacity-50 transition"
          >
            {loading ? "Generating ZK Proof..." : "Get Ticket"}
          </button>
        )}
      </div>
    </main>
  );
}
