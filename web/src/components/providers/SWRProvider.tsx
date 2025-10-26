"use client";
import { SWRConfig } from "swr";
import type { PropsWithChildren } from "react";

export default function SWRProvider({ children }: PropsWithChildren) {
  return (
    <SWRConfig
      value={{
        // Ensure views refresh immediately when users return to the tab.
        revalidateOnFocus: true,
        revalidateOnReconnect: true,
        // If cache is stale, revalidate on access to avoid “stuck” data.
        revalidateIfStale: true,
        // Dedup short bursts but keep well below hook intervals (10s).
        dedupingInterval: 2_000,
        // Don’t aggressively retry; our data is mostly periodic.
        shouldRetryOnError: false,
      }}
    >
      {children}
    </SWRConfig>
  );
}
