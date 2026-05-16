"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";

interface Event {
  id: string;
  title: string;
  description?: string;
  venue_name?: string;
  start_time: string;
  end_time: string;
  capacity: number;
  min_age: number;
  category?: string;
}

export default function Home() {
  const [events, setEvents] = useState<Event[]>([]);
  const [token, setToken] = useState<string | null>(null);
  const [user, setUser] = useState<any>(null);

  useEffect(() => {
    const t = localStorage.getItem("aev_token");
    if (t) setToken(t);
  }, []);

  useEffect(() => {
    api.listEvents().then(setEvents).catch(console.error);
  }, []);

  useEffect(() => {
    if (token) api.me().then(setUser).catch(() => localStorage.removeItem("aev_token"));
  }, [token]);

  return (
    <main className="max-w-6xl mx-auto px-4 py-8">
      <header className="flex justify-between items-center mb-12">
        <h1 className="text-2xl font-bold tracking-tight">aeV</h1>
        <div className="flex gap-3 items-center">
          {user ? (
            <span className="text-sm text-zinc-400">{user.display_name}</span>
          ) : (
            <a href="/auth" className="text-sm text-purple-400 hover:text-purple-300">Sign In</a>
          )}
          <a href="/tickets" className="text-sm text-purple-400 hover:text-purple-300">My Tickets</a>
        </div>
      </header>

      <section>
        <h2 className="text-3xl font-semibold mb-2">Upcoming Events</h2>
        <p className="text-zinc-500 mb-8">Privacy-verified. ZK-protected. No data shared.</p>

        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {events.map((e) => (
            <a
              key={e.id}
              href={`/events/${e.id}`}
              className="block border border-zinc-800 rounded-xl p-5 hover:border-purple-600 transition"
            >
              <div className="flex justify-between items-start mb-2">
                <h3 className="font-semibold text-lg">{e.title}</h3>
                {e.category && (
                  <span className="text-xs bg-zinc-800 px-2 py-1 rounded">{e.category}</span>
                )}
              </div>
              {e.venue_name && <p className="text-sm text-zinc-400">📍 {e.venue_name}</p>}
              <p className="text-sm text-zinc-500 mt-1">
                {new Date(e.start_time).toLocaleDateString("en-US", {
                  weekday: "short", month: "short", day: "numeric", hour: "2-digit", minute: "2-digit"
                })}
              </p>
              <div className="flex gap-3 mt-3 text-xs text-zinc-500">
                <span>🎫 {e.capacity} spots</span>
                <span>🔞 {e.min_age}+</span>
              </div>
            </a>
          ))}
          {events.length === 0 && (
            <p className="text-zinc-600 col-span-full text-center py-12">No events yet</p>
          )}
        </div>
      </section>
    </main>
  );
}
