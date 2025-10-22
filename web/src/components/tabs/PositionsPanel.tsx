"use client";
import { useState } from "react";
import { usePositions } from "@/lib/api/hooks/usePositions";
import { fmtUSD } from "@/lib/utils/formatters";
import ExitPlanModal from "@/components/positions/ExitPlanModal";
import clsx from "clsx";

type SortKey = "symbol" | "leverage" | "entry_price" | "current_price" | "unrealized_pnl" | "side";

export function PositionsPanel() {
  const { positionsByModel, isLoading } = usePositions();
  const [modalOpen, setModalOpen] = useState(false);
  const [modalCtx, setModalCtx] = useState<{ modelId: string; symbol: string; } | null>(null);
  const [sortKey, setSortKey] = useState<SortKey>("unrealized_pnl");
  const [sortDir, setSortDir] = useState<"asc" | "desc">("desc");

  if (isLoading) return <div className="text-sm text-zinc-400">加载持仓中…</div>;

  if (!positionsByModel.length) {
    return <div className="text-sm text-zinc-400">暂无持仓。</div>;
  }

  return (
    <div className="space-y-6">
      {positionsByModel.map((m) => {
        const positionsRaw = Object.values(m.positions || {});
        type PositionWithSide = typeof positionsRaw[number] & { side: "LONG" | "SHORT" };
        const positions = (() => {
          const arr: PositionWithSide[] = positionsRaw.map((p) => ({
            ...p,
            side: p.quantity > 0 ? ("LONG" as const) : ("SHORT" as const),
          }));
          const dir = sortDir === "asc" ? 1 : -1;
          return arr.sort((a, b) => {
            const av = sortKey === "side" ? a.side : a[sortKey];
            const bv = sortKey === "side" ? b.side : b[sortKey];
            if (av == null && bv == null) return 0;
            if (av == null) return 1;
            if (bv == null) return -1;
            if (typeof av === "string" && typeof bv === "string") return av.localeCompare(bv) * dir;
            return (Number(av) - Number(bv)) * dir;
          });
        })();
        const totalUnreal = positions.reduce((acc, p) => acc + (p.unrealized_pnl || 0), 0);
        return (
          <div key={m.id} className="rounded-md border border-white/10 bg-zinc-950 p-4">
            <div className="mb-3 flex items-center justify-between">
              <div className="text-sm font-semibold text-purple-300">{m.id}</div>
              <div className="text-xs text-zinc-400">未实现盈亏合计：<span className={totalUnreal >= 0 ? "text-green-400" : "text-red-400"}>{fmtUSD(totalUnreal)}</span></div>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs">
                <thead className="text-zinc-400">
                  <tr className="border-b border-white/10">
                    {[
                      { k: "side", label: "方向" },
                      { k: "symbol", label: "币种" },
                      { k: "leverage", label: "杠杆" },
                      { k: "entry_price", label: "入场价" },
                      { k: "current_price", label: "当前价" },
                      { k: "unrealized_pnl", label: "未实现盈亏" },
                    ].map((c) => (
                      <th key={c.k} className="py-2 pr-4">
                        <button
                          className={clsx("flex items-center gap-1 hover:text-zinc-200", sortKey === (c.k as SortKey) && "text-zinc-200")}
                          onClick={() => {
                            if (sortKey === (c.k as SortKey)) setSortDir(sortDir === "asc" ? "desc" : "asc");
                            setSortKey(c.k as SortKey);
                          }}
                        >
                          {c.label}
                          {sortKey === (c.k as SortKey) ? <span className="text-[10px]">{sortDir === "asc" ? "▲" : "▼"}</span> : null}
                        </button>
                      </th>
                    ))}
                    <th className="py-2 pr-4">退出计划</th>
                  </tr>
                </thead>
                <tbody>
                  {positions.map((p, i) => {
                    const side = p.quantity > 0 ? "LONG" : "SHORT";
                    return (
                      <tr key={i} className="border-b border-white/5">
                        <td className="py-2 pr-4">{side}</td>
                        <td className="py-2 pr-4">{p.symbol}</td>
                        <td className="py-2 pr-4">{p.leverage}x</td>
                        <td className="py-2 pr-4">{fmtUSD(p.entry_price)}</td>
                        <td className="py-2 pr-4">{fmtUSD(p.current_price)}</td>
                        <td className={`py-2 pr-4 ${p.unrealized_pnl >= 0 ? "text-green-400" : "text-red-400"}`}>{fmtUSD(p.unrealized_pnl)}</td>
                        <td className="py-2 pr-4">
                          {p.exit_plan?.profit_target || p.exit_plan?.stop_loss || p.exit_plan?.invalidation_condition ? (
                            <button
                              className="rounded border border-white/10 px-2 py-1 text-[11px] text-zinc-200 hover:bg-white/5"
                              onClick={() => {
                                setModalCtx({ modelId: m.id, symbol: p.symbol });
                                setModalOpen(true);
                              }}
                            >
                              查看
                            </button>
                          ) : (
                            <span className="text-zinc-500">—</span>
                          )}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
            <ExitPlanModal
              open={modalOpen && !!modalCtx}
              onClose={() => setModalOpen(false)}
              modelId={modalCtx?.modelId || m.id}
              symbol={modalCtx?.symbol || ""}
              exitPlan={modalCtx ? m.positions[modalCtx.symbol]?.exit_plan : undefined}
            />
          </div>
        );
      })}
    </div>
  );
}
