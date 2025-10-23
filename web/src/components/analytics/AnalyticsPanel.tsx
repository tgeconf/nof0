"use client";
import { useAnalytics } from "@/lib/api/hooks/useAnalytics";
import ErrorBanner from "@/components/ui/ErrorBanner";
import { useTheme } from "@/store/useTheme";

export default function AnalyticsPanel() {
  const isDark = useTheme((s) => s.resolved) === 'dark';
  const { data, isLoading, isError } = useAnalytics();
  const fee = data?.fee_pnl_moves_breakdown_table || [];
  const wl = data?.winners_losers_breakdown_table || [];
  return (
    <div className="space-y-3">
      <ErrorBanner message={isError ? "分析数据暂不可用。" : undefined} />
      <div className="grid grid-cols-2 gap-3 text-[11px]">
        <Kpi label="胜率" value={fmtPct(data?.win_rate)} />
        <Kpi label="多空比" value={data?.long_short_trades_ratio != null ? data?.long_short_trades_ratio.toFixed(2) : "—"} />
        <Kpi label="平均置信度" value={fmtPct(data?.avg_confidence)} />
        <Kpi label="中位置信度" value={fmtPct(data?.median_confidence)} />
      </div>
      <Block title="费用/盈亏分解">
        <MiniTable rows={fee} />
      </Block>
      <Block title="赢家/输家分布">
        <MiniTable rows={wl} />
      </Block>
      {isLoading ? <div className="text-xs text-zinc-500">加载中…</div> : null}
    </div>
  );
}

function Kpi({ label, value }: { label: string; value?: string }) {
  const isDark = useTheme((s) => s.resolved) === 'dark';
  return (
    <div className={`rounded-md border p-3 ${isDark?"border-white/10 bg-zinc-950":"border-black/10 bg-white"}`}>
      <div className={isDark?"text-zinc-400":"text-zinc-600"}>{label}</div>
      <div className={`mt-1 text-sm ${isDark?"text-zinc-100":"text-zinc-800"}`}>{value ?? "—"}</div>
    </div>
  );
}

function Block({ title, children }: { title: string; children: React.ReactNode }) {
  const isDark = useTheme((s) => s.resolved) === 'dark';
  return (
    <div className={`rounded-md border ${isDark?"border-white/10":"border-black/10"}`}>
      <div className={`border-b px-3 py-2 text-xs ${isDark?"border-white/10 text-zinc-400":"border-black/10 text-zinc-600"}`}>{title}</div>
      <div className="p-2">{children}</div>
    </div>
  );
}

function MiniTable({ rows }: { rows: any[] }) {
  const isDark = useTheme((s) => s.resolved) === 'dark';
  if (!rows?.length) return <div className="text-xs text-zinc-500">暂无数据</div>;
  const cols = Object.keys(rows[0] || {});
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-left text-[11px]">
        <thead className={isDark?"text-zinc-400":"text-zinc-600"}>
          <tr className={`border-b ${isDark?"border-white/10":"border-black/10"}`}>
            {cols.map((c) => (
              <th key={c} className="py-1.5 pr-3">{c}</th>
            ))}
          </tr>
        </thead>
        <tbody className={isDark?"text-zinc-200":"text-zinc-800"}>
          {rows.slice(0, 20).map((r, idx) => (
            <tr key={idx} className={`border-b ${isDark?"border-white/5":"border-black/5"} ${!isDark && idx%2===1?"bg-black/3":""}`}>
              {cols.map((c) => (
                <td key={c} className="py-1.5 pr-3">{fmtAny(r[c])}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function fmtPct(n?: number) {
  if (n == null) return "—";
  return (n * 100).toFixed(1) + "%";
}
function fmtAny(v: any) {
  if (v == null) return "—";
  if (typeof v === "number") return String(Math.round(v * 1000) / 1000);
  return String(v);
}
