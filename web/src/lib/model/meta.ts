export type ModelMeta = {
  id: string;
  name: string;
  color: string;
  icon?: string; // future use
};

const DEFAULT_COLOR = "#a1a1aa";

const METAS: Record<string, ModelMeta> = {
  "gpt-5": { id: "gpt-5", name: "GPT‑5", color: "#7c3aed", icon: "/logos/GPT_logo.png" },
  "claude-sonnet-4-5": { id: "claude-sonnet-4-5", name: "Claude Sonnet 4.5", color: "#22d3ee", icon: "/logos/Claude_logo.png" },
  "deepseek-chat-v3.1": { id: "deepseek-chat-v3.1", name: "DeepSeek v3.1", color: "#10b981", icon: "/logos/deepseek_logo.png" },
  "gemini-2-5-pro": { id: "gemini-2-5-pro", name: "Gemini 2.5 Pro", color: "#3b82f6", icon: "/logos/Gemini_logo.webp" },
  "grok-4": { id: "grok-4", name: "Grok 4", color: "#f59e0b", icon: "/logos/Grok_logo.webp" },
  "qwen3-max": { id: "qwen3-max", name: "Qwen3 Max", color: "#ef4444", icon: "/logos/qwen_logo.png" },
  "buynhold_btc": { id: "buynhold_btc", name: "Buy&Hold BTC", color: "#a3e635" },
};

export function getModelMeta(id: string): ModelMeta {
  return METAS[id] ?? { id, name: id, color: DEFAULT_COLOR };
}

export function getModelColor(id: string): string {
  return getModelMeta(id).color;
}

export function getModelName(id: string): string {
  return getModelMeta(id).name;
}

export function getModelIcon(id: string): string | undefined {
  return getModelMeta(id).icon;
}
