"use client";
import { useEffect, useState } from "react";

export default function ReadmePanel() {
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

  if (error) return <div className="text-xs text-red-400">{error}</div>;
  if (!text) return <div className="text-xs text-zinc-500">加载 README.md…</div>;

  return (
    <article className="max-w-none text-sm">
      <pre className="whitespace-pre-wrap text-zinc-200">{text}</pre>
    </article>
  );
}
