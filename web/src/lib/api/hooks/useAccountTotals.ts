"use client";
import useSWR from "swr";
import { activityAwareRefresh } from "./activityAware";
import { endpoints, fetcher } from "../nof1";
import type { RawPositionRow } from "./usePositions";

export interface AccountTotalsRow {
  model_id: string;
  timestamp: number; // unix seconds
  equity?: number;
  dollar_equity?: number;
  account_value?: number;
  realized_pnl?: number;
  unrealized_pnl?: number;
  return_pct?: number;
  positions?: Record<string, RawPositionRow>;
}

type AccountTotalsResponse = { accountTotals: AccountTotalsRow[] };

export function useAccountTotals(lastHourlyMarker?: number) {
  const { data, error, isLoading } = useSWR<AccountTotalsResponse>(
    endpoints.accountTotals(lastHourlyMarker),
    fetcher,
    {
      ...activityAwareRefresh(10_000),
    },
  );

  return {
    data,
    isLoading,
    isError: !!error,
  };
}
