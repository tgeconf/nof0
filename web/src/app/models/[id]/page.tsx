import ModelDetailsPanel from "@/components/model/ModelDetailsPanel";

export default async function ModelPage(props: { params: Promise<{ id: string }> }) {
  const { id } = await props.params;
  return (
    <main className="min-h-screen w-full p-3">
      <ModelDetailsPanel modelId={decodeURIComponent(id)} />
    </main>
  );
}
