import { useState } from 'react'

interface DashboardPreferencesProps {
  initialValue: boolean
  onToggle: (newValue: boolean) => Promise<void>
}

export default function DashboardPreferences({
  initialValue,
  onToggle,
}: DashboardPreferencesProps) {
  const [enabled, setEnabled] = useState(initialValue)

  async function handleClick() {
    const newValue = !enabled
    setEnabled(newValue)
    try {
      await onToggle(newValue)
    } catch {
      setEnabled(!newValue) // Revert on failure
    }
  }

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl shadow-md p-6">
      <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
        Dashboard Preferences
      </h2>

      <div className="flex items-center justify-between">
        <span className="text-sm font-semibold text-gray-700 dark:text-gray-300">
          Show growth badges
        </span>

        <button
          type="button"
          role="switch"
          aria-checked={enabled}
          aria-label="Show growth badges"
          onClick={handleClick}
          className={`relative inline-flex w-10 h-[22px] rounded-full transition-colors duration-150 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 ${
            enabled
              ? 'bg-blue-600 dark:bg-blue-500'
              : 'bg-gray-300 dark:bg-gray-600'
          }`}
        >
          <span
            className={`inline-block w-[18px] h-[18px] rounded-full bg-white shadow-sm transition-transform duration-150 ${
              enabled ? 'translate-x-[20px]' : 'translate-x-[2px]'
            }`}
            style={{ marginTop: '2px' }}
          />
        </button>
      </div>
    </div>
  )
}
