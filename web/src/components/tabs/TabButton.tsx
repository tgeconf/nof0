"use client";
import { useSearchParams, useRouter, usePathname } from "next/navigation";

export default function TabButton({ name, tabKey }: { name: string; tabKey: string }) {
  const search = useSearchParams();
  const router = useRouter();
  const pathname = usePathname();
  const tab = search.get("tab") || "positions";
  const active = tab === tabKey;
  return (
    <button
      onClick={() => {
        const params = new URLSearchParams(search.toString());
        if (tabKey === "positions") params.delete("tab"); else params.set("tab", tabKey);
        router.replace(`${pathname}?${params.toString()}`);
      }}
      className={`rounded border px-2 py-1 ${active ? "border-white/20 bg-white/10 text-zinc-100" : "border-white/10 text-zinc-300 hover:bg-white/5"}`}
    >
      {name}
    </button>
  );
}

