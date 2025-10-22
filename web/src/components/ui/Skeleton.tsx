export function SkeletonRow({ cols = 6 }: { cols?: number }) {
  return (
    <tr>
      {Array.from({ length: cols }).map((_, i) => (
        <td key={i} className="py-2 pr-4">
          <div className="h-3 w-24 animate-pulse rounded bg-white/10" />
        </td>
      ))}
    </tr>
  );
}

export function SkeletonBlock({ className = "h-64" }: { className?: string }) {
  return <div className={`animate-pulse rounded-md bg-white/10 ${className}`} />;
}

