import { useState } from 'react'
import { formatCurrency } from '../utils/format'
import { AllocationRow } from './AllocationRow'

interface AllocationTarget {
  id: string
  name: string
  percentage: string
}

interface IncomeModelingSectionProps {
  enabled: boolean
  annualIncome: string
  monthlySavingsPct: string
  allocationTargets: AllocationTarget[]
  onToggle: (enabled: boolean) => void
  onAnnualIncomeChange: (value: string) => void
  onMonthlySavingsPctChange: (value: string) => void
  onAllocationChange: (id: string, value: string) => void
  isDark: boolean
}

function ChevronIcon({ expanded }: { expanded: boolean }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={`transition-transform duration-200 motion-reduce:transition-none ${expanded ? 'rotate-90' : ''}`}
      aria-hidden="true"
    >
      <path d="m9 18 6-6-6-6" />
    </svg>
  )
}

export function IncomeModelingSection({
  enabled,
  annualIncome,
  monthlySavingsPct,
  allocationTargets,
  onToggle,
  onAnnualIncomeChange,
  onMonthlySavingsPctChange,
  onAllocationChange,
}: IncomeModelingSectionProps) {
  const [expanded, setExpanded] = useState(false)

  const incomeNum = parseFloat(annualIncome) || 0
  const savingsPct = parseFloat(monthlySavingsPct) || 0
  const monthlyAllocation = incomeNum / 12 * savingsPct / 100

  const allocationSum = allocationTargets.reduce(
    (sum, t) => sum + (parseFloat(t.percentage) || 0),
    0
  )

  // Calculate content max-height: rough estimate per element
  const contentHeight = 300 + allocationTargets.length * 50

  return (
    <div
      className={`bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 ${
        !enabled && !expanded ? 'opacity-60' : ''
      }`}
    >
      {/* Header */}
      <div className="flex items-center justify-between p-6">
        {/* Left: chevron + heading */}
        <div
          className="flex items-center gap-2 cursor-pointer"
          role="button"
          tabIndex={0}
          aria-expanded={expanded}
          aria-controls="income-modeling-content"
          onClick={() => setExpanded(!expanded)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault()
              setExpanded(!expanded)
            }
          }}
        >
          <ChevronIcon expanded={expanded} />
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Income Modeling
          </h2>
        </div>

        {/* Right: toggle */}
        <button
          type="button"
          role="switch"
          aria-checked={enabled}
          aria-label="Enable income modeling"
          onClick={(e) => {
            e.stopPropagation()
            onToggle(!enabled)
          }}
          className={`relative inline-flex w-10 h-[22px] rounded-full transition-colors duration-200 ${
            enabled
              ? 'bg-blue-600 dark:bg-blue-500'
              : 'bg-gray-300 dark:bg-gray-600'
          }`}
        >
          <span
            className={`absolute top-[2px] w-[18px] h-[18px] rounded-full bg-white transition-transform duration-200 ${
              enabled ? 'translate-x-[20px]' : 'translate-x-[2px]'
            }`}
          />
        </button>
      </div>

      {/* Expandable content */}
      <div
        id="income-modeling-content"
        className="overflow-hidden transition-[max-height] duration-300 ease-in-out motion-reduce:transition-none"
        style={{ maxHeight: expanded ? `${contentHeight}px` : '0px' }}
      >
        <div
          className={`px-6 pb-6 pt-0 ${
            !enabled ? 'opacity-50 pointer-events-none' : ''
          }`}
        >
          {/* Income inputs */}
          <div className="flex flex-col gap-4 md:flex-row md:gap-6">
            {/* Annual Income */}
            <div className="flex-1 md:w-48 md:flex-none">
              <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-1">
                Annual Income
              </label>
              <div className="flex items-center">
                <span className="text-sm text-gray-500 dark:text-gray-400 mr-1">$</span>
                <input
                  type="text"
                  inputMode="decimal"
                  placeholder="0"
                  value={annualIncome}
                  disabled={!enabled}
                  onChange={(e) => onAnnualIncomeChange(e.target.value)}
                  className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-sm text-gray-900 dark:text-gray-100 bg-white dark:bg-gray-800 placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  aria-label="Annual income amount"
                />
              </div>
            </div>

            {/* Monthly Savings */}
            <div>
              <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-1">
                Monthly Savings
              </label>
              <div className="flex items-center">
                <input
                  type="text"
                  inputMode="decimal"
                  placeholder="0"
                  value={monthlySavingsPct}
                  disabled={!enabled}
                  onChange={(e) => onMonthlySavingsPctChange(e.target.value)}
                  className="w-24 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-sm text-gray-900 dark:text-gray-100 bg-white dark:bg-gray-800 placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  aria-label="Monthly savings percentage"
                />
                <span className="text-xs text-gray-500 dark:text-gray-400 ml-1">%</span>
              </div>
            </div>

            {/* Computed monthly allocation */}
            <div className="flex items-end">
              <p className="text-sm text-gray-600 dark:text-gray-300">
                {formatCurrency(String(monthlyAllocation.toFixed(2)))} / month to allocate
              </p>
            </div>
          </div>

          {/* Divider */}
          <div className="border-t border-gray-200 dark:border-gray-700 my-4" />

          {/* Allocation heading */}
          <h3 className="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-1">
            Savings Allocation
          </h3>
          <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">
            Distribute monthly savings across your included accounts. Must total 100%.
          </p>

          {/* Allocation rows */}
          {allocationTargets.map((target) => (
            <AllocationRow
              key={target.id}
              name={target.name}
              percentage={target.percentage}
              onChange={(value) => onAllocationChange(target.id, value)}
            />
          ))}

          {/* Allocation sum validation */}
          <div className="flex items-center justify-end gap-2 mt-3" role="status">
            {Math.round(allocationSum * 100) / 100 === 100 ? (
              <span className="text-sm font-semibold text-green-600 dark:text-green-400">
                Total: 100%
              </span>
            ) : (
              <span className="text-sm font-semibold text-red-600 dark:text-red-400">
                Total: {Math.round(allocationSum * 100) / 100}% (must equal 100%)
              </span>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
