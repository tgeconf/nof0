"use client";
import { useConversations } from "@/lib/api/hooks/useConversations";
import { useSearchParams, useRouter, usePathname } from "next/navigation";
import { useMemo, useState } from "react";
import { getModelName, getModelColor } from "@/lib/model/meta";

export default function ModelChatPanel() {
  const { items, isLoading, isError } = useConversations();
  const search = useSearchParams();
  const router = useRouter();
  const pathname = usePathname();
  const qModel = (search.get("model") || "ALL").trim();

  // Group conversations by model, sorted by time desc
  const grouped = useMemo(() => {
    const by: Record<string, any[]> = {};
    for (const it of items) {
      const id = (it as any).model_id;
      if (!id) continue;
      const ts = (it as any).timestamp || (it as any).inserted_at || 0;
      const content = (it as any).cot_trace_summary || (it as any).summary || "";
      const user_prompt = (it as any).user_prompt || "";
      const cot_trace = (it as any).cot_trace || {};
      const llm_response = (it as any).llm_response || {};
      (by[id] ||= []).push({ model_id: id, timestamp: ts, content, user_prompt, cot_trace, llm_response });
    }
    for (const k of Object.keys(by)) {
      by[k].sort((a, b) => Number(b.timestamp) - Number(a.timestamp));
    }
    return by;
  }, [items]);

  const list = useMemo(() => {
    const ids = Object.keys(grouped).filter((id) => (qModel === "ALL" ? true : id === qModel));
    ids.sort((a, b) => getModelName(a).localeCompare(getModelName(b)));
    return ids.map((id) => ({ model_id: id, latest: grouped[id][0], history: grouped[id].slice(1) }));
  }, [grouped, qModel]);

  if (isLoading) return <div className="text-xs text-zinc-500">加载模型对话中…</div>;
  if (isError) return <div className="text-xs text-red-400">模型对话接口暂不可用，请稍后重试。</div>;
  if (!list.length) return <div className="text-xs text-zinc-500">暂无模型对话。</div>;

  return (
    <div className="space-y-3">
      <FilterBar model={qModel} onChange={(v) => setModel(v)} models={["ALL", ...Object.keys(grouped)]} />
      {list.map((row) => (
        <ChatCard
          key={row.model_id}
          modelId={row.model_id}
          content={row.latest?.content}
          timestamp={row.latest?.timestamp}
          user_prompt={row.latest?.user_prompt}
          cot_trace={row.latest?.cot_trace}
          llm_response={row.latest?.llm_response}
          history={row.history}
        />
      ))}
    </div>
  );

  function setModel(v: string) {
    const params = new URLSearchParams(search.toString());
    if (!v || v === "ALL") params.delete("model"); else params.set("model", v);
    router.replace(`${pathname}?${params.toString()}`);
  }
}

