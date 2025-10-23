"use client";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useMemo } from "react";
import { useTheme } from "@/store/useTheme";

const SIDES = ["ALL", "LONG", "SHORT"] as const;

export default function PositionsFilter({ models, symbols }: { models: string[]; symbols: string[] }) {
  const isDark = useTheme((s) => s.resolved) === 'dark';
  const search = useSearchParams();
  const router = useRouter();
  const pathname = usePathname();

  const model = search.get("model") || "ALL";
  const symbol = search.get("symbol") || "ALL";
  const side = search.get("side") || "ALL";

  const modelOptions = useMemo(() => ["ALL", ...models], [models]);
  const symbolOptions = useMemo(() => ["ALL", ...symbols], [symbols]);

  function setQuery(next: Record<string, string>) {
    const params = new URLSearchParams(search.toString());
    for (const [k, v] of Object.entries(next)) {
      if (!v || v === "ALL") params.delete(k);
      else params.set(k, v);
    }
    router.replace(`${pathname}?${params.toString()}`);
  }

  return (
    <div className={`mb-2 flex flex-wrap items-center gap-2 text-[11px] ${isDark?"text-zinc-300":"text-zinc-700"}`}>
      <Select label="模型" value={model} options={modelOptions} onChange={(v) => setQuery({ model: v })} isDark={isDark} />
      <Select label="币种" value={symbol} options={symbolOptions} onChange={(v) => setQuery({ symbol: v })} isDark={isDark} />
      <Select label="方向" value={side} options={SIDES as unknown as string[]} onChange={(v) => setQuery({ side: v })} isDark={isDark} />
    </div>
  );
}

function Select({ label, value, options, onChange, isDark }: { label: string; value: string; options: string[]; onChange: (v: string) => void; isDark: boolean }) {
  return (
    <label className="flex items-center gap-1">
      <span className={isDark?"text-zinc-400":"text-zinc-600"}>{label}</span>
      <select
        className={`rounded border px-2 py-1 text-xs ${isDark?"border-white/10 bg-zinc-950 text-zinc-200":"border-black/10 bg-white text-zinc-800"}`}
        value={value}
        onChange={(e) => onChange(e.target.value)}
      >
        {options.map((opt) => (
          <option key={opt} value={opt}>
            {opt}
          </option>
        ))}
      </select>
    </label>
  );
}
