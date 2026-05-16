"use client";

import { useState } from "react";
import { api } from "@/lib/api";

export default function AuthPage() {
  const [step, setStep] = useState<"world-id" | "oauth" | "siwe">("world-id");
  const [error, setError] = useState("");

  // World ID verification is the primary auth method
  const handleWorldID = async () => {
    try {
      const { IDKitWidget } = await import("@worldcoin/idkit");
      // IDKitWidget would be rendered here
      // For now, redirect to World ID verify endpoint
      window.location.href = "https://id.worldcoin.org";
    } catch (e) {
      setError("World ID widget failed to load");
    }
  };

  const handleOAuth = (provider: string) => {
    const redirect = `${window.location.origin}/auth/callback`;
    const urls: Record<string, string> = {
      google: `https://accounts.google.com/o/oauth2/v2/auth?client_id=${process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID}&redirect_uri=${redirect}&response_type=code&scope=openid%20email%20profile`,
      github: `https://github.com/login/oauth/authorize?client_id=${process.env.NEXT_PUBLIC_GITHUB_CLIENT_ID}&redirect_uri=${redirect}&scope=user:email`,
    };
    window.location.href = urls[provider] || "";
  };

  return (
    <main className="max-w-md mx-auto px-4 py-20">
      <h1 className="text-3xl font-bold mb-2">Sign In</h1>
      <p className="text-zinc-500 mb-8">Verify your identity to access events</p>

      <div className="space-y-4">
        {/* World ID — Primary */}
        <button
          onClick={handleWorldID}
          className="w-full border-2 border-purple-600 rounded-xl p-5 text-left hover:bg-purple-950/30 transition"
        >
          <div className="font-semibold text-lg">🌐 World ID</div>
          <div className="text-sm text-zinc-400">Privacy-preserving proof of personhood (Recommended)</div>
        </button>

        {/* OAuth fallbacks */}
        <button
          onClick={() => handleOAuth("google")}
          className="w-full border border-zinc-700 rounded-xl p-4 text-left hover:bg-zinc-900 transition"
        >
          <div className="font-semibold">Google</div>
          <div className="text-sm text-zinc-500">Sign in with Google</div>
        </button>

        <button
          onClick={() => handleOAuth("github")}
          className="w-full border border-zinc-700 rounded-xl p-4 text-left hover:bg-zinc-900 transition"
        >
          <div className="font-semibold">GitHub</div>
          <div className="text-sm text-zinc-500">Sign in with GitHub</div>
        </button>

        <button
          onClick={() => setStep("siwe")}
          className="w-full border border-zinc-700 rounded-xl p-4 text-left hover:bg-zinc-900 transition"
        >
          <div className="font-semibold">Wallet (SIWE)</div>
          <div className="text-sm text-zinc-500">Sign-In with Ethereum</div>
        </button>
      </div>

      {error && <p className="text-red-400 mt-4 text-sm">{error}</p>}
    </main>
  );
}
