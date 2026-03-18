'use client';
import Timeline from '@/components/Timeline';
import StatsBar from '@/components/StatsBar';

export default function Home() {
  return (
    <div className="p-6 max-w-7xl mx-auto space-y-4">
      <StatsBar />
      <Timeline />
    </div>
  );
}
