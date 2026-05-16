"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

export default function NewEventPage() {
  const router = useRouter();
  const [form, setForm] = useState({
    title: "",
    description: "",
    venue_name: "",
    start_time: "",
    end_time: "",
    capacity: 100,
    min_age: 18,
    category: "",
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      const token = localStorage.getItem("aev_token");
      const headers: Record<string, string> = { "Content-Type": "application/json" };
      if (token) headers["Authorization"] = `Bearer ${token}`;

      const res = await fetch("/api/events", {
        method: "POST",
        headers,
        body: JSON.stringify({
          ...form,
          capacity: Number(form.capacity),
          min_age: Number(form.min_age),
          start_time: new Date(form.start_time).toISOString(),
          end_time: new Date(form.end_time).toISOString(),
        }),
      });
      if (!res.ok) {
        const data = await res.json();
        throw new Error(data.error || "Failed to create event");
      }
      router.push("/admin");
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  };

  const handleChange = (key: string) => (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
    setForm((f) => ({ ...f, [key]: e.target.value }));

  return (
    <main className="max-w-2xl mx-auto px-4 py-8">
      <h1 className="text-3xl font-bold mb-8">Create Event</h1>

      <form onSubmit={handleSubmit} className="space-y-5">
        <div>
          <label className="text-sm text-zinc-400 block mb-1">Title</label>
          <input
            type="text"
            value={form.title}
            onChange={handleChange("title")}
            required
            className="w-full bg-zinc-900 border border-zinc-700 rounded-xl px-4 py-3"
          />
        </div>

        <div>
          <label className="text-sm text-zinc-400 block mb-1">Description</label>
          <textarea
            value={form.description}
            onChange={handleChange("description")}
            rows={3}
            className="w-full bg-zinc-900 border border-zinc-700 rounded-xl px-4 py-3"
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="text-sm text-zinc-400 block mb-1">Venue</label>
            <input
              type="text"
              value={form.venue_name}
              onChange={handleChange("venue_name")}
              className="w-full bg-zinc-900 border border-zinc-700 rounded-xl px-4 py-3"
            />
          </div>
          <div>
            <label className="text-sm text-zinc-400 block mb-1">Category</label>
            <input
              type="text"
              value={form.category}
              onChange={handleChange("category")}
              className="w-full bg-zinc-900 border border-zinc-700 rounded-xl px-4 py-3"
            />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="text-sm text-zinc-400 block mb-1">Start Date/Time</label>
            <input
              type="datetime-local"
              value={form.start_time}
              onChange={handleChange("start_time")}
              required
              className="w-full bg-zinc-900 border border-zinc-700 rounded-xl px-4 py-3"
            />
          </div>
          <div>
            <label className="text-sm text-zinc-400 block mb-1">End Date/Time</label>
            <input
              type="datetime-local"
              value={form.end_time}
              onChange={handleChange("end_time")}
              required
              className="w-full bg-zinc-900 border border-zinc-700 rounded-xl px-4 py-3"
            />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="text-sm text-zinc-400 block mb-1">Capacity</label>
            <input
              type="number"
              value={form.capacity}
              onChange={(e) => setForm((f) => ({ ...f, capacity: +e.target.value }))}
              min={1}
              className="w-full bg-zinc-900 border border-zinc-700 rounded-xl px-4 py-3"
            />
          </div>
          <div>
            <label className="text-sm text-zinc-400 block mb-1">Min Age</label>
            <input
              type="number"
              value={form.min_age}
              onChange={(e) => setForm((f) => ({ ...f, min_age: +e.target.value }))}
              min={0}
              className="w-full bg-zinc-900 border border-zinc-700 rounded-xl px-4 py-3"
            />
          </div>
        </div>

        {error && <p className="text-red-400 text-sm">{error}</p>}

        <button
          type="submit"
          disabled={loading}
          className="w-full bg-purple-600 rounded-xl py-3 font-semibold hover:bg-purple-500 disabled:opacity-50 transition"
        >
          {loading ? "Creating..." : "Create Event"}
        </button>
      </form>
    </main>
  );
}
