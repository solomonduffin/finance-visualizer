import { useState } from 'react'
import type { ProjectionAccountSetting } from '../api/client'
import { PANEL_COLORS } from './panelColors'
import { formatCurrency } from '../utils/format'
import { HoldingsRow } from './HoldingsRow'

interface RateConfigTableProps {
  accounts: ProjectionAccountSetting[]
  onApyChange: (accountId: string, apy: string) => void
  onHoldingApyChange: (holdingId: string, apy: string) => void
  onCompoundChange: (accountId: string, compound: boolean) => void
  onHoldingCompoundChange: (holdingId: string, compound: boolean) => void
  onIncludeChange: (accountId: string, included: boolean) => void
  onHoldingIncludeChange: (holdingId: string, included: boolean) => void
  isDark: boolean
}

type PanelKey = 'liquid' | 'savings' | 'investments'

const PANEL_TYPE_MAP: Record<string, PanelKey> = {
  checking: 'liquid',
  credit: 'liquid',
  savings: 'savings',
  brokerage: 'investments',
  retirement: 'investments',
  crypto: 'investments',
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

export function RateConfigTable({
  accounts,
  onApyChange,
  onHoldingApyChange,
  onCompoundChange,
  onHoldingCompoundChange,
  onIncludeChange,
  onHoldingIncludeChange,
  isDark,
}: RateConfigTableProps) {
  const [expandedAccounts, setExpandedAccounts] = useState<Set<string>>(new Set())

  function toggleExpand(accountId: string) {
    setExpandedAccounts((prev) => {
      const next = new Set(prev)
      if (next.has(accountId)) next.delete(accountId)
      else next.add(accountId)
      return next
    })
  }

  // Group accounts by panel type
  const groups: Record<PanelKey, ProjectionAccountSetting[]> = {
    liquid: [],
    savings: [],
    investments: [],
  }

  for (const account of accounts) {
    const panelKey = PANEL_TYPE_MAP[account.account_type] ?? 'liquid'
    groups[panelKey].push(account)
  }

  const panelOrder: PanelKey[] = ['liquid', 'savings', 'investments']

  if (accounts.length === 0) {
    return (
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 p-6">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
          Projection Settings
        </h2>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          No accounts found. Sync your accounts first.
        </p>
      </div>
    )
  }

  function handleMasterIncludeChange(account: ProjectionAccountSetting, checked: boolean) {
    onIncludeChange(account.account_id, checked)
    for (const holding of account.holdings) {
      onHoldingIncludeChange(holding.holding_id, checked)
    }
  }

  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 p-6">
      <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
        Projection Settings
      </h2>

      {/* Column headers - hidden on mobile */}
      <div className="hidden sm:grid grid-cols-[1fr_80px_80px_64px] gap-2 items-center pb-3 border-b border-gray-200 dark:border-gray-700">
        <span className="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">
          Account
        </span>
        <span className="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400 text-center">
          APY %
        </span>
        <span className="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400 text-center">
          Compound
        </span>
        <span className="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400 text-center">
          Include
        </span>
      </div>

      {panelOrder.map((panelKey) => {
        const panelAccounts = groups[panelKey]
        if (panelAccounts.length === 0) return null
        const colors = PANEL_COLORS[panelKey]
        const panelColor = isDark ? colors.darkAccent : colors.accent

        return (
          <div key={panelKey}>
            {/* Panel group header */}
            <div className="flex items-center gap-2 pt-4 pb-2">
              <div
                className="w-2 h-2 rounded-full"
                style={{ backgroundColor: panelColor }}
              />
              <span
                className="text-sm font-semibold uppercase tracking-wide"
                style={{ color: panelColor }}
              >
                {colors.label}
              </span>
            </div>

            {panelAccounts.map((account) => {
              const hasHoldings = account.holdings.length > 0
              const isExpanded = expandedAccounts.has(account.account_id)

              if (hasHoldings) {
                // Investment account WITH holdings - expandable
                return (
                  <div key={account.account_id}>
                    {/* Desktop layout */}
                    <div className="hidden sm:grid grid-cols-[1fr_80px_80px_64px] gap-2 items-center py-2 border-b border-gray-100 dark:border-gray-700/50">
                      {/* Account name with chevron */}
                      <div
                        className="flex items-center gap-1 cursor-pointer"
                        role="button"
                        tabIndex={0}
                        aria-expanded={isExpanded}
                        aria-controls={`holdings-${account.account_id}`}
                        onClick={() => toggleExpand(account.account_id)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter' || e.key === ' ') {
                            e.preventDefault()
                            toggleExpand(account.account_id)
                          }
                        }}
                      >
                        <ChevronIcon expanded={isExpanded} />
                        <span className="text-sm text-gray-800 dark:text-gray-200 truncate">
                          {account.account_name}
                        </span>
                        <span className="text-xs text-gray-500 dark:text-gray-400 ml-2">
                          {formatCurrency(account.balance)}
                        </span>
                      </div>

                      {/* APY - hidden for accounts with holdings */}
                      <div className="text-center text-sm text-gray-400 dark:text-gray-600">
                        &mdash;
                      </div>

                      {/* Compound - hidden for accounts with holdings */}
                      <div className="text-center text-sm text-gray-400 dark:text-gray-600">
                        &mdash;
                      </div>

                      {/* Master include checkbox */}
                      <div className="flex justify-center">
                        <input
                          type="checkbox"
                          checked={account.included}
                          onChange={(e) => handleMasterIncludeChange(account, e.target.checked)}
                          className="w-4 h-4 rounded border border-gray-300 dark:border-gray-600 text-blue-600 focus:ring-2 focus:ring-blue-500 accent-blue-600"
                          aria-label={`Include ${account.account_name} in projection`}
                        />
                      </div>
                    </div>

                    {/* Mobile layout */}
                    <div className="sm:hidden py-2 border-b border-gray-100 dark:border-gray-700/50">
                      <div
                        className="grid grid-cols-[1fr_auto] gap-2 items-center cursor-pointer"
                        role="button"
                        tabIndex={0}
                        aria-expanded={isExpanded}
                        aria-controls={`holdings-${account.account_id}`}
                        onClick={() => toggleExpand(account.account_id)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter' || e.key === ' ') {
                            e.preventDefault()
                            toggleExpand(account.account_id)
                          }
                        }}
                      >
                        <div className="flex items-center gap-1">
                          <ChevronIcon expanded={isExpanded} />
                          <span className="text-sm text-gray-800 dark:text-gray-200 truncate">
                            {account.account_name}
                          </span>
                          <span className="text-xs text-gray-500 dark:text-gray-400 ml-2">
                            {formatCurrency(account.balance)}
                          </span>
                        </div>
                        <input
                          type="checkbox"
                          checked={account.included}
                          onChange={(e) => {
                            e.stopPropagation()
                            handleMasterIncludeChange(account, e.target.checked)
                          }}
                          className="w-4 h-4 rounded border border-gray-300 dark:border-gray-600 text-blue-600 focus:ring-2 focus:ring-blue-500 accent-blue-600"
                          aria-label={`Include ${account.account_name} in projection`}
                        />
                      </div>
                    </div>

                    {/* Holdings sub-rows */}
                    <HoldingsRow
                      holdings={account.holdings}
                      accountId={account.account_id}
                      expanded={isExpanded}
                      onApyChange={onHoldingApyChange}
                      onCompoundChange={onHoldingCompoundChange}
                      onIncludeChange={onHoldingIncludeChange}
                      isDark={isDark}
                    />
                  </div>
                )
              }

              // Standard account row (no holdings)
              return (
                <div key={account.account_id}>
                  {/* Desktop layout */}
                  <div className="hidden sm:grid grid-cols-[1fr_80px_80px_64px] gap-2 items-center py-2 border-b border-gray-100 dark:border-gray-700/50">
                    <span className="text-sm text-gray-800 dark:text-gray-200 truncate">
                      {account.account_name}
                    </span>

                    {/* APY input */}
                    <div className="flex items-center gap-0.5">
                      <input
                        type="text"
                        inputMode="decimal"
                        placeholder="0.0"
                        value={account.apy}
                        onChange={(e) => onApyChange(account.account_id, e.target.value)}
                        className="w-full px-2 py-1.5 border border-gray-300 dark:border-gray-600 rounded-md text-sm text-right text-gray-900 dark:text-gray-100 bg-white dark:bg-gray-800 placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                        aria-label={`APY for ${account.account_name}`}
                      />
                      <span className="text-xs text-gray-500 dark:text-gray-400">%</span>
                    </div>

                    {/* Compound toggle */}
                    <div className="flex justify-center">
                      <button
                        type="button"
                        role="switch"
                        aria-checked={account.compound}
                        aria-label={`Compound interest for ${account.account_name}`}
                        title={account.compound ? 'Compound' : 'Simple'}
                        onClick={() => onCompoundChange(account.account_id, !account.compound)}
                        className={`relative inline-flex w-8 h-[18px] rounded-full transition-colors duration-200 ${
                          account.compound
                            ? 'bg-blue-600 dark:bg-blue-500'
                            : 'bg-gray-300 dark:bg-gray-600'
                        }`}
                      >
                        <span
                          className={`absolute top-[2px] w-[14px] h-[14px] rounded-full bg-white transition-transform duration-200 ${
                            account.compound ? 'translate-x-[14px]' : 'translate-x-[2px]'
                          }`}
                        />
                      </button>
                    </div>

                    {/* Include checkbox */}
                    <div className="flex justify-center">
                      <input
                        type="checkbox"
                        checked={account.included}
                        onChange={(e) => onIncludeChange(account.account_id, e.target.checked)}
                        className="w-4 h-4 rounded border border-gray-300 dark:border-gray-600 text-blue-600 focus:ring-2 focus:ring-blue-500 accent-blue-600"
                        aria-label={`Include ${account.account_name} in projection`}
                      />
                    </div>
                  </div>

                  {/* Mobile layout */}
                  <div className="sm:hidden py-2 border-b border-gray-100 dark:border-gray-700/50">
                    <div className="grid grid-cols-[1fr_auto] gap-2 items-center">
                      <span className="text-sm text-gray-800 dark:text-gray-200 truncate">
                        {account.account_name}
                      </span>
                      <input
                        type="checkbox"
                        checked={account.included}
                        onChange={(e) => onIncludeChange(account.account_id, e.target.checked)}
                        className="w-4 h-4 rounded border border-gray-300 dark:border-gray-600 text-blue-600 focus:ring-2 focus:ring-blue-500 accent-blue-600"
                        aria-label={`Include ${account.account_name} in projection`}
                      />
                    </div>
                    <div className="flex items-center gap-3 mt-2">
                      <div className="flex items-center gap-0.5">
                        <input
                          type="text"
                          inputMode="decimal"
                          placeholder="0.0"
                          value={account.apy}
                          onChange={(e) => onApyChange(account.account_id, e.target.value)}
                          className="w-20 px-2 py-1.5 border border-gray-300 dark:border-gray-600 rounded-md text-sm text-right text-gray-900 dark:text-gray-100 bg-white dark:bg-gray-800 placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                          aria-label={`APY for ${account.account_name}`}
                        />
                        <span className="text-xs text-gray-500 dark:text-gray-400">%</span>
                      </div>
                      <button
                        type="button"
                        role="switch"
                        aria-checked={account.compound}
                        aria-label={`Compound interest for ${account.account_name}`}
                        title={account.compound ? 'Compound' : 'Simple'}
                        onClick={() => onCompoundChange(account.account_id, !account.compound)}
                        className={`relative inline-flex w-8 h-[18px] rounded-full transition-colors duration-200 ${
                          account.compound
                            ? 'bg-blue-600 dark:bg-blue-500'
                            : 'bg-gray-300 dark:bg-gray-600'
                        }`}
                      >
                        <span
                          className={`absolute top-[2px] w-[14px] h-[14px] rounded-full bg-white transition-transform duration-200 ${
                            account.compound ? 'translate-x-[14px]' : 'translate-x-[2px]'
                          }`}
                        />
                      </button>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        )
      })}
    </div>
  )
}
