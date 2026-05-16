"use client";

import { use, useEffect, useState, useCallback } from "react";

interface SectionCapacity {
  section: string;
  capacity: number;
  occupied: number;
  checkins: number;
  percentage: number;
}

interface Alert {
  id: string;
  type: string;
  event_id: string;
  message: string;
  severity: string;
  timestamp: string;
}

const severityBadge: Record<string, string> = {
  info: "bg-blue-900/50 text-blue-300",
  warning: "bg-yellow-900/50 text-yellow-300",
  critical: "bg-red-900/50 text-red-300",
};

function SectionCard({ section }: { section: SectionCapacity }) {
  const barColor =
    section.percentage >= 90
      ? "bg-red-500"
      : section.percentage >= 70
        ? "bg-yellow-500"
        : "bg-green-500";

  return (
    <div className="border border-zinc-800 rounded-xl p-4">
      <div className="flex justify-between items-center mb-2">
        <h3 className="font-semibold text-lg">{section.section}</h3>
        <span className="text-sm text-zinc-400">
          {section.occupied}/{section.capacity}
        </span>
      </div>
      <div className="w-full bg-zinc-800 rounded-full h-3 mb-2">
        <div
          className={`${barColor} h-3 rounded-full transition-all duration-500`}
          style={{ width: `${Math.min(section.percentage, 100)}%` }}
        />
      </div>
      <div className="flex justify-between text-xs text-zinc-500">
        <span>{section.checkins} checked in</span>
        <span>{section.percentage.toFixed(1)}%</span>
      </div>
    </div>
  );
}

export default function VenuePage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const [sections, setSections] = useState<SectionCapacity[]>([]);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [connected, setConnected] = useState(false);
  const [alertForm, setAlertForm] = useState({
    type: "general",
    message: "",
    severity: "info",
  });
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    fetch(`/api/events/${id}/heatmap`)
      .then((r) => r.json())
      .then(setSections)
      .catch(console.error);

    const liveSource = new EventSource(`/api/events/${id}/live`);
    liveSource.onopen = () => setConnected(true);
    liveSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.type === "connected") return;
      } catch {
        /* ignore */
      }
    };
    liveSource.onerror = () => setConnected(false);

    const alertSource = new EventSource(`/api/events/${id}/alerts`);
    alertSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data) as Alert;
        if (data.type === "connected") return;
        setAlerts((prev) => [data, ...prev].slice(0, 50));
      } catch {
        /* ignore */
      }
    };

    return () => {
      liveSource.close();
      alertSource.close();
    };
  }, [id]);

  useEffect(() => {
    const interval = setInterval(() => {
      fetch(`/api/events/${id}/heatmap`)
        .then((r) => r.json())
        .then(setSections)
        .catch(() => {});
    }, 30000);
    return () => clearInterval(interval);
  }, [id]);

  const handleCreateAlert = useCallback(async () => {
    if (!alertForm.message) return;
    setSubmitting(true);
    try {
      const token = localStorage.getItem("aev_token");
      const headers: Record<string, string> = {
        "Content-Type": "application/json",
      };
      if (token) headers["Authorization"] = `Bearer ${token}`;

      const res = await fetch(`/api/events/${id}/alerts`, {
        method: "POST",
        headers,
        body: JSON.stringify(alertForm),
      });
      if (res.ok) {
        setAlertForm({ type: "general", message: "", severity: "info" });
      }
    } catch {
      /* ignore */
    } finally {
      setSubmitting(false);
    }
  }, [id, alertForm]);

  return (
    <main className="max-w-6xl mx-auto px-4 py-8">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold">Venue Dashboard</h1>
          <p className="text-zinc-500 text-sm mt-1">Event ID: {id}</p>
        </div>
        <div className="flex items-center gap-2">
          <span
            className={`inline-block w-3 h-3 rounded-full ${connected ? "bg-green-500 animate-pulse" : "bg-red-500"}`}
          />
          <span className="text-sm text-zinc-400">
            {connected ? "Live" : "Disconnected"}
          </span>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Heatmap grid */}
        <div className="lg:col-span-2">
          <h2 className="text-xl font-semibold mb-4">Section Capacity</h2>
          {sections.length === 0 ? (
            <div className="border border-dashed border-zinc-700 rounded-xl p-12 text-center text-zinc-500">
              No venue sections configured for this event.
            </div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              {sections.map((s, i) => (
                <SectionCard key={i} section={s} />
              ))}
            </div>
          )}

          {/* Alert creation */}
          <div className="mt-8 border border-zinc-800 rounded-xl p-6">
            <h3 className="font-semibold mb-4">Create Alert</h3>
            <div className="space-y-4">
              <div className="flex gap-4">
                <select
                  value={alertForm.severity}
                  onChange={(e) =>
                    setAlertForm((f) => ({ ...f, severity: e.target.value }))
                  }
                  className="bg-zinc-900 border border-zinc-700 rounded-xl px-3 py-2 text-sm"
                >
                  <option value="info">Info</option>
                  <option value="warning">Warning</option>
                  <option value="critical">Critical</option>
                </select>
                <select
                  value={alertForm.type}
                  onChange={(e) =>
                    setAlertForm((f) => ({ ...f, type: e.target.value }))
                  }
                  className="bg-zinc-900 border border-zinc-700 rounded-xl px-3 py-2 text-sm"
                >
                  <option value="general">General</option>
                  <option value="capacity">Capacity</option>
                  <option value="security">Security</option>
                  <option value="technical">Technical</option>
                </select>
              </div>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={alertForm.message}
                  onChange={(e) =>
                    setAlertForm((f) => ({ ...f, message: e.target.value }))
                  }
                  placeholder="Alert message..."
                  className="flex-1 bg-zinc-900 border border-zinc-700 rounded-xl px-4 py-2 text-sm"
                />
                <button
                  onClick={handleCreateAlert}
                  disabled={submitting || !alertForm.message}
                  className="bg-purple-600 rounded-xl px-6 py-2 text-sm font-semibold hover:bg-purple-500 disabled:opacity-50 transition"
                >
                  {submitting ? "..." : "Send"}
                </button>
              </div>
            </div>
          </div>
        </div>

        {/* Alert feed */}
        <div>
          <h2 className="text-xl font-semibold mb-4">Alerts</h2>
          <div className="border border-zinc-800 rounded-xl p-4 max-h-[600px] overflow-y-auto space-y-2">
            {alerts.length === 0 ? (
              <p className="text-zinc-500 text-sm text-center py-8">
                No alerts yet.
              </p>
            ) : (
              alerts.map((a) => (
                <div
                  key={a.id}
                  className={`rounded-xl p-3 text-sm ${severityBadge[a.severity] || severityBadge.info}`}
                >
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-xs font-semibold uppercase">
                      {a.type}
                    </span>
                    <span className="text-xs opacity-60">
                      {new Date(a.timestamp).toLocaleTimeString()}
                    </span>
                  </div>
                  <p>{a.message}</p>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </main>
  );
}
