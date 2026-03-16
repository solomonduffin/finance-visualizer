import { useState, useEffect, useRef } from 'react'
import type { AlertRule, Operand } from '../api/client'
import { timeAgo } from '../utils/time'
import { formatCurrency } from '../utils/format'

interface AlertRuleCardProps {
  rule: AlertRule
  onToggle: (id: number, enabled: boolean) => void
  onEdit: (rule: AlertRule) => void
  onDelete: (id: number) => void
  isEditing?: boolean
  editForm?: React.ReactNode
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
      className={`w-4 h-4 shrink-0 transition-transform duration-200 motion-reduce:transition-none ${expanded ? 'rotate-90' : ''}`}
      aria-hidden="true"
    >
      <path d="m9 18 6-6-6-6" />
    </svg>
  )
}

export function formatExpressionSummary(
  operands: Operand[],
  comparison: string,
  threshold: string
): string {
  const parts = operands.map((op, i) => {
    if (i === 0) return op.label
    return `${op.operator} ${op.label}`
  })
  const formattedThreshold = `$${Number(threshold).toLocaleString()}`
  return `${parts.join(' ')} ${comparison} ${formattedThreshold}`
}

function StatusBadge({ rule }: { rule: AlertRule }) {
  if (!rule.enabled) {
    return (
      <span className="bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400 px-2 py-1 rounded-full text-xs font-semibold">
        Disabled
      </span>
    )
  }
  if (rule.last_state === 'triggered') {
    return (
      <span className="bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200 px-2 py-1 rounded-full text-xs font-semibold">
        Triggered
      </span>
    )
  }
  return (
    <span className="bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200 px-2 py-1 rounded-full text-xs font-semibold">
      Normal
    </span>
  )
}

