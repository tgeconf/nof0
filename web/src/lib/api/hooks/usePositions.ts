"use client";
import useSWR from "swr";
import { endpoints, fetcher } from "../nof1";

export interface ExitPlan {
  profit_target?: number;
  stop_loss?: number;
  invalidation_condition?: string;
}

export interface RawPositionRow {
  entry_oid: number;
  risk_usd: number;
  confidence: number;
  exit_plan: ExitPlan;
  entry_time: number; // unix seconds
  symbol: string;
  entry_price: number;
  margin: number;
  leverage: number;
  quantity: number; // positive long, negative short
  current_price: number;
  unrealized_pnl: number;
  closed_pnl?: number;
}

export interface PositionsByModel {
  id: string; // model id
  positions: Record<string, RawPositionRow>;
}

type PositionsResponse = { positions: PositionsByModel[] };

export function usePositions(limit = 1000) {
  const { data, error, isLoading } = useSWR<PositionsResponse>(endpoints.positions(limit), fetcher, {
    refreshInterval: 5000,
    dedupingInterval: 2000,
  });

  return {
    positionsByModel: data?.positions ?? [],
    isLoading,
    isError: !!error,
  };
}

