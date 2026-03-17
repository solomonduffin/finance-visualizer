import { useState, useEffect, useRef, useMemo, useCallback } from 'react'
import { Link } from 'react-router-dom'
import {
  getProjectionSettings,
  saveProjectionSettings,
  saveIncomeSettings,
  getNetWorth,
  type ProjectionAccountSetting,
  type ProjectionIncomeSettings,
} from '../api/client'
import {
  calculateProjection,
  type AccountProjection,
  type HoldingProjection,
  type IncomeSettings,
} from '../utils/projection'
import { ProjectionChart } from '../components/ProjectionChart'
import { RateConfigTable } from '../components/RateConfigTable'
import { IncomeModelingSection } from '../components/IncomeModelingSection'
import { HorizonSelector } from '../components/HorizonSelector'
import { useDarkMode } from '../hooks/useDarkMode'

export default function Projections() {
  const { isDark } = useDarkMode()

  // Loading / error
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Projection settings (from API)
  const [accounts, setAccounts] = useState<ProjectionAccountSetting[]>([])
  const [income, setIncome] = useState<ProjectionIncomeSettings>({
    enabled: false,
    annual_income: '0',
    monthly_savings_pct: '0',
    allocation_json: '{}',
  })

  // Historical data (for chart)
  const [historicalData, setHistoricalData] = useState<Array<{ date: string; value: number }>>([])

  // UI state
  const [horizonYears, setHorizonYears] = useState(5)

  // Debounce refs
  const saveTimeoutRef = useRef<number>()
  const incomeTimeoutRef = useRef<number>()

  // Cleanup timeouts on unmount
  useEffect(() => {
    return () => {
      clearTimeout(saveTimeoutRef.current)
      clearTimeout(incomeTimeoutRef.current)
    }
  }, [])

  // Data fetching on mount
  const load = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const [settingsRes, netWorthRes] = await Promise.all([
        getProjectionSettings(),
        getNetWorth(180), // ~6 months of historical data
      ])
      setAccounts(settingsRes.accounts)
      setIncome(settingsRes.income)
      // Transform net worth points to total net worth per date
      setHistoricalData(
        netWorthRes.points.map((p) => ({
          date: p.date,
          value:
            parseFloat(p.liquid) +
            parseFloat(p.savings) +
            parseFloat(p.investments),
        })),
      )
    } catch {
      setError('Something went wrong loading projection data')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  // Parse allocation_json safely
  function parseAllocationJson(json: string): Record<string, number> {
    try {
      const parsed = JSON.parse(json)
      if (typeof parsed === 'object' && parsed !== null) return parsed
      return {}
    } catch {
      return {}
    }
  }

  // Projection calculation (via useMemo)
  const projectionData = useMemo(() => {
    const allocationMap = parseAllocationJson(income.allocation_json)

    const accountProjections: AccountProjection[] = accounts.map((a) => ({
      id: a.account_id,
      currentBalance: parseFloat(a.balance) || 0,
      apy: parseFloat(a.apy) || 0,
      compound: a.compound,
      included: a.included,
      allocation: allocationMap[`acct:${a.account_id}`] ?? 0,
      hasHoldings: a.holdings.length > 0,
    }))

    const holdingProjections: HoldingProjection[] = accounts.flatMap((a) =>
      a.holdings.map((h) => ({
        id: h.holding_id,
        accountId: a.account_id,
        currentValue: parseFloat(h.market_value) || 0,
        apy: parseFloat(h.apy) || 0,
        compound: h.compound,
        included: h.included,
        allocation: parseFloat(h.allocation) || 0,
      })),
    )

    const incomeSettings: IncomeSettings = {
      enabled: income.enabled,
      annualIncome: parseFloat(income.annual_income) || 0,
      monthlySavingsPct: parseFloat(income.monthly_savings_pct) || 0,
    }

    return calculateProjection(
      accountProjections,
      holdingProjections,
      incomeSettings,
      horizonYears,
    )
  }, [accounts, income, horizonYears])

  // Debounced auto-save for account/holding settings
  function debouncedSaveSettings(updatedAccounts: ProjectionAccountSetting[]) {
    clearTimeout(saveTimeoutRef.current)
    saveTimeoutRef.current = window.setTimeout(async () => {
      try {
        await saveProjectionSettings({
          accounts: updatedAccounts.map((a) => ({
            account_id: a.account_id,
            apy: a.apy,
            compound: a.compound,
            included: a.included,
          })),
          holdings: updatedAccounts.flatMap((a) =>
            a.holdings.map((h) => ({
              holding_id: h.holding_id,
              account_id: a.account_id,
              apy: h.apy,
              compound: h.compound,
              included: h.included,
              allocation: h.allocation,
            })),
          ),
        })
      } catch {
        // Save failed silently - settings will be retried on next change
      }
    }, 500)
  }

  // Debounced auto-save for income settings
  function debouncedSaveIncome(updatedIncome: ProjectionIncomeSettings) {
    clearTimeout(incomeTimeoutRef.current)
    incomeTimeoutRef.current = window.setTimeout(async () => {
      try {
        await saveIncomeSettings(updatedIncome)
      } catch {
        // Save failed silently
      }
    }, 500)
  }

  // onChange handlers for RateConfigTable
  const handleApyChange = useCallback((accountId: string, apy: string) => {
    setAccounts((prev) => {
      const next = prev.map((a) =>
        a.account_id === accountId ? { ...a, apy } : a,
      )
      debouncedSaveSettings(next)
      return next
    })
  }, [])

  const handleHoldingApyChange = useCallback((holdingId: string, apy: string) => {
    setAccounts((prev) => {
      const next = prev.map((a) => ({
        ...a,
        holdings: a.holdings.map((h) =>
          h.holding_id === holdingId ? { ...h, apy } : h,
        ),
      }))
      debouncedSaveSettings(next)
      return next
    })
  }, [])

  const handleCompoundChange = useCallback((accountId: string, compound: boolean) => {
    setAccounts((prev) => {
      const next = prev.map((a) =>
        a.account_id === accountId ? { ...a, compound } : a,
      )
      debouncedSaveSettings(next)
      return next
    })
  }, [])

  const handleHoldingCompoundChange = useCallback((holdingId: string, compound: boolean) => {
    setAccounts((prev) => {
      const next = prev.map((a) => ({
        ...a,
        holdings: a.holdings.map((h) =>
          h.holding_id === holdingId ? { ...h, compound } : h,
        ),
      }))
      debouncedSaveSettings(next)
      return next
    })
  }, [])

  const handleIncludeChange = useCallback((accountId: string, included: boolean) => {
    setAccounts((prev) => {
      const next = prev.map((a) => {
        if (a.account_id !== accountId) return a
        // Master include: cascade to all holdings
        return {
          ...a,
          included,
          holdings: a.holdings.map((h) => ({ ...h, included })),
        }
      })
      debouncedSaveSettings(next)
      return next
    })
  }, [])

  const handleHoldingIncludeChange = useCallback((holdingId: string, included: boolean) => {
    setAccounts((prev) => {
      const next = prev.map((a) => ({
        ...a,
        holdings: a.holdings.map((h) =>
          h.holding_id === holdingId ? { ...h, included } : h,
        ),
      }))
      debouncedSaveSettings(next)
      return next
    })
  }, [])

  // onChange handlers for IncomeModelingSection
  const handleIncomeToggle = useCallback((enabled: boolean) => {
    setIncome((prev) => {
      const next = { ...prev, enabled }
      debouncedSaveIncome(next)
      return next
    })
  }, [])

  const handleAnnualIncomeChange = useCallback((value: string) => {
    setIncome((prev) => {
      const next = { ...prev, annual_income: value }
      debouncedSaveIncome(next)
      return next
    })
  }, [])

  const handleMonthlySavingsPctChange = useCallback((value: string) => {
    setIncome((prev) => {
      const next = { ...prev, monthly_savings_pct: value }
      debouncedSaveIncome(next)
      return next
    })
  }, [])

  const handleAllocationChange = useCallback((id: string, value: string) => {
    setAccounts((prev) => {
      // If id starts with "hold:", update the holding allocation directly
      if (id.startsWith('hold:')) {
        const holdingId = id.slice(5)
        const next = prev.map((a) => ({
          ...a,
          holdings: a.holdings.map((h) =>
            h.holding_id === holdingId ? { ...h, allocation: value } : h,
          ),
        }))
        debouncedSaveSettings(next)
        return next
      }
      return prev
    })

    // If id starts with "acct:", update allocation_json in income
    if (id.startsWith('acct:')) {
      setIncome((prev) => {
        const alloc = parseAllocationJson(prev.allocation_json)
        alloc[id] = parseFloat(value) || 0
        const next = { ...prev, allocation_json: JSON.stringify(alloc) }
        debouncedSaveIncome(next)
        return next
      })
    }
  }, [])

  // Derive allocation targets for IncomeModelingSection
  const allocationTargets = useMemo(() => {
    const targets: Array<{ id: string; name: string; percentage: string }> = []
    const allocationMap = parseAllocationJson(income.allocation_json)

    for (const account of accounts) {
      if (!account.included) continue
      if (account.holdings.length > 0) {
        // Investment account with holdings: add each included holding
        for (const h of account.holdings) {
          if (!h.included) continue
          targets.push({
            id: `hold:${h.holding_id}`,
            name: h.description || h.symbol,
            percentage: h.allocation,
          })
        }
      } else {
        // Account without holdings: add account-level allocation
        targets.push({
          id: `acct:${account.account_id}`,
          name: account.account_name,
          percentage: String(allocationMap[`acct:${account.account_id}`] ?? '0'),
        })
      }
    }
    return targets
  }, [accounts, income.allocation_json])

  // Loading state
  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
        <div className="max-w-5xl mx-auto px-4 py-6">
          <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100 mb-6">
            Projections
          </h1>
          {/* Skeleton chart */}
          <div className="animate-pulse">
            <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 p-6">
              <div className="h-[400px] bg-gray-200 dark:bg-gray-700 rounded" />
            </div>
          </div>
          {/* Skeleton table rows */}
          <div className="mt-6 space-y-3">
            {[1, 2, 3, 4, 5, 6].map((i) => (
              <div
                key={i}
                className="h-10 bg-gray-200 dark:bg-gray-700 rounded animate-pulse"
              />
            ))}
          </div>
        </div>
      </div>
    )
  }

  // Error state
  if (error) {
    return (
      <div className="min-h-screen bg-gray-50 dark:bg-gray-900 flex items-center justify-center">
        <div className="text-center">
          <p className="text-gray-700 dark:text-gray-300 text-lg mb-4">
            {error}
          </p>
          <button
            type="button"
            onClick={load}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors"
          >
            Retry
          </button>
        </div>
      </div>
    )
  }

  // Empty state
  if (accounts.length === 0) {
    return (
      <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
        <div className="max-w-5xl mx-auto px-4 py-6">
          <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100 mb-6">
            Projections
          </h1>
          <div className="flex items-center justify-center min-h-[400px] px-4">
            <div className="bg-white dark:bg-gray-800 rounded-xl shadow-md p-10 max-w-md w-full text-center">
              <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">
                No accounts to project
              </h2>
              <p className="text-sm text-gray-500 dark:text-gray-400 mb-6">
                Sync your financial accounts to start building projections.
              </p>
              <Link
                to="/settings"
                className="inline-block bg-blue-600 hover:bg-blue-700 text-white font-semibold py-2 px-6 rounded-lg transition-colors"
              >
                Go to Settings
              </Link>
            </div>
          </div>
        </div>
      </div>
    )
  }

  // Main content
  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      <div className="max-w-5xl mx-auto px-4 py-6">
        <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100 mb-6">
          Projections
        </h1>

        <HorizonSelector years={horizonYears} onChange={setHorizonYears} />
        <div className="mb-4" />
        <ProjectionChart
          historicalData={historicalData}
          projectionData={projectionData}
          isDark={isDark}
        />
        <div className="mt-6">
          <RateConfigTable
            accounts={accounts}
            onApyChange={handleApyChange}
            onHoldingApyChange={handleHoldingApyChange}
            onCompoundChange={handleCompoundChange}
            onHoldingCompoundChange={handleHoldingCompoundChange}
            onIncludeChange={handleIncludeChange}
            onHoldingIncludeChange={handleHoldingIncludeChange}
            isDark={isDark}
          />
        </div>
        <div className="mt-6">
          <IncomeModelingSection
            enabled={income.enabled}
            annualIncome={income.annual_income}
            monthlySavingsPct={income.monthly_savings_pct}
            allocationTargets={allocationTargets}
            onToggle={handleIncomeToggle}
            onAnnualIncomeChange={handleAnnualIncomeChange}
            onMonthlySavingsPctChange={handleMonthlySavingsPctChange}
            onAllocationChange={handleAllocationChange}
            isDark={isDark}
          />
        </div>
      </div>
    </div>
  )
}