export default function AlertRuleCard({
  rule,
  onToggle,
  onEdit,
  onDelete,
  isEditing,
  editForm,
}: AlertRuleCardProps) {
  const [expanded, setExpanded] = useState(false)
  const [menuOpen, setMenuOpen] = useState(false)
  const [confirmingDelete, setConfirmingDelete] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  // Close menu on click outside
  useEffect(() => {
    if (!menuOpen) return
    function handleClickOutside(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [menuOpen])

  if (isEditing && editForm) {
    return <>{editForm}</>
  }

  const expressionSummary = formatExpressionSummary(rule.operands, rule.comparison, rule.threshold)
  const historyId = `alert-history-${rule.id}`
  const recentHistory = [...(rule.history || [])].sort(
    (a, b) => new Date(b.notified_at).getTime() - new Date(a.notified_at).getTime()
  ).slice(0, 10)

  function handleToggleExpand() {
    setExpanded((prev) => !prev)
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      handleToggleExpand()
    }
  }

  return (
    <div
      className={`bg-white dark:bg-gray-800 rounded-xl shadow-md p-6 border border-transparent dark:border-gray-700 ${
        !rule.enabled ? 'opacity-60' : ''
      }`}
    >
      {/* Row 1: Header */}
      <div className="flex items-center justify-between">
        <div
          className="flex items-center gap-2 cursor-pointer min-w-0"
          role="button"
          tabIndex={0}
          aria-expanded={expanded}
          aria-controls={historyId}
          aria-label={expanded ? 'Hide alert history' : 'Show alert history'}
          onClick={handleToggleExpand}
          onKeyDown={handleKeyDown}
        >
          <ChevronIcon expanded={expanded} />
          <span className="text-sm font-semibold text-gray-900 dark:text-gray-100 truncate">
            {rule.name}
          </span>
        </div>

        <div className="flex items-center gap-3 shrink-0">
          <StatusBadge rule={rule} />

          {/* Toggle switch */}
          <button
            type="button"
            role="switch"
            aria-checked={rule.enabled}
            aria-label={rule.enabled ? 'Disable alert rule' : 'Enable alert rule'}
            onClick={() => onToggle(rule.id, !rule.enabled)}
            className={`relative inline-flex w-10 h-[22px] rounded-full transition-colors duration-150 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 ${
              rule.enabled
                ? 'bg-blue-600 dark:bg-blue-500'
                : 'bg-gray-300 dark:bg-gray-600'
            }`}
          >
            <span
              className={`inline-block w-[18px] h-[18px] rounded-full bg-white shadow-sm transition-transform duration-150 ${
                rule.enabled ? 'translate-x-[20px]' : 'translate-x-[2px]'
              }`}
              style={{ marginTop: '2px' }}
            />
          </button>

          {/* Actions menu */}
          <div className="relative" ref={menuRef}>
            <button
              type="button"
              aria-label="Alert rule actions"
              aria-haspopup="true"
              aria-expanded={menuOpen}
              onClick={() => setMenuOpen((prev) => !prev)}
              className="p-2 rounded-lg text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <circle cx="12" cy="5" r="1.5" />
                <circle cx="12" cy="12" r="1.5" />
                <circle cx="12" cy="19" r="1.5" />
              </svg>
            </button>

            {menuOpen && (
              <div className="absolute right-0 mt-1 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg py-1 z-10">
                <button
                  type="button"
                  onClick={() => {
                    setMenuOpen(false)
                    onEdit(rule)
                  }}
                  className="w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer"
                >
                  Edit Rule
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setMenuOpen(false)
                    setConfirmingDelete(true)
                  }}
                  className="w-full text-left px-4 py-2 text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 cursor-pointer"
                >
                  Delete Rule
                </button>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Row 2: Expression summary */}
      <p className="text-sm text-gray-500 dark:text-gray-400 mt-1 truncate" title={expressionSummary}>
        {expressionSummary}
      </p>

      {/* Row 3: Metadata */}
      <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
        {rule.last_eval_at ? `Last checked: ${timeAgo(rule.last_eval_at)}` : 'Never checked'}
        {rule.last_value != null && ` \u00B7 Value: ${formatCurrency(rule.last_value)}`}
      </p>

      {/* Expanded history */}
      <div
        id={historyId}
        className={`overflow-hidden transition-[max-height] duration-200 ease-in-out motion-reduce:transition-none ${
          expanded ? 'max-h-96' : 'max-h-0'
        }`}
      >
        <div className="pl-4 mt-3 border-t border-gray-100 dark:border-gray-700 pt-3">
          {recentHistory.length === 0 ? (
            <p className="text-xs text-gray-500 dark:text-gray-400 italic">No events yet</p>
          ) : (
            recentHistory.map((entry) => (
              <div key={entry.id} className="flex items-center gap-2 py-1">
                <span
                  className={`text-xs font-semibold ${
                    entry.state === 'triggered'
                      ? 'text-red-600 dark:text-red-400'
                      : 'text-green-600 dark:text-green-400'
                  }`}
                >
                  {entry.state === 'triggered' ? 'Triggered' : 'Recovered'}
                </span>
                {entry.value != null && (
                  <span className="text-xs text-gray-600 dark:text-gray-300">
                    at {formatCurrency(entry.value)}
                  </span>
                )}
                <span className="text-xs text-gray-500 dark:text-gray-400">
                  {timeAgo(entry.notified_at)}
                </span>
              </div>
            ))
          )}
        </div>
      </div>

      {/* Delete confirmation */}
      {confirmingDelete && (
        <div className="mt-4 p-4 border border-gray-200 dark:border-gray-700 rounded-lg bg-gray-50 dark:bg-gray-900">
          <p className="text-sm text-gray-700 dark:text-gray-300 mb-3">
            Delete this rule? This cannot be undone.
          </p>
          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={() => {
                setConfirmingDelete(false)
                onDelete(rule.id)
              }}
              className="text-sm font-semibold text-white bg-red-600 hover:bg-red-700 py-2 px-4 rounded-lg transition-colors"
            >
              Delete Rule
            </button>
            <button
              type="button"
              onClick={() => setConfirmingDelete(false)}
              className="text-sm font-semibold text-gray-600 dark:text-gray-400 py-2 px-4"
            >
              Keep Rule
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
