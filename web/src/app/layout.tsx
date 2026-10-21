import type { Metadata } from "next";
import localFont from "next/font/local";
import "./globals.css";

const geistSans = localFont({
  src: "./fonts/GeistVF.woff",
  variable: "--font-geist-sans",
  weight: "100 900",
});
const geistMono = localFont({
  src: "./fonts/GeistMonoVF.woff",
  variable: "--font-geist-mono",
  weight: "100 900",
});

export const metadata: Metadata = {
  title: "VEX — Distributed Vector Exchange",
  description: "A decentralized, masterless vector database in Go with SWIM gossip discovery, consistent hash partitioning, and sub-millisecond HNSW search.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="scroll-smooth">
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased bg-[#fafafa] text-zinc-900 min-h-screen selection:bg-zinc-900 selection:text-white`}
      >
        {children}
      </body>
    </html>
  );
}
