"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";

interface Ticket {
  id: string;
  event_id: string;
  ticket_type: string;
  status: string;
  qr_secret?: string;
  issued_at: string;
  checked_in_at?: string;
}

export default function TicketsPage() {
  const [tickets, setTickets] = useState<Ticket[]>([]);

  useEffect(() => {
    const token = localStorage.getItem("aev_token");
    if (!token) return;
    api.myTickets().then(setTickets).catch(console.error);
  }, []);

  return (
    <main className="max-w-4xl mx-auto px-4 py-8">
      <h1 className="text-3xl font-bold mb-8">My Tickets</h1>

      <div className="grid gap-4 md:grid-cols-2">
        {tickets.map((t) => (
          <div key={t.id} className="border border-zinc-800 rounded-xl p-5">
            <div className="flex justify-between items-start mb-3">
              <span className={`text-xs px-2 py-1 rounded ${
                t.status === "active" ? "bg-green-900 text-green-300" :
                t.status === "checked_in" ? "bg-blue-900 text-blue-300" : "bg-zinc-800 text-zinc-400"
              }`}>{t.status}</span>
              <span className="text-xs text-zinc-500">{t.ticket_type}</span>
            </div>

            {t.qr_secret && t.status === "active" && (
              <div className="bg-white rounded-lg p-3 flex justify-center my-3">
                <div className="w-32 h-32 bg-black flex items-center justify-center text-xs text-zinc-500">
                  QR: {t.qr_secret.slice(0, 16)}...
                </div>
              </div>
            )}

            <p className="text-xs text-zinc-500">
              Issued: {new Date(t.issued_at).toLocaleDateString()}
            </p>
            {t.checked_in_at && (
              <p className="text-xs text-green-500">
                Checked in: {new Date(t.checked_in_at).toLocaleString()}
              </p>
            )}
          </div>
        ))}
        {tickets.length === 0 && (
          <p className="text-zinc-600 col-span-full text-center py-12">No tickets yet</p>
        )}
      </div>
    </main>
  );
}
