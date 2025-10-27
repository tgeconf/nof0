"use client";

import type { SWRConfiguration } from "swr";

export interface ActivityAwareConfig {
  refreshIntervalWhenHidden?: number;
  disableActivityTracking?: boolean;
}

export interface AdaptiveRefreshOptions {
  hiddenInterval?: number;
  disableTracking?: boolean;
}

export function activityAwareRefresh(
  interval: number,
  options: AdaptiveRefreshOptions = {},
) {
  const { hiddenInterval = 0, disableTracking = false } = options;

  return {
    refreshInterval: interval,
    refreshIntervalWhenHidden: hiddenInterval,
    disableActivityTracking: disableTracking,
  };
}
