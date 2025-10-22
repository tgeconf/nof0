"use client";
import { useEffect, useMemo, useRef, useState } from "react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  Legend,
  ResponsiveContainer,
  CartesianGrid,
} from "recharts";
import { useAccountValueSeries } from "@/lib/api/hooks/useAccountValueSeries";
import { format } from "date-fns";
import ErrorBanner from "@/components/ui/ErrorBanner";
import { SkeletonBlock } from "@/components/ui/Skeleton";

type Range = "ALL" | "72H";
type Mode = "$" | "%";

interface ChartDataPoint {
  timestamp: Date;
  [modelId: string]: Date | number;
}

const palette = [
  "#7c3aed",
  "#3b82f6",
  "#10b981",
  "#ef4444",
  "#f59e0b",
  "#22d3ee",
  "#a3e635",
];

export default function AccountValueChart() {
  const { series, modelIds, isLoading, isError } = useAccountValueSeries();
  const [range, setRange] = useState<Range>("ALL");
  const [mode, setMode] = useState<Mode>("$");
  const cutoffTimeRef = useRef<number>(Date.now() - 72 * 3600 * 1000);
  const [dataRows, setDataRows] = useState<ChartDataPoint[]>([]);
  const [ids, setIds] = useState<string[]>([]);
  const lastTsRef = useRef<number | null>(null);

  // Append-only updates to避免整表重建
  useEffect(() => {
    if (!series.length) return;
    const nextIds = Array.from(new Set([...(ids || []), ...modelIds]));
    const sorted = [...series].sort((a, b) => a.timestamp - b.timestamp);
    const nextRows: ChartDataPoint[] = [...dataRows];
    if (lastTsRef.current == null) {
      for (const p of sorted) {
        const row: ChartDataPoint = { timestamp: new Date(p.timestamp) } as any;
        for (const id of nextIds) if (typeof (p as any)[id] === "number") (row as any)[id] = (p as any)[id];
        nextRows.push(row);
      }
      lastTsRef.current = sorted[sorted.length - 1]?.timestamp ?? null;
      setDataRows(nextRows);
      setIds(nextIds);
      return;
    }
    for (const p of sorted) {
      if (p.timestamp <= (lastTsRef.current as number)) continue;
      const row: ChartDataPoint = { timestamp: new Date(p.timestamp) } as any;
      for (const id of nextIds) if (typeof (p as any)[id] === "number") (row as any)[id] = (p as any)[id];
      nextRows.push(row);
      lastTsRef.current = p.timestamp;
    }
    if (nextRows.length !== dataRows.length) {
      setDataRows(nextRows);
      setIds(nextIds);
    }
  }, [series]);

  const { data, models } = useMemo(() => {
    let points = [...dataRows];
    // Filter by range
    if (range === "72H" && points.length) {
      const cutoff = cutoffTimeRef.current;
      points = points.filter((p) => (p.timestamp as Date).getTime() >= cutoff);
    }
    // Percent mode normalization
    if (mode === "%") {
      const bases: Record<string, number> = {};
      for (const id of ids) {
        for (const p of points) {
          const v = p[id];
          if (typeof v === "number") {
            bases[id] = v;
            break;
          }
        }
      }
      points = points.map((p) => {
        const cp: ChartDataPoint = { ...p } as any;
        for (const id of ids) {
          const base = bases[id];
          const v = p[id];
          if (typeof v === "number" && base) {
            cp[id] = ((v / base) - 1) * 100; // percent
          }
        }
        return cp;
      });
    }
    return { data: points, models: ids };
  }, [dataRows, ids, range, mode]);

  return (
    <div className="rounded-md border border-white/10 bg-zinc-950 p-3">
      <div className="mb-2 flex items-center justify-between">
        <div className="text-sm font-semibold text-zinc-100">账户价值</div>
        <div className="flex items-center gap-2 text-xs">
          <div className="flex overflow-hidden rounded border border-white/10">
            {(["ALL", "72H"] as Range[]).map((r) => (
              <button
                key={r}
                className={`px-2 py-1 ${range === r ? "bg-white/10 text-zinc-100" : "text-zinc-300 hover:bg-white/5"}`}
                onClick={() => setRange(r)}
              >
                {r}
              </button>
            ))}
          </div>
          <div className="flex overflow-hidden rounded border border-white/10">
            {(["$", "%"] as Mode[]).map((m) => (
              <button
                key={m}
                className={`px-2 py-1 ${mode === m ? "bg-white/10 text-zinc-100" : "text-zinc-300 hover:bg-white/5"}`}
                onClick={() => setMode(m)}
              >
                {m}
              </button>
            ))}
          </div>
        </div>
      </div>
      <ErrorBanner message={isError ? "账户价值数据源暂时不可用，请稍后重试。" : undefined} />
      <div className="h-72 w-full">
        {isLoading ? (
          <SkeletonBlock className="h-72" />
        ) : data.length >= 2 ? (
          <ResponsiveContainer>
            <LineChart data={data} margin={{ top: 8, right: 16, bottom: 8, left: 0 }}>
              <CartesianGrid stroke="rgba(255,255,255,0.06)" strokeDasharray="3 3" />
              <XAxis
                dataKey="timestamp"
                tickFormatter={(v: Date) => format(v, "MM-dd HH:mm")}
                tick={{ fill: "#a1a1aa", fontSize: 11 }}
              />
              <YAxis
                tickFormatter={(v: number) => (mode === "%" ? `${v.toFixed(1)}%` : `$${Math.round(v).toLocaleString()}`)}
                tick={{ fill: "#a1a1aa", fontSize: 11 }}
                width={60}
              />
              <Tooltip
                contentStyle={{ background: "#09090b", border: "1px solid rgba(255,255,255,0.1)", color: "#e4e4e7" }}
                labelFormatter={(v) => (v instanceof Date ? format(v, "yyyy-MM-dd HH:mm") : String(v))}
                formatter={(val: number) => (mode === "%" ? `${Number(val).toFixed(2)}%` : `$${Number(val).toFixed(2)}`)}
              />
              <Legend wrapperStyle={{ color: "#a1a1aa" }} />
              {models.map((id, i) => (
                <Line
                  key={id}
                  type="monotone"
                  dataKey={id}
                  dot={data.length < 50}
                  connectNulls
                  stroke={palette[i % palette.length]}
                  strokeWidth={1.8}
                  isAnimationActive
                  animationDuration={700}
                  animationEasing="ease-out"
                />
              ))}
            </LineChart>
          </ResponsiveContainer>
        ) : (
          <div className="flex h-full flex-col items-center justify-center gap-2 text-sm text-zinc-400">
            <div>暂无足够的图表数据（需要至少 2 个时间点）。</div>
            <div className="text-xs text-zinc-500">建议保持页面打开几分钟，我们会按 5 秒节奏累积最新净值点。</div>
            <div className="text-xs text-zinc-500">
              调试：<a className="underline" href="/api/nof1/since-inception-values" target="_blank">since-inception-values</a>
              ，<a className="underline" href="/api/nof1/account-totals" target="_blank">account-totals</a>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
