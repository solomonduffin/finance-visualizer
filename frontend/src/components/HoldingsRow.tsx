import type { ProjectionHoldingSetting } from '../api/client'
import { formatCurrency } from '../utils/format'

interface HoldingsRowProps {
  holdings: ProjectionHoldingSetting[]
  accountId: string
  expanded: boolean
  onApyChange: (holdingId: string, apy: string) => void
  onCompoundChange: (holdingId: string, compound: boolean) => void
  onIncludeChange: (holdingId: string, included: boolean) => void
  isDark: boolean
}

export function HoldingsRow({
  holdings,
  accountId,
  expanded,
  onApyChange,
  onCompoundChange,
  onIncludeChange,
}: HoldingsRowProps) {
  return (
    <div
      id={`holdings-${accountId}`}
      className="overflow-hidden transition-[max-height] duration-200 ease-in-out motion-reduce:transition-none"
      style={{ maxHeight: expanded ? `${holdings.length * 60}px` : '0px' }}
    >
      {holdings.map((holding) => (
        <div
          key={holding.holding_id}
          className="grid grid-cols-[1fr_80px_80px_64px] gap-2 items-center py-2 pl-8"
        >
          {/* Holding name and value */}
          <div className="min-w-0">
            <div className="text-sm text-gray-600 dark:text-gray-300 truncate">
              {holding.description || holding.symbol}
            </div>
            <div className="text-xs text-gray-500 dark:text-gray-400">
              {formatCurrency(holding.market_value)}
            </div>
          </div>

          {/* APY input */}
          <div className="flex items-center gap-0.5">
            <input
              type="text"
              inputMode="decimal"
              placeholder="0.0"
              value={holding.apy}
              onChange={(e) => onApyChange(holding.holding_id, e.target.value)}
              className="w-full px-2 py-1.5 border border-gray-300 dark:border-gray-600 rounded-md text-sm text-right text-gray-900 dark:text-gray-100 bg-white dark:bg-gray-800 placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              aria-label={`APY for ${holding.description || holding.symbol}`}
            />
            <span className="text-xs text-gray-500 dark:text-gray-400">%</span>
          </div>

          {/* Compound toggle */}
          <div className="flex justify-center">
            <button
              type="button"
              role="switch"
              aria-checked={holding.compound}
              aria-label={`Compound interest for ${holding.description || holding.symbol}`}
              title={holding.compound ? 'Compound' : 'Simple'}
              onClick={() => onCompoundChange(holding.holding_id, !holding.compound)}
              className={`relative inline-flex w-8 h-[18px] rounded-full transition-colors duration-200 ${
                holding.compound
                  ? 'bg-blue-600 dark:bg-blue-500'
                  : 'bg-gray-300 dark:bg-gray-600'
              }`}
            >
              <span
                className={`absolute top-[2px] w-[14px] h-[14px] rounded-full bg-white transition-transform duration-200 ${
                  holding.compound ? 'translate-x-[14px]' : 'translate-x-[2px]'
                }`}
              />
            </button>
          </div>

          {/* Include checkbox */}
          <div className="flex justify-center">
            <input
              type="checkbox"
              checked={holding.included}
              onChange={(e) => onIncludeChange(holding.holding_id, e.target.checked)}
              className="w-4 h-4 rounded border border-gray-300 dark:border-gray-600 text-blue-600 focus:ring-2 focus:ring-blue-500 accent-blue-600"
              aria-label={`Include ${holding.description || holding.symbol} in projection`}
            />
          </div>
        </div>
      ))}
    </div>
  )
}
