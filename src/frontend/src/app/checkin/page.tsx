"use client";

import { useState } from "react";

export default function CheckInPage() {
  const [qrSecret, setQrSecret] = useState("");
  const [result, setResult] = useState<{ status: string; ticket_id?: string } | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleCheckIn = async () => {
    setLoading(true);
    setError("");
    setResult(null);
    try {
      const token = localStorage.getItem("aev_token");
      const headers: Record<string, string> = { "Content-Type": "application/json" };
      if (token) headers["Authorization"] = `Bearer ${token}`;

      const res = await fetch("/api/checkin", {
        method: "POST",
        headers,
        body: JSON.stringify({ qr_secret: qrSecret, method: "manual" }),
      });
      const data = await res.json();
      if (!res.ok) {
        setError(data.error || "Check-in failed");
      } else {
        setResult(data);
      }
    } catch {
      setError("Network error");
    } finally {
      setLoading(false);
    }
  };

  return (
    <main className="max-w-md mx-auto px-4 py-8">
      <h1 className="text-3xl font-bold mb-2">Venue Check-In</h1>
      <p className="text-zinc-500 mb-8">Scan or enter QR code to verify entry</p>

      <div className="border-2 border-dashed border-zinc-700 rounded-xl p-12 mb-6 flex flex-col items-center justify-center">
        <div className="w-48 h-48 bg-zinc-900 rounded-lg flex items-center justify-center text-zinc-600 text-sm mb-4">
          Camera Preview
        </div>
        <p className="text-xs text-zinc-600">Camera scanner placeholder</p>
      </div>

      <div className="space-y-4">
        <input
          type="text"
          value={qrSecret}
          onChange={(e) => setQrSecret(e.target.value)}
          placeholder="Enter QR secret manually"
          className="w-full bg-zinc-900 border border-zinc-700 rounded-xl px-4 py-3 text-white placeholder-zinc-500"
        />
        <button
          onClick={handleCheckIn}
          disabled={loading || !qrSecret}
          className="w-full bg-purple-600 rounded-xl py-3 font-semibold hover:bg-purple-500 disabled:opacity-50 transition"
        >
          {loading ? "Verifying..." : "Check In"}
        </button>
      </div>

      {result && (
        <div className="mt-6 bg-green-900/30 border border-green-700 rounded-xl p-4">
          <p className="text-green-400 font-semibold">✓ Checked In</p>
          <p className="text-xs text-zinc-400 mt-1">Ticket: {result.ticket_id}</p>
        </div>
      )}

      {error && (
        <div className="mt-6 bg-red-900/30 border border-red-700 rounded-xl p-4">
          <p className="text-red-400">✗ {error}</p>
        </div>
      )}
    </main>
  );
}
