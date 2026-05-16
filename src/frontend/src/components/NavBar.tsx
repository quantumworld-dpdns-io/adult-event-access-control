"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";

export default function NavBar() {
  const pathname = usePathname();
  const router = useRouter();
  const [user, setUser] = useState<{ display_name: string } | null>(null);

  useEffect(() => {
    const token = localStorage.getItem("aev_token");
    if (token) {
      fetch("/api/auth/me", {
        headers: { Authorization: `Bearer ${token}` },
      })
        .then((r) => r.json())
        .then((data) => setUser(data))
        .catch(() => {});
    } else {
      setUser(null);
    }
  }, [pathname]);

  const handleSignOut = () => {
    localStorage.removeItem("aev_token");
    setUser(null);
    router.push("/");
    router.refresh();
  };

  return (
    <nav className="border-b border-zinc-800">
      <div className="max-w-6xl mx-auto px-4 h-14 flex items-center justify-between">
        <div className="flex items-center gap-6">
          <Link href="/" className="font-bold tracking-tight">
            aeV
          </Link>
          <Link href="/tickets" className="text-sm text-zinc-400 hover:text-zinc-200">
            Tickets
          </Link>
          <Link href="/checkin" className="text-sm text-zinc-400 hover:text-zinc-200">
            Check-In
          </Link>
          <Link href="/admin" className="text-sm text-zinc-400 hover:text-zinc-200">
            Admin
          </Link>
        </div>
        <div className="flex items-center gap-3">
          {user ? (
            <>
              <span className="text-sm text-zinc-400">{user.display_name}</span>
              <button
                onClick={handleSignOut}
                className="text-sm text-zinc-500 hover:text-zinc-300"
              >
                Sign Out
              </button>
            </>
          ) : (
            <Link href="/auth" className="text-sm text-purple-400 hover:text-purple-300">
              Sign In
            </Link>
          )}
        </div>
      </div>
    </nav>
  );
}
