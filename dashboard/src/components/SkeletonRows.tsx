interface SkeletonRowsProps {
  rows?: number;
}

const cellWidths = ['w-16', 'w-24', 'w-20', 'w-12', 'w-16', 'w-12', 'w-8'] as const;

export default function SkeletonRows({ rows = 5 }: SkeletonRowsProps) {
  return (
    <>
      {Array.from({ length: rows }, (_, rowIndex) => (
        <tr key={rowIndex}>
          {cellWidths.map((width, colIndex) => (
            <td key={colIndex} className="px-3 py-2">
              <div className={`skeleton h-4 ${width}`} />
            </td>
          ))}
        </tr>
      ))}
    </>
  );
}
