import { useState, useEffect, useCallback } from 'react'
import { Link } from 'react-router-dom'
import { getNetWorth } from '../api/client'
import type { NetWorthResponse } from '../api/client'
import { StackedAreaChart, prepareNetWorthData } from '../components/StackedAreaChart'
import { NetWorthStats } from '../components/NetWorthStats'
import { TimeRangeSelector } from '../components/TimeRangeSelector'
import { useDarkMode } from '../hooks/useDarkMode'

export default function NetWorth() {
  const { isDark } = useDarkMode()
  const [selectedDays, setSelectedDays] = useState(90)
  const [data, setData] = useState<NetWorthResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)

  const fetchData = useCallback(async () => {
    setError(false)
    setLoading(true)
    try {
      const result = await getNetWorth(selectedDays)
      setData(result)
    } catch {
      setError(true)
    } finally {
      setLoading(false)
    }
  }, [selectedDays])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
        <div className="max-w-6xl mx-auto px-4 py-6">
          <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100 mb-6">
            Net Worth
          </h1>
          {/* Skeleton stats */}
          <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 p-6">
            <div className="flex flex-col gap-4 md:flex-row md:gap-8">
              <div className="animate-pulse">
                <div className="h-3 w-24 bg-gray-200 dark:bg-gray-700 rounded mb-2" />
                <div className="h-8 w-40 bg-gray-200 dark:bg-gray-700 rounded" />
              </div>
              <div className="animate-pulse">
                <div className="h-3 w-24 bg-gray-200 dark:bg-gray-700 rounded mb-2" />
                <div className="h-8 w-32 bg-gray-200 dark:bg-gray-700 rounded" />
              </div>
              <div className="animate-pulse">
                <div className="h-3 w-24 bg-gray-200 dark:bg-gray-700 rounded mb-2" />
                <div className="h-8 w-32 bg-gray-200 dark:bg-gray-700 rounded" />
              </div>
            </div>
          </div>
          {/* Skeleton chart */}
          <div className="mt-4 animate-pulse">
            <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 p-6">
              <div className="h-[400px] bg-gray-200 dark:bg-gray-700 rounded" />
            </div>
          </div>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="min-h-screen bg-gray-50 dark:bg-gray-900 flex items-center justify-center">
        <div className="text-center">
          <p className="text-gray-700 dark:text-gray-300 text-lg mb-4">Something went wrong</p>
          <button
            type="button"
            onClick={fetchData}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors"
          >
            Retry
          </button>
        </div>
      </div>
    )
  }

  if (!data || data.points.length === 0 || !data.stats) {
    return (
      <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
        <div className="max-w-6xl mx-auto px-4 py-6">
          <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100 mb-6">
            Net Worth
          </h1>
          <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 p-12 text-center">
            <p className="text-gray-500 dark:text-gray-400 text-lg mb-4">
              No balance data yet. Sync your accounts to see net worth history.
            </p>
            <Link
              to="/settings"
              className="text-blue-600 hover:text-blue-700 font-medium transition-colors"
            >
              Go to Settings
            </Link>
          </div>
        </div>
      </div>
    )
  }

  const chartData = prepareNetWorthData(data.points)

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      <div className="max-w-6xl mx-auto px-4 py-6">
        <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100 mb-6">
          Net Worth
        </h1>

        <NetWorthStats stats={data.stats} selectedDays={selectedDays} />

        <div className="mt-4 mb-4">
          <TimeRangeSelector selected={selectedDays} onChange={setSelectedDays} />
        </div>

        <StackedAreaChart data={chartData} isDark={isDark} />
      </div>
    </div>
  )
}
