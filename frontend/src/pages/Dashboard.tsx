import { useState, useEffect, useCallback } from 'react'
import { getSummary, getAccounts, getBalanceHistory } from '../api/client'
import type { SummaryResponse, AccountsResponse, BalanceHistoryResponse } from '../api/client'
import { SkeletonDashboard } from '../components/SkeletonDashboard'
import { EmptyState } from '../components/EmptyState'
import { PanelCard } from '../components/PanelCard'
import { timeAgo } from '../utils/time'

const PANEL_KEYS = ['liquid', 'savings', 'investments'] as const
type PanelKey = typeof PANEL_KEYS[number]

export default function Dashboard() {
  const [summary, setSummary] = useState<SummaryResponse | null>(null)
  const [accounts, setAccounts] = useState<AccountsResponse | null>(null)
  const [history, setHistory] = useState<BalanceHistoryResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)

  const fetchData = useCallback(async () => {
    setError(false)
    setLoading(true)
    try {
      const [summaryData, accountsData, historyData] = await Promise.all([
        getSummary(),
        getAccounts(),
        getBalanceHistory(30),
      ])
      setSummary(summaryData)
      setAccounts(accountsData)
      setHistory(historyData)
    } catch {
      setError(true)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
        <div className="max-w-6xl mx-auto px-4 py-6">
          <SkeletonDashboard />
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

  if (!summary || summary.last_synced_at === null) {
    return (
      <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
        <EmptyState />
      </div>
    )
  }

  // Visible panels: only render if there are accounts for that panel
  const visiblePanels = PANEL_KEYS.filter(
    (key) => accounts && accounts[key].length > 0
  )

  // Responsive grid: 1 col mobile, 2 col if 2 panels, 3 col if 3 panels
  const gridCols =
    visiblePanels.length === 1
      ? 'grid-cols-1'
      : visiblePanels.length === 2
        ? 'grid-cols-1 md:grid-cols-2'
        : 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3'

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      <div className="max-w-6xl mx-auto px-4 py-6">
        {/* Header row */}
        <div className="flex justify-between items-center">
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">Dashboard</h1>
          <span className="text-sm text-gray-500 dark:text-gray-400">
            Last updated {timeAgo(summary.last_synced_at)}
          </span>
        </div>

        {/* Panel grid */}
        <div className={`grid ${gridCols} gap-4 mt-6`}>
          {visiblePanels.map((key: PanelKey) => (
            <PanelCard
              key={key}
              panelKey={key}
              total={summary[key]}
              accounts={accounts![key]}
            />
          ))}
        </div>

        {/* Charts placeholder — filled in Plan 03 */}
        <div id="charts-section" className="mt-8">
          {/* Charts added in Plan 03 */}
        </div>
      </div>
    </div>
  )
}
