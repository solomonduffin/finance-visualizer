import { PANEL_COLORS } from './panelColors'
import { formatCurrency } from '../utils/format'

interface Account {
  id: string
  name: string
  balance: string
}

interface PanelCardProps {
  panelKey: 'liquid' | 'savings' | 'investments'
  total: string
  accounts: Account[]
}

export function PanelCard({ panelKey, total, accounts }: PanelCardProps) {
  const colors = PANEL_COLORS[panelKey]

  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl shadow-md p-5 flex flex-col gap-3 border border-transparent dark:border-gray-700">
      {/* Accent bar */}
      <div
        className="h-1 w-full rounded-full"
        style={{ backgroundColor: colors.accent }}
        aria-hidden="true"
      />

      {/* Label */}
      <p className="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">
        {colors.label}
      </p>

      {/* Total balance */}
      <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">
        {formatCurrency(total)}
      </p>

      {/* Account list */}
      {accounts.length > 0 && (
        <ul className="space-y-1 mt-1">
          {accounts.map((account) => (
            <li
              key={account.id}
              className="flex justify-between items-center text-sm"
            >
              <span className="text-gray-600 dark:text-gray-300 truncate pr-2">
                {account.name}
              </span>
              <span className="text-gray-800 dark:text-gray-200 font-medium shrink-0">
                {formatCurrency(account.balance)}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
