import type { Metadata } from "next";
import "./globals.css";
import NavBar from "@/components/NavBar";

export const metadata: Metadata = {
  title: "aeV — Adult Event Access Control",
  description: "Privacy-preserving event access with ZK age proofs",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body className="min-h-screen bg-black text-white antialiased">
        <NavBar />
        {children}
      </body>
    </html>
  );
}
