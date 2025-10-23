"use client";
import { useMemo } from "react";
import { useSearchParams } from "next/navigation";
import { useAccountTotals } from "@/lib/api/hooks/useAccountTotals";
import { usePositions } from "@/lib/api/hooks/usePositions";
import { useTrades } from "@/lib/api/hooks/useTrades";
import { fmtUSD, pnlClass } from "@/lib/utils/formatters";
import { getModelName } from "@/lib/model/meta";

export default function ModelDetailsPanel({ modelId: propModelId }: { modelId?: string }) {
  const search = useSearchParams();
  const urlModel = search.get("model") || undefined;
  const modelId = (propModelId || urlModel || "").trim();

  const { data: totalsData } = useAccountTotals();
  const { positionsByModel } = usePositions();
  const { trades } = useTrades();

  const latest = useMemo(() => {
    const list: any[] = (totalsData && (totalsData as any).accountTotals) ? (totalsData as any).accountTotals : [];
    // Find latest entry for modelId
    let row: any | undefined;
    for (let i = list.length - 1; i >= 0; i--) {
      const r = list[i];
      if (r?.model_id === modelId || r?.id === modelId) { row = r; break; }
    }
    if (!row && list.length) row = list[list.length - 1];
    return row;
  }, [totalsData, modelId]);

  const positions = useMemo(() => {
    const found = positionsByModel.find((m) => m.id === modelId);
    return found ? Object.values(found.positions || {}) : [];
  }, [positionsByModel, modelId]);

  const recentTrades = useMemo(() => trades.filter((t) => t.model_id === modelId).slice(-5).reverse(), [trades, modelId]);

  if (!modelId) return <div className="text-xs text-zinc-500">请选择模型（右上筛选的“模型”）。</div>;

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div className="text-sm font-semibold text-zinc-100">{getModelName(modelId)}</div>
        <div className="text-xs text-zinc-400">模型ID：{modelId}</div>
      </div>

      <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-[11px] text-zinc-300">
        <div>净值：<span className="tabular-nums">{fmtUSD(latest?.dollar_equity ?? latest?.equity ?? latest?.account_value)}</span></div>
        <div>累计收益：<span className={pnlClass(latest?.cum_pnl_pct)}>{latest?.cum_pnl_pct != null ? `${latest.cum_pnl_pct.toFixed(2)}%` : '—'}</span></div>
        <div>已实现盈亏：<span className={pnlClass(latest?.realized_pnl)}>{fmtUSD(latest?.realized_pnl)}</span></div>
        <div>未实现盈亏：<span className={pnlClass(latest?.total_unrealized_pnl)}>{fmtUSD(latest?.total_unrealized_pnl)}</span></div>
      </div>

      <div className="rounded-md border border-white/10">
        <div className="border-b border-white/10 px-3 py-2 text-xs text-zinc-400">当前持仓</div>
        <div className="max-h-64 overflow-auto">
          <table className="w-full text-left text-[11px]">
            <thead className="text-zinc-400">
              <tr className="border-b border-white/10">
                <th className="py-1.5 pr-3">方向</th>
                <th className="py-1.5 pr-3">币种</th>
                <th className="py-1.5 pr-3">杠杆</th>
                <th className="py-1.5 pr-3">入场价</th>
                <th className="py-1.5 pr-3">当前价</th>
                <th className="py-1.5 pr-3">未实现盈亏</th>
              </tr>
            </thead>
            <tbody>
              {positions.length ? positions.map((p: any, i: number) => {
                const side = p.quantity > 0 ? 'LONG' : 'SHORT';
                return (
                  <tr key={i} className="border-b border-white/5">
                    <td className="py-1.5 pr-3">{side}</td>
                    <td className="py-1.5 pr-3">{p.symbol}</td>
                    <td className="py-1.5 pr-3">{p.leverage}x</td>
                    <td className="py-1.5 pr-3 tabular-nums">{fmtUSD(p.entry_price)}</td>
                    <td className="py-1.5 pr-3 tabular-nums">{fmtUSD(p.current_price)}</td>
                    <td className={`py-1.5 pr-3 tabular-nums ${pnlClass(p.unrealized_pnl)}`}>{fmtUSD(p.unrealized_pnl)}</td>
                  </tr>
                );
              }) : (
                <tr><td className="p-3 text-xs text-zinc-500" colSpan={6}>暂无持仓</td></tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      <div className="rounded-md border border-white/10">
        <div className="border-b border-white/10 px-3 py-2 text-xs text-zinc-400">最近成交</div>
        <div className="max-h-48 overflow-auto">
          <table className="w-full text-left text-[11px]">
            <thead className="text-zinc-400">
              <tr className="border-b border-white/10">
                <th className="py-1.5 pr-3">币种</th>
                <th className="py-1.5 pr-3">方向</th>
                <th className="py-1.5 pr-3">杠杆</th>
                <th className="py-1.5 pr-3">净盈亏</th>
              </tr>
            </thead>
            <tbody>
              {recentTrades.length ? recentTrades.map((t: any) => (
                <tr key={t.id} className="border-b border-white/5">
                  <td className="py-1.5 pr-3">{t.symbol}</td>
                  <td className="py-1.5 pr-3">{t.side?.toUpperCase()}</td>
                  <td className="py-1.5 pr-3">{t.leverage}x</td>
                  <td className={`py-1.5 pr-3 tabular-nums ${pnlClass(t.realized_net_pnl)}`}>{fmtUSD(t.realized_net_pnl)}</td>
                </tr>
              )) : (
                <tr><td className="p-3 text-xs text-zinc-500" colSpan={4}>暂无成交</td></tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
