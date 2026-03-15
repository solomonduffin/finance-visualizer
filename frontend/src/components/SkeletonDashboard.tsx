export function SkeletonDashboard() {
  return (
    <div className="flex flex-col gap-6">
      {/* 3-column grid of skeleton panel cards */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        {[0, 1, 2].map((i) => (
          <div
            key={i}
            data-testid="skeleton-card"
            className="bg-white dark:bg-gray-800 rounded-xl shadow-md p-5 animate-pulse"
          >
            {/* Accent bar placeholder */}
            <div className="h-1 w-full rounded-full bg-gray-200 dark:bg-gray-700 mb-3" />

            {/* Label placeholder */}
            <div className="h-3 w-1/3 rounded bg-gray-200 dark:bg-gray-700 mb-3" />

            {/* Balance placeholder */}
            <div className="h-7 w-2/3 rounded bg-gray-200 dark:bg-gray-700 mb-4" />

            {/* Account lines */}
            <div className="space-y-2">
              <div className="h-3 w-full rounded bg-gray-200 dark:bg-gray-700" />
              <div className="h-3 w-5/6 rounded bg-gray-200 dark:bg-gray-700" />
              <div className="h-3 w-4/6 rounded bg-gray-200 dark:bg-gray-700" />
            </div>
          </div>
        ))}
      </div>

      {/* Chart area skeleton */}
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-md p-5 animate-pulse">
        <div className="h-4 w-1/4 rounded bg-gray-200 dark:bg-gray-700 mb-4" />
        <div className="h-48 w-full rounded bg-gray-200 dark:bg-gray-700" />
      </div>
    </div>
  )
}
