import type { GroupItem } from '../api/client'
import { formatCurrency } from '../utils/format'
import { getAccountDisplayName } from '../utils/account'
import { GrowthBadge } from './GrowthBadge'

function ChevronIcon({ expanded }: { expanded: boolean }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24"
      fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"
      className={`transition-transform duration-200 motion-reduce:transition-none ${expanded ? 'rotate-90' : ''}`}
      aria-hidden="true"
    >
      <path d="m9 18 6-6-6-6" />
    </svg>
  )
}

interface GroupRowProps {
  group: GroupItem
  pctChange?: string | null
  dollarChange?: string | null
  growthVisible?: boolean
  expanded: boolean
  onToggle: () => void
}

export function GroupRow({ group, pctChange, dollarChange, growthVisible, expanded, onToggle }: GroupRowProps) {
  return (
    <div>
      <div
        className="flex justify-between items-center text-sm py-1 cursor-pointer"
        onClick={onToggle}
        role="button"
        tabIndex={0}
        aria-expanded={expanded}
        aria-controls={`group-members-${group.id}`}
        onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onToggle() } }}
      >
        <div className="flex items-center gap-1">
          <ChevronIcon expanded={expanded} />
          <span className="font-semibold text-gray-800 dark:text-gray-100 truncate">{group.name}</span>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <span className="font-semibold text-gray-800 dark:text-gray-200">{formatCurrency(group.total_balance)}</span>
          <GrowthBadge pctChange={pctChange ?? null} dollarChange={dollarChange ?? null} visible={growthVisible ?? false} />
        </div>
      </div>
      <div
        id={`group-members-${group.id}`}
        className="overflow-hidden transition-all duration-200 ease-in-out motion-reduce:transition-none"
        style={{ maxHeight: expanded ? `${group.members.length * 40}px` : '0px' }}
      >
        <div className="pl-6">
          {group.members.map((member) => (
            <div key={member.id} className="flex justify-between items-center text-sm py-1">
              <span className="text-gray-600 dark:text-gray-300 truncate pr-2">
                {getAccountDisplayName(member)}
              </span>
              <span className="text-gray-800 dark:text-gray-200 font-semibold shrink-0">
                {formatCurrency(member.balance)}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
