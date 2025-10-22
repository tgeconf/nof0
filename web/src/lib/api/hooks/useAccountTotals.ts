"use client";
import useSWR from "swr";
import { endpoints, fetcher } from "../nof1";

export type AccountTotalsResponse = Record<string, unknown>; // shape varies; keep generic for now

export function useAccountTotals(lastHourlyMarker?: number) {
  const { data, error, isLoading } = useSWR<AccountTotalsResponse>(
    endpoints.accountTotals(lastHourlyMarker),
    fetcher,
    { refreshInterval: 5000 }
  );
  return { data, isLoading, isError: !!error };
}

