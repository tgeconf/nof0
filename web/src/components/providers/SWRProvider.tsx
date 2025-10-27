"use client";
import { SWRConfig, type Middleware } from "swr";
import type { PropsWithChildren } from "react";
import { useMemo } from "react";
import { usePageActivity } from "@/store/usePageActivity";
import type { ActivityAwareConfig } from "@/lib/api/hooks/activityAware";

const activityAwarePolling: Middleware = (useSWRNext) => {
  return (key, fetcher, config) => {
    const { isActive } = usePageActivity();
    const extended = (config ?? {}) as typeof config & ActivityAwareConfig;

    if (
      !extended ||
      typeof extended.refreshInterval !== "number" ||
      extended.refreshInterval <= 0 ||
      extended.disableActivityTracking
    ) {
      return useSWRNext(key, fetcher, config);
    }

    const baseInterval = extended.refreshInterval;
    const hiddenInterval =
      extended.refreshIntervalWhenHidden != null
        ? extended.refreshIntervalWhenHidden
        : 0;

    const appliedInterval = isActive ? baseInterval : hiddenInterval;
    const scopedConfig =
      config && config.refreshInterval === appliedInterval
        ? config
        : ({
            ...(config ?? {}),
            refreshInterval: appliedInterval,
          } as typeof config);

    return useSWRNext(key, fetcher, scopedConfig);
  };
};

export default function SWRProvider({ children }: PropsWithChildren) {
  const middlewares = useMemo(() => [activityAwarePolling], []);

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
        refreshWhenHidden: false,
        focusThrottleInterval: 2_000,
        use: middlewares,
      }}
    >
      {children}
    </SWRConfig>
  );
}
