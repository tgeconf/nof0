import PriceTicker from "@/components/layout/PriceTicker";
import { PositionsPanel } from "@/components/tabs/PositionsPanel";

export default function Home() {
  return (
    <main className="mx-auto min-h-screen max-w-7xl px-4">
      <PriceTicker />
      <section className="py-6">
        <h2 className="mb-3 text-lg font-semibold tracking-wide text-zinc-100">
          持仓
        </h2>
        <PositionsPanel />
      </section>
    </main>
  );
}
