const TIME_RANGES = [
  { label: '30d', days: 30 },
  { label: '90d', days: 90 },
  { label: '6m', days: 180 },
  { label: '1y', days: 365 },
  { label: 'All', days: 0 },
] as const

interface TimeRangeSelectorProps {
  selected: number
  onChange: (days: number) => void
}

export function TimeRangeSelector({ selected, onChange }: TimeRangeSelectorProps) {
  return (
    <div
      role="radiogroup"
      aria-label="Select time range"
      className="inline-flex rounded-lg bg-gray-100 dark:bg-gray-800 p-1"
    >
      {TIME_RANGES.map(({ label, days }) => {
        const isActive = selected === days
        return (
          <button
            key={label}
            type="button"
            role="radio"
            aria-checked={isActive}
            onClick={() => onChange(days)}
            className={`px-3 py-2 text-sm rounded-md transition-colors ${
              isActive
                ? 'bg-blue-600 text-white font-semibold shadow-sm'
                : 'bg-transparent text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200 font-normal'
            }`}
          >
            {label}
          </button>
        )
      })}
    </div>
  )
}
