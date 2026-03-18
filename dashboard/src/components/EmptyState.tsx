'use client';

import { useState } from 'react';

interface EmptyStateProps {
  title: string;
  description: string;
  code?: string;
}

function WiredEyeIcon() {
  return (
    <svg
      width="48"
      height="48"
      viewBox="0 0 48 48"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
    >
      {/* Dashed connection lines radiating outward */}
      <line
        x1="24" y1="8" x2="24" y2="2"
        stroke="#3f3f46"
        strokeWidth="1.5"
        strokeDasharray="2 2"
        strokeLinecap="round"
      />
      <line
        x1="24" y1="40" x2="24" y2="46"
        stroke="#3f3f46"
        strokeWidth="1.5"
        strokeDasharray="2 2"
        strokeLinecap="round"
      />
      <line
        x1="8" y1="24" x2="2" y2="24"
        stroke="#3f3f46"
        strokeWidth="1.5"
        strokeDasharray="2 2"
        strokeLinecap="round"
      />
      <line
        x1="40" y1="24" x2="46" y2="24"
        stroke="#3f3f46"
        strokeWidth="1.5"
        strokeDasharray="2 2"
        strokeLinecap="round"
      />
      {/* Diagonal dashed lines */}
      <line
        x1="13" y1="13" x2="8" y2="8"
        stroke="#3f3f46"
        strokeWidth="1.5"
        strokeDasharray="2 2"
        strokeLinecap="round"
      />
      <line
        x1="35" y1="13" x2="40" y2="8"
        stroke="#3f3f46"
        strokeWidth="1.5"
        strokeDasharray="2 2"
        strokeLinecap="round"
      />
      <line
        x1="13" y1="35" x2="8" y2="40"
        stroke="#3f3f46"
        strokeWidth="1.5"
        strokeDasharray="2 2"
        strokeLinecap="round"
      />
      <line
        x1="35" y1="35" x2="40" y2="40"
        stroke="#3f3f46"
        strokeWidth="1.5"
        strokeDasharray="2 2"
        strokeLinecap="round"
      />
      {/* Eye outline (almond shape) */}
      <path
        d="M8 24 C12 16, 20 12, 24 12 C28 12, 36 16, 40 24 C36 32, 28 36, 24 36 C20 36, 12 32, 8 24 Z"
        stroke="#52525b"
        strokeWidth="1.5"
        fill="none"
      />
      {/* Iris */}
      <circle
        cx="24"
        cy="24"
        r="6"
        stroke="#52525b"
        strokeWidth="1.5"
        fill="none"
        style={{ animation: 'eye-blink 6s ease-in-out infinite' }}
      />
      {/* Pupil */}
      <circle
        cx="24"
        cy="24"
        r="2.5"
        fill="#52525b"
        style={{ animation: 'eye-blink 6s ease-in-out infinite' }}
      />
    </svg>
  );
}

export default function EmptyState({ title, description, code }: EmptyStateProps) {
  const [copied, setCopied] = useState(false);

  function handleCopy() {
    if (!code) return;
    navigator.clipboard.writeText(code).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }

  return (
    <div className="flex flex-col items-center py-16">
      <WiredEyeIcon />
      <h2 className="text-zinc-200 text-lg font-semibold mt-6">{title}</h2>
      <p className="text-zinc-500 text-sm mt-2 max-w-md text-center">{description}</p>
      {code && (
        <div className="relative mt-4">
          <pre className="bg-zinc-800 text-zinc-300 px-4 py-2 rounded font-mono text-sm">
            {code}
          </pre>
          <button
            type="button"
            onClick={handleCopy}
            className="absolute top-1.5 right-2 text-xs text-zinc-400 hover:text-zinc-200 transition-colors cursor-pointer"
          >
            {copied ? 'Copied!' : 'Copy'}
          </button>
        </div>
      )}
    </div>
  );
}
