"use client";
import { useSearchParams } from "next/navigation";
import ModelDetailsPanel from "@/components/model/ModelDetailsPanel";
import { PositionsPanel } from "@/components/tabs/PositionsPanel";
import TradesTable from "@/components/trades/TradesTable";
import AnalyticsPanel from "@/components/analytics/AnalyticsPanel";
import ReadmePanel from "@/components/tabs/ReadmePanel";

export default function RightTabs() {
  const search = useSearchParams();
  const tab = search.get("tab") || "positions";
  if (tab === "model") return <ModelDetailsPanel />;
  if (tab === "trades") return <TradesTable />;
  if (tab === "analytics") return <AnalyticsPanel />;
  if (tab === "readme") return <ReadmePanel />;
  return <PositionsPanel />;
}
