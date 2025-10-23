"use client";
import { useEffect, useMemo, useRef, useState } from "react";
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid, ReferenceLine } from "recharts";
import { useAccountValueSeries } from "@/lib/api/hooks/useAccountValueSeries";
import { format } from "date-fns";
import { getModelColor, getModelName, getModelIcon } from "@/lib/model/meta";
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
  const [vw, setVw] = useState<number>(0);

  // Initialize cutoff time on client side to avoid hydration mismatch
  useEffect(() => {
    cutoffTimeRef.current = Date.now() - 72 * 3600 * 1000;
  }, []);

  // Track viewport width for responsive sizing (logo size, button sizing)
  useEffect(() => {
    const upd = () => setVw(typeof window !== "undefined" ? window.innerWidth : 0);
    upd();
    window.addEventListener("resize", upd);
    return () => window.removeEventListener("resize", upd);
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

  // compute last index per model to render only the end dot as logo
  const lastIdxById = useMemo(() => {
    const m: Record<string, number> = {};
    for (const id of models) {
      for (let i = data.length - 1; i >= 0; i--) {
        const v = (data[i] as any)[id];
        if (typeof v === "number") { m[id] = i; break; }
      }
    }
    return m;
  }, [models, data]);

  const renderEndDot = (id: string) => (p: any) => {
    const { cx, cy, index } = p || {};
    if (cx == null || cy == null) return null;
    if (typeof lastIdxById[id] !== "number" || index !== lastIdxById[id]) return null;
    if (active.size && !active.has(id)) return null;
    const icon = getModelIcon(id);
    const color = getModelColor(id);
    const size = vw < 640 ? 30 : vw < 1024 ? 26 : 22; // bigger, more prominent
    return (
      <g key={`${id}-dot-${index}`} transform={`translate(${cx}, ${cy})`} pointerEvents="none">
        <circle r={Math.round(size * 0.9)} className="animate-ping" fill={color} opacity={0.2} />
        {icon ? (
          <image href={icon} x={-size / 2} y={-size / 2} width={size} height={size} style={{ filter: "drop-shadow(0 0 2px rgba(0,0,0,0.6))" }} />
        ) : (
          <circle r={Math.max(6, Math.round(size * 0.42))} fill={color} />
        )}
      </g>
    );
  };

  return (
    <div className="flex h-full flex-col rounded-md border border-white/10 bg-zinc-950 p-3">
      <div className="mb-2 flex items-center justify-between">
        <div className="text-xs font-semibold tracking-wider text-zinc-300">账户总资产</div>
      </div>
      {/* Legend below chart */}
      <ErrorBanner message={isError ? "账户价值数据源暂时不可用，请稍后重试。" : undefined} />
      <div className="w-full flex-1">
        {isLoading ? (
          <SkeletonBlock className="h-full" />
        ) : data.length >= 2 ? (
          <>
          <ResponsiveContainer>
            <LineChart data={data} margin={{ top: 8, right: 110, bottom: 8, left: 0 }}>
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
              {models.map((id, i) => (
                <Line
                  key={id}
                  type="monotone"
                  dataKey={id}
                  dot={renderEndDot(id)}
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
          {/* Bottom legend inside chart area flow */}
          {models.length > 0 && (
            <div className="mt-3">
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 md:grid-cols-4 xl:grid-cols-5">
                {models.map((id) => {
                  const activeOn = active.size ? active.has(id) : true;
                  const icon = getModelIcon(id);
                  return (
                    <button
                      key={id}
                      className={`w-full group inline-flex items-center justify-start gap-2 rounded border px-3 py-2 text-[13px] sm:text-[14px] transition-colors ${
                        activeOn
                          ? "border-white/25 bg-white/5 text-zinc-100 hover:bg-white/10"
                          : "border-white/10 text-zinc-400 hover:bg-white/5"
                      }`}
                      onClick={() => {
                        setActive((prev) => {
                          if (prev.size === 1 && prev.has(id)) return new Set(models);
                          return new Set([id]);
                        });
                      }}
                    >
                      {icon ? (
                        // eslint-disable-next-line @next/next/no-img-element
                        <img src={icon} alt="" className="h-5 w-5 sm:h-6 sm:w-6 rounded-sm object-contain opacity-95" />
                      ) : (
                        <span
                          className="inline-block h-3.5 w-3.5 sm:h-4 sm:w-4 rounded-full"
                          style={{ background: getModelColor(id) }}
                        />
                      )}
                      <span className="truncate">{getModelName(id)}</span>
                    </button>
                  );
                })}
              </div>
              <div className="mt-2 flex flex-wrap items-center gap-2">
                <div className="flex overflow-hidden rounded border border-white/15">
                  {(["ALL", "72H"] as Range[]).map((r) => (
                    <button
                      key={r}
                      className={`px-3 py-2 sm:px-3.5 sm:py-2 ${range === r ? "bg-white/10 text-zinc-100" : "text-zinc-300 hover:bg-white/5"}`}
                      onClick={() => setRange(r)}
                    >
                      {r}
                    </button>
                  ))}
                </div>
                <div className="flex overflow-hidden rounded border border-white/15">
                  {(["$", "%"] as Mode[]).map((m) => (
                    <button
                      key={m}
                      className={`px-3 py-2 sm:px-3.5 sm:py-2 ${mode === m ? "bg-white/10 text-zinc-100" : "text-zinc-300 hover:bg-white/5"}`}
                      onClick={() => setMode(m)}
                    >
                      {m}
                    </button>
                  ))}
                </div>
              </div>
            </div>
          )}
          </>
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
