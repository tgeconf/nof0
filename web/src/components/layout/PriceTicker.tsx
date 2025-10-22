"use client";
import { useCryptoPrices } from "@/lib/api/hooks/useCryptoPrices";
import { fmtUSD } from "@/lib/utils/formatters";

export default function PriceTicker() {
  const { prices } = useCryptoPrices();
  const entries = Object.values(prices);

  return (
    <div className="w-full overflow-hidden border-b border-white/10 bg-zinc-950">
      <div
        className="flex animate-[ticker_30s_linear_infinite] gap-8 whitespace-nowrap py-2 text-xs text-zinc-200"
        style={{
          // Duplicate the content to create an infinite marquee feeling
          ['--items' as string]: entries.length || 1,
        } as React.CSSProperties}
      >
        {[...entries, ...entries].map((p, idx) => (
          <span key={idx} className="tabular-nums text-zinc-300">
            <b className="text-zinc-100">{p.symbol}</b> {fmtUSD(p.price)}
          </span>
        ))}
      </div>
      <style>{`
        @keyframes ticker { from { transform: translateX(0); } to { transform: translateX(-50%); } }
      `}</style>
    </div>
  );
}
