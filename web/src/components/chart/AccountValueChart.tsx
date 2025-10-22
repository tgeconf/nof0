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
  ReferenceLine,
} from "recharts";
import { useAccountValueSeries } from "@/lib/api/hooks/useAccountValueSeries";
import { format } from "date-fns";
import { getModelColor, getModelName } from "@/lib/model/meta";
import ErrorBanner from "@/components/ui/ErrorBanner";
import { SkeletonBlock } from "@/components/ui/Skeleton";

type Range = "ALL" | "72H";
type Mode = "$" | "%";

interface ChartDataPoint {
  timestamp: Date;
  [modelId: string]: Date | number;
}

interface SeriesPoint {
  timestamp: number;
  [modelId: string]: number;
}

const MAX_POINTS_72H = 600;

export default function AccountValueChart() {
  const { series, modelIds, isLoading, isError } = useAccountValueSeries();
  const [range, setRange] = useState<Range>("ALL");
  const [mode, setMode] = useState<Mode>("$");
  const cutoffTimeRef = useRef<number>(0);
  const [dataRows, setDataRows] = useState<ChartDataPoint[]>([]);
  const [ids, setIds] = useState<string[]>([]);
  const lastTsRef = useRef<number | null>(null);
  const [active, setActive] = useState<Set<string>>(new Set());

  // Initialize cutoff time on client side to avoid hydration mismatch
  useEffect(() => {
    cutoffTimeRef.current = Date.now() - 72 * 3600 * 1000;
  }, []);

  // Append-only updates to避免整表重建
  useEffect(() => {
    if (!series.length) return;
    const nextIds = Array.from(new Set([...(ids || []), ...modelIds]));
    const sorted = [...series].sort((a, b) => a.timestamp - b.timestamp);
    const nextRows: ChartDataPoint[] = [...dataRows];
    if (lastTsRef.current == null) {
      for (const p of sorted) {
        const row: ChartDataPoint = { timestamp: new Date(p.timestamp) };
        for (const id of nextIds) {
          const value = (p as SeriesPoint)[id];
          if (typeof value === "number") {
            (row as any)[id] = value;
          }
        }
        nextRows.push(row);
      }
      lastTsRef.current = sorted[sorted.length - 1]?.timestamp ?? null;
      setDataRows(nextRows);
      setIds(nextIds);
      setActive((prev) => (prev.size ? prev : new Set(nextIds))); // initialize legend selection
      return;
    }
    for (const p of sorted) {
      if (p.timestamp < (lastTsRef.current as number)) continue;
      const row: ChartDataPoint = { timestamp: new Date(p.timestamp) };
      for (const id of nextIds) {
        const value = (p as SeriesPoint)[id];
        if (typeof value === "number") {
          (row as any)[id] = value;
        }
      }
      if (p.timestamp === (lastTsRef.current as number)) {
        // in-place merge for the last timestamp
        const idx = nextRows.length - 1;
        nextRows[idx] = { ...nextRows[idx], ...row } as any;
      } else {
        nextRows.push(row);
        lastTsRef.current = p.timestamp;
      }
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
    // Downsample for 72H range to keep chart performant
    if (range === "72H" && points.length > MAX_POINTS_72H) {
      const step = Math.ceil(points.length / MAX_POINTS_72H);
      const sampled: ChartDataPoint[] = [];
      for (let i = 0; i < points.length; i += step) sampled.push(points[i]);
      // ensure last point included
      if (sampled[sampled.length - 1] !== points[points.length - 1]) sampled.push(points[points.length - 1]);
      points = sampled;
    }
    return { data: points, models: ids };
  }, [dataRows, ids, range, mode]);

  return (
    <div className="flex h-full flex-col rounded-md border border-white/10 bg-zinc-950 p-3">
      <div className="mb-1 flex items-center justify-between">
        <div className="text-xs font-semibold tracking-wider text-zinc-300">TOTAL ACCOUNT VALUE</div>
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
      <div className="w-full flex-1">
        {isLoading ? (
          <SkeletonBlock className="h-full" />
        ) : data.length >= 2 ? (
          <ResponsiveContainer>
            <LineChart data={data} margin={{ top: 8, right: 80, bottom: 8, left: 0 }}>
              <CartesianGrid stroke="rgba(255,255,255,0.06)" strokeDasharray="3 3" />
              <XAxis
                dataKey="timestamp"
                tickFormatter={(v: Date) => format(v, "MM-dd HH:mm")}
                tick={{ fill: "#a1a1aa", fontSize: 11 }}
              />
              <YAxis tickFormatter={(v: number) => (mode === "%" ? `${v.toFixed(1)}%` : `$${Math.round(v).toLocaleString()}`)} tick={{ fill: "#a1a1aa", fontSize: 11 }} width={60} domain={["auto", "auto"]} />
              <Tooltip
                contentStyle={{ background: "#09090b", border: "1px solid rgba(255,255,255,0.1)", color: "#e4e4e7" }}
                labelFormatter={(v) => (v instanceof Date ? format(v, "yyyy-MM-dd HH:mm") : String(v))}
                formatter={(val: number) => (mode === "%" ? `${Number(val).toFixed(2)}%` : `$${Number(val).toFixed(2)}`)}
              />
              {mode === "$" ? (
                <ReferenceLine y={10000} stroke="#a1a1aa" strokeDasharray="4 4" />
              ) : (
                <ReferenceLine y={0} stroke="#a1a1aa" strokeDasharray="4 4" />
              )}
              <Legend
                wrapperStyle={{ color: "#a1a1aa" }}
                onClick={(e: any) => {
                  const id = e?.dataKey as string | undefined;
                  if (!id) return;
                  setActive((prev) => {
                    const next = new Set(prev);
                    if (next.has(id)) next.delete(id); else next.add(id);
                    return next;
                  });
                }}
              />
              {models.map((id, i) => (
                <Line
                  key={id}
                  type="monotone"
                  dataKey={id}
                  dot={data.length < 50}
                  connectNulls
                  stroke={getModelColor(id)}
                  strokeWidth={1.8}
                  isAnimationActive
                  animationDuration={700}
                  animationEasing="ease-out"
                  name={getModelName(id)}
                  hide={active.size ? !active.has(id) : false}
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
