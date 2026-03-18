'use client';

import Header from './Header';
import TabNav from './TabNav';

export default function LayoutShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen bg-zinc-950">
      <Header />
      {/* Push content below the fixed 48px header */}
      <div className="pt-12">
        <TabNav />
        <main className="pt-0">{children}</main>
      </div>
    </div>
  );
}
