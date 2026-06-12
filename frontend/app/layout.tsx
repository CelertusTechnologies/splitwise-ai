import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Nivra",
  description: "Shared money, calmly settled."
};

export default function RootLayout({
  children
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body>{children}</body>
    </html>
  );
}

