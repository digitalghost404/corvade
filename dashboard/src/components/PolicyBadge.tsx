interface Props {
  mode: 'observe' | 'enforce';
}

export default function PolicyBadge({ mode }: Props) {
  const fill = mode === 'enforce' ? '#ef4444' : '#8b5cf6';

  return (
    <svg
      width="14"
      height="14"
      viewBox="0 0 14 14"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-label={`Policy mode: ${mode}`}
      role="img"
    >
      <path
        d="M7 1L2 3.2V7.4C2 10.12 4.16 12.68 7 13.4C9.84 12.68 12 10.12 12 7.4V3.2L7 1Z"
        fill={fill}
      />
    </svg>
  );
}
