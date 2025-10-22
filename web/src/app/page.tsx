import PriceTicker from "@/components/layout/PriceTicker";
import AccountValueChart from "@/components/chart/AccountValueChart";
import { PositionsPanel } from "@/components/tabs/PositionsPanel";

export default function Home() {
  return (
    <main className="min-h-screen w-full terminal-scan">
      <PriceTicker />
      <section className="grid grid-cols-1 gap-3 p-3 lg:grid-cols-3 lg:gap-3 lg:p-3">
        <div className="lg:col-span-2">
          <AccountValueChart />
        </div>
        <div className="lg:col-span-1">
          <h2 className="mb-2 text-sm font-semibold tracking-wide text-zinc-100">持仓</h2>
          <PositionsPanel />
        </div>
      </section>
    </main>
  );
}
