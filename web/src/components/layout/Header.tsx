"use client";

import Link from "next/link";

export function Header() {
  return (
    <header className="sticky top-0 z-50 w-full border-b border-white/10 bg-black/70 backdrop-blur">
      <div className="flex h-[var(--header-h)] w-full items-center justify-between px-3 text-xs text-zinc-200">
        <Link href="/" className="font-semibold tracking-wide text-purple-300">
          nof0
        </Link>
        <nav className="flex items-center gap-6">
          <Link href="/" className="hover:text-white">
            实盘
          </Link>
          <Link href="/leaderboard" className="hover:text-white">
            排行榜
          </Link>
          <Link href="/models" className="hover:text-white">
            模型
          </Link>
        </nav>
      </div>
    </header>
  );
}

export default Header;
