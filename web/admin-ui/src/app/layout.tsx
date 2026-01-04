import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "CLI Gateway Admin",
  description: "Gateway admin interface"
};

export default function RootLayout({
  children
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="zh-CN">
      <body className="min-h-screen bg-bg text-ink antialiased">{children}</body>
    </html>
  );
}
