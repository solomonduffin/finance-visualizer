interface AllocationRowProps {
  name: string
  percentage: string
  onChange: (value: string) => void
}

export function AllocationRow({ name, percentage, onChange }: AllocationRowProps) {
  const numericPct = Math.min(parseFloat(percentage) || 0, 100)

  return (
    <div className="flex items-center gap-3 py-2">
      <span className="text-sm text-gray-700 dark:text-gray-300 truncate flex-1">
        {name}
      </span>
      <div className="flex items-center gap-1">
        <input
          type="text"
          inputMode="decimal"
          placeholder="0"
          value={percentage}
          onChange={(e) => onChange(e.target.value)}
          className="w-16 px-2 py-1.5 border border-gray-300 dark:border-gray-600 rounded-md text-sm text-right text-gray-900 dark:text-gray-100 bg-white dark:bg-gray-800 focus:outline-none focus:ring-2 focus:ring-blue-500"
          aria-label={`Allocation percentage for ${name}`}
        />
        <span className="text-xs text-gray-500 dark:text-gray-400">%</span>
      </div>
      <div className="h-1.5 flex-1 bg-gray-200 dark:bg-gray-700 rounded-full">
        <div
          className="h-1.5 bg-blue-500 dark:bg-blue-400 rounded-full transition-all duration-200"
          style={{ width: `${numericPct}%` }}
        />
      </div>
    </div>
  )
}
