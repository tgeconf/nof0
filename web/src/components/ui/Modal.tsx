"use client";
import { ReactNode, useEffect } from "react";

export default function Modal({ open, onClose, title, children }: { open: boolean; onClose: () => void; title?: string; children: ReactNode; }) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    if (open) document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open) return null;
  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center">
      <div className="absolute inset-0 bg-black/70" onClick={onClose} />
      <div className="relative z-[101] w-full max-w-lg rounded-md border border-white/10 bg-zinc-950 p-4 shadow-xl">
        <div className="mb-3 flex items-center justify-between">
          <h3 className="text-sm font-semibold text-zinc-100">{title ?? "详情"}</h3>
          <button className="rounded px-2 py-1 text-xs text-zinc-400 hover:bg-white/5" onClick={onClose}>
            关闭
          </button>
        </div>
        <div className="text-sm text-zinc-200">{children}</div>
      </div>
    </div>
  );
}
