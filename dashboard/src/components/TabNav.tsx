'use client';

import { usePathname } from 'next/navigation';

const TABS = [
  { label: 'Traces', href: '/' },
  { label: 'Sessions', href: '/sessions' },
  { label: 'Diff', href: '/diff' },
  { label: 'Stats', href: '/stats' },
];

export default function TabNav() {
  const pathname = usePathname();

  return (
    <nav className="w-full flex flex-row overflow-x-auto bg-zinc-950/80 border-b border-zinc-800" style={{ backdropFilter: 'blur(8px)', WebkitBackdropFilter: 'blur(8px)' }}>
      {TABS.map(({ label, href }) => {
        const isActive = pathname === href;
        return (
          <a
            key={href}
            href={href}
            className={
              `px-4 py-2 text-sm font-medium border-b-2 whitespace-nowrap ` +
              (isActive
                ? 'text-zinc-50 border-violet-500'
                : 'text-zinc-500 hover:text-zinc-300 border-transparent')
            }
          >
            {label}
          </a>
        );
      })}
    </nav>
  );
}
