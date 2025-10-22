"use client";
export default function ErrorBanner({ message }: { message?: string }) {
  if (!message) return null;
  return (
    <div className="mb-3 rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-300">
      {message}
    </div>
  );
}

