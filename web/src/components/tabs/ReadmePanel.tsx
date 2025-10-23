"use client";
import { useEffect, useState } from "react";
import { useTheme } from "@/store/useTheme";

export default function ReadmePanel() {
  const isDark = useTheme((s) => s.resolved) === 'dark';
  const [text, setText] = useState<string>("");
  const [error, setError] = useState<string>("");

  useEffect(() => {
    let mounted = true;
    fetch("/README.md", { cache: "no-store" })
      .then((r) => r.text())
      .then((t) => mounted && setText(t))
      .catch(() => mounted && setError("无法加载 README.md"));
    return () => {
      mounted = false;
    };
  }, []);

  if (error) return <div className={`text-xs ${isDark?"text-red-400":"text-red-600"}`}>{error}</div>;
  if (!text) return <div className={`text-xs ${isDark?"text-zinc-500":"text-zinc-600"}`}>加载 README.md…</div>;

  return (
    <article className="max-w-none text-sm">
      <pre className={`whitespace-pre-wrap ${isDark?"text-zinc-200":"text-zinc-800"}`}>{text}</pre>
    </article>
  );
}
