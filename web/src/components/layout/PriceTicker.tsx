"use client";
import { useMemo } from "react";
import { useCryptoPrices } from "@/lib/api/hooks/useCryptoPrices";
import { fmtUSD } from "@/lib/utils/formatters";

const ORDER = ["BTC", "ETH", "SOL", "BNB", "DOGE", "XRP"] as const;

export default function PriceTicker() {
  const { prices } = useCryptoPrices();
  const list = useMemo(() => {
    const vals = Object.values(prices);
    return ORDER.map((s) => vals.find((v) => v.symbol === s)).filter(Boolean) as typeof vals;
  }, [prices]);

  return (
    <div className="w-full border-b border-white/10 bg-zinc-950">
      <div className="flex flex-wrap gap-x-6 gap-y-1 px-3 py-1.5 text-[11px] leading-5 text-zinc-200">
        {list.map((p) => (
          <span key={p.symbol} className="tabular-nums text-zinc-300">
            <b className="mr-1 text-zinc-100">{p.symbol}</b>
            {fmtUSD(p.price)}
          </span>
        ))}
      </div>
    </div>
  );
}