function fmtTime(t?: number | string) {
  if (!t) return "";
  const n = typeof t === "string" ? Number(t) : t;
  const ms = n > 1e12 ? n : n * 1000;
  const d = new Date(ms);
  const pad = (x: number) => String(x).padStart(2, "0");
  return `${d.getMonth() + 1}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function FilterBar({ model, onChange, models }: { model: string; onChange: (v: string) => void; models: string[] }) {
  const uniq = Array.from(new Set(models));
  return (
    <div className="mb-1 flex items-center gap-2 text-[12px]">
      <span className="font-semibold tracking-wide text-zinc-300">FILTER:</span>
      <select className="rounded border border-white/10 bg-zinc-950 px-2 py-1 text-xs text-zinc-200" value={model} onChange={(e) => onChange(e.target.value)}>
        {uniq.map((m) => (
          <option key={m} value={m}>{m}</option>
        ))}
      </select>
    </div>
  );
}

function ChatCard({ modelId, content, timestamp, user_prompt, cot_trace, llm_response, history }:
  { modelId: string; content?: string; timestamp?: number | string; user_prompt?: string; cot_trace?: any; llm_response?: any; history?: any[] }) {
  const color = getModelColor(modelId);
  const [open, setOpen] = useState(false);
  const [openHist, setOpenHist] = useState<Record<string, boolean>>({});
  return (
    <div className="rounded-md border p-3" style={{ borderColor: `${color}55`, background: `linear-gradient(0deg, ${color}14, transparent)` }}>
      <div className="mb-2 flex items-center justify-between">
        <div className="text-sm font-semibold" style={{ color }}>{getModelName(modelId)}</div>
        <div className="text-[11px] text-zinc-400">{fmtTime(timestamp)}</div>
      </div>
      <div className="relative">
        <div className="whitespace-pre-wrap text-[13px] leading-6 text-zinc-100" style={{ fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace' }}>
          {content || "(no summary)"}
        </div>
        <button className="absolute bottom-0 right-0 translate-y-5 text-[11px] italic text-zinc-400 hover:text-zinc-200" onClick={() => setOpen(!open)}>
          {open ? "click to collapse" : "click to expand"}
        </button>
      </div>
      {open && (
        <div className="mt-3 space-y-3 text-[12px]">
          <Section title="USER_PROMPT">
            <pre className="whitespace-pre-wrap text-zinc-200">{user_prompt || "—"}</pre>
          </Section>
          <Section title="CHAIN_OF_THOUGHT">
            <pre className="whitespace-pre-wrap text-zinc-200">{formatCot(cot_trace)}</pre>
          </Section>
          <Section title="TRADING_DECISIONS">
            {renderDecisions(llm_response)}
          </Section>
        </div>
      )}

      {!!history?.length && (
        <div className="mt-4">
          <div className="mb-1 text-[11px] font-semibold text-zinc-400">历史对话</div>
          <div className="space-y-2">
            {history.slice(0, 5).map((h, idx) => {
              const key = String(h.timestamp || idx);
              const isOpen = !!openHist[key];
              return (
                <div key={key} className="rounded border border-white/10 p-2">
                  <div className="mb-1 flex items-center justify-between text-[11px] text-zinc-400">
                    <span>{fmtTime(h.timestamp)}</span>
                    <button className="text-[11px] italic hover:text-zinc-200" onClick={() => setOpenHist({ ...openHist, [key]: !isOpen })}>
                      {isOpen ? "collapse" : "click to expand"}
                    </button>
                  </div>
                  {isOpen && (
                    <div className="space-y-2">
                      <Section title="USER_PROMPT"><pre className="whitespace-pre-wrap text-zinc-200">{h.user_prompt || "—"}</pre></Section>
                      <Section title="CHAIN_OF_THOUGHT"><pre className="whitespace-pre-wrap text-zinc-200">{formatCot(h.cot_trace)}</pre></Section>
                      <Section title="TRADING_DECISIONS">{renderDecisions(h.llm_response)}</Section>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <div className="mb-1 text-[11px] font-semibold tracking-wide text-zinc-300">{title}</div>
      <div>{children}</div>
    </div>
  );
}

function formatCot(cot: any) {
  try {
    return JSON.stringify(cot ?? {}, null, 2);
  } catch {
    return String(cot ?? "—");
  }
}

function renderDecisions(resp: any) {
  const rows = [] as any[];
  if (resp && typeof resp === "object") {
    for (const [coin, v] of Object.entries(resp as Record<string, any>)) {
      rows.push({
        coin,
        signal: (v as any).signal,
        leverage: (v as any).leverage,
        target: (v as any).profit_target,
        stop: (v as any).stop_loss,
        risk: (v as any).risk_usd,
        invalid: (v as any).invalidation_condition,
        confidence: (v as any).confidence,
      });
    }
  }
  if (!rows.length) return <div className="text-zinc-500">—</div>;
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-left text-[11px]">
        <thead className="text-zinc-400">
          <tr className="border-b border-white/10">
            <th className="py-1.5 pr-3">币种</th>
            <th className="py-1.5 pr-3">动作</th>
            <th className="py-1.5 pr-3">杠杆</th>
            <th className="py-1.5 pr-3">目标价</th>
            <th className="py-1.5 pr-3">止损价</th>
            <th className="py-1.5 pr-3">风险USD</th>
            <th className="py-1.5 pr-3">置信度</th>
            <th className="py-1.5 pr-3">失效条件</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r, i) => (
            <tr key={i} className="border-b border-white/5">
              <td className="py-1.5 pr-3">{r.coin}</td>
              <td className="py-1.5 pr-3">{String(r.signal || "—").toUpperCase()}</td>
              <td className="py-1.5 pr-3">{r.leverage != null ? `${r.leverage}x` : "—"}</td>
              <td className="py-1.5 pr-3">{r.target ?? "—"}</td>
              <td className="py-1.5 pr-3">{r.stop ?? "—"}</td>
              <td className="py-1.5 pr-3">{r.risk ?? "—"}</td>
              <td className="py-1.5 pr-3">{r.confidence != null ? (r.confidence * 100).toFixed(0) + '%' : '—'}</td>
              <td className="py-1.5 pr-3">{r.invalid ?? "—"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
