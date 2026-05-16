"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import Link from "next/link";

interface AdminStats {
  total_users: number;
  total_events: number;
  total_tickets: number;
  total_checkins: number;
}

export default function AdminPage() {
  const [stats, setStats] = useState<AdminStats | null>(null);
  const [analytics, setAnalytics] = useState<any>(null);

  useEffect(() => {
    api.getAdminStats().then(setStats).catch(console.error);
    api.getAnalytics().then(setAnalytics).catch(() => {});
  }, []);

  return (
    <main className="max-w-5xl mx-auto px-4 py-8">
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-3xl font-bold">Admin Dashboard</h1>
        <Link
          href="/admin/events/new"
          className="bg-purple-600 px-4 py-2 rounded-xl text-sm font-semibold hover:bg-purple-500 transition"
        >
          Create Event
        </Link>
      </div>

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-10">
        {[
          { label: "Users", value: stats?.total_users },
          { label: "Events", value: stats?.total_events },
          { label: "Tickets", value: stats?.total_tickets },
          { label: "Check-ins", value: stats?.total_checkins },
        ].map((s) => (
          <div key={s.label} className="border border-zinc-800 rounded-xl p-5">
            <p className="text-zinc-500 text-sm">{s.label}</p>
            <p className="text-3xl font-bold mt-1">{s.value ?? "—"}</p>
          </div>
        ))}
      </div>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-4">Event Management</h2>
        <div className="border border-zinc-800 rounded-xl p-8 text-center text-zinc-500">
          <p>Event list and management coming soon</p>
          <Link
            href="/admin/events/new"
            className="text-purple-400 hover:text-purple-300 text-sm mt-2 inline-block"
          >
            Create your first event →
          </Link>
        </div>
      </section>

      {analytics && (
        <section>
          <h2 className="text-xl font-semibold mb-4">Analytics</h2>
          <pre className="border border-zinc-800 rounded-xl p-4 text-sm text-zinc-400 overflow-x-auto">
            {JSON.stringify(analytics, null, 2)}
          </pre>
        </section>
      )}
    </main>
  );
}
