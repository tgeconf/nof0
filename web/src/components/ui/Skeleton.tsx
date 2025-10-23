import { useTheme } from "@/store/useTheme";

export function SkeletonRow({ cols = 6 }: { cols?: number }) {
  const isDark = typeof window !== 'undefined' ? (document.documentElement.dataset.theme ?? 'dark') === 'dark' : true;
  return (
    <tr>
      {Array.from({ length: cols }).map((_, i) => (
        <td key={i} className="py-2 pr-4">
          <div className={`h-3 w-24 animate-pulse rounded ${isDark?"bg-white/10":"bg-black/10"}`} />
        </td>
      ))}
    </tr>
  );
}

export function SkeletonBlock({ className = "h-64" }: { className?: string }) {
  const isDark = typeof window !== 'undefined' ? (document.documentElement.dataset.theme ?? 'dark') === 'dark' : true;
  return <div className={`animate-pulse rounded-md ${isDark?"bg-white/10":"bg-black/10"} ${className}`} />;
}
