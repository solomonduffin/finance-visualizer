import { useState } from 'react'
import { PANEL_COLORS } from './panelColors'
import { formatCurrency } from '../utils/format'
import { getAccountDisplayName } from '../utils/account'
import { GrowthBadge } from './GrowthBadge'
import { GroupRow } from './GroupRow'
import type { GroupItem, GroupGrowthData } from '../api/client'

interface Account {
  id: string
  name: string
  balance: string
  org_name: string
  display_name?: string | null
}

interface PanelCardProps {
  panelKey: 'liquid' | 'savings' | 'investments'
  total: string
  accounts: Account[]
  groups?: GroupItem[]
  groupGrowth?: GroupGrowthData[]
  pctChange?: string | null
  dollarChange?: string | null
  growthVisible?: boolean
}

export function PanelCard({ panelKey, total, accounts, groups, groupGrowth, pctChange, dollarChange, growthVisible }: PanelCardProps) {
  const colors = PANEL_COLORS[panelKey]
  const [expandedGroups, setExpandedGroups] = useState<Set<number>>(new Set())

  function toggleGroup(id: number) {
    setExpandedGroups((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

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
      <p className="text-2xl font-semibold text-gray-900 dark:text-gray-100 flex items-baseline gap-2">
        <span>{formatCurrency(total)}</span>
        <GrowthBadge
          pctChange={pctChange ?? null}
          dollarChange={dollarChange ?? null}
          visible={growthVisible ?? false}
        />
      </p>

      {/* Group rows */}
      {groups && groups.length > 0 && (
        <div className="space-y-1 mt-1">
          {groups.map((group) => {
            const growth = groupGrowth?.find(g => g.group_id === group.id)?.growth
            return (
              <GroupRow
                key={group.id}
                group={group}
                pctChange={growth?.pct_change ?? null}
                dollarChange={growth?.dollar_change ?? null}
                growthVisible={growthVisible ?? false}
                expanded={expandedGroups.has(group.id)}
                onToggle={() => toggleGroup(group.id)}
              />
            )
          })}
        </div>
      )}

      {/* Account list */}
      {accounts.length > 0 && (
        <ul className="space-y-1 mt-1">
          {accounts.map((account) => (
            <li
              key={account.id}
              className="flex justify-between items-center text-sm"
            >
              <span className="text-gray-600 dark:text-gray-300 truncate pr-2">
                {getAccountDisplayName(account)}
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
