"use client";
import { useTheme } from "@/store/useTheme";

export default function ErrorBanner({ message }: { message?: string }) {
  if (!message) return null;
  const isDark = useTheme((s) => s.resolved) === 'dark';
  return (
    <div className={`mb-3 rounded-md border px-3 py-2 text-xs ${isDark?"border-red-500/30 bg-red-500/10 text-red-300":"border-red-500/25 bg-red-500/10 text-red-700"}`}>
      {message}
    </div>
  );
}
