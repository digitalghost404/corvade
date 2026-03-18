import Timeline from '@/components/Timeline';

export default function Home() {
  return (
    <main className="min-h-screen bg-zinc-950 text-zinc-100 p-6">
      <div className="max-w-7xl mx-auto">
        <div className="flex items-center gap-3 mb-6">
          <h1 className="text-xl font-semibold">Corvade</h1>
          <span className="text-zinc-500 text-sm">AI Agent Control Plane</span>
        </div>
        <Timeline />
      </div>
    </main>
  );
}
