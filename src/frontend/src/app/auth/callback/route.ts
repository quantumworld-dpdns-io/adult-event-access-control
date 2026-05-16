import { NextRequest, NextResponse } from "next/server";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export async function GET(req: NextRequest) {
  const code = req.nextUrl.searchParams.get("code");
  const state = req.nextUrl.searchParams.get("state");
  const error = req.nextUrl.searchParams.get("error");

  if (error || !code) {
    return NextResponse.redirect(new URL("/auth?error=" + (error || "no_code"), req.url));
  }

  try {
    const res = await fetch(`${API}/api/auth/oauth`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        provider: state || "google",
        code,
        state,
        redirect_uri: `${req.nextUrl.origin}/auth/callback`,
      }),
    });

    if (!res.ok) {
      return NextResponse.redirect(new URL("/auth?error=exchange_failed", req.url));
    }

    const data = await res.json();

    const response = NextResponse.redirect(new URL("/", req.url));
    response.cookies.set("aev_token", data.token, {
      httpOnly: true,
      sameSite: "lax",
      maxAge: 60 * 60 * 24 * 7,
      path: "/",
      secure: process.env.NODE_ENV === "production",
    });

    return response;
  } catch {
    return NextResponse.redirect(new URL("/auth?error=server_error", req.url));
  }
}
