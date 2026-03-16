import { useState, useEffect } from 'react'
import {
  getAccounts,
  type Operand,
  type CreateAlertRequest,
  type AccountItem,
  type GroupItem,
} from '../api/client'

interface AlertRuleFormProps {
  initialData?: {
    name: string
    operands: Operand[]
    comparison: '<' | '<=' | '>' | '>=' | '=='
    threshold: string
    notify_on_recovery: boolean
  }
  onSave: (data: CreateAlertRequest) => Promise<void>
  onCancel: () => void
  isEdit?: boolean
}

function makeDefaultOperand(): Operand {
  return {
    id: crypto.randomUUID(),
    type: 'bucket',
    ref: 'liquid',
    label: 'Liquid Balance',
    operator: '+',
  }
}

const BUCKET_OPTIONS: { ref: string; label: string }[] = [
  { ref: 'liquid', label: 'Liquid Balance' },
  { ref: 'savings', label: 'Savings Balance' },
  { ref: 'investments', label: 'Investments Balance' },
  { ref: 'net_worth', label: 'Net Worth' },
]

function parseOperandValue(value: string): Pick<Operand, 'type' | 'ref' | 'label'> {
  const [type, ...rest] = value.split(':')
  const ref = rest.join(':')
  if (type === 'bucket') {
    const bucket = BUCKET_OPTIONS.find((b) => b.ref === ref)
    return { type: 'bucket', ref, label: bucket?.label ?? ref }
  }
  if (type === 'group') {
    // label is encoded after the second colon
    const [groupRef, ...labelParts] = ref.split(':')
    return { type: 'group', ref: groupRef, label: labelParts.join(':') }
  }
  // account
  const [accountRef, ...labelParts] = ref.split(':')
  return { type: 'account', ref: accountRef, label: labelParts.join(':') }
}

function encodeOperandValue(op: Operand): string {
  if (op.type === 'bucket') return `bucket:${op.ref}`
  return `${op.type}:${op.ref}:${op.label}`
}

export default function AlertRuleForm({ initialData, onSave, onCancel, isEdit }: AlertRuleFormProps) {
  const [name, setName] = useState(initialData?.name ?? '')
  const [operands, setOperands] = useState<Operand[]>(
    initialData?.operands?.length ? initialData.operands : [makeDefaultOperand()]
  )
  const [comparison, setComparison] = useState<string>(initialData?.comparison ?? '<')
  const [threshold, setThreshold] = useState(initialData?.threshold ?? '')
  const [notifyOnRecovery, setNotifyOnRecovery] = useState(initialData?.notify_on_recovery ?? true)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [saving, setSaving] = useState(false)
  const [accounts, setAccounts] = useState<AccountItem[]>([])
  const [groups, setGroups] = useState<GroupItem[]>([])

  useEffect(() => {
    async function load() {
      try {
        const data = await getAccounts()
        const allAccounts = [
          ...data.liquid,
          ...data.savings,
          ...data.investments,
          ...data.other,
        ]
        setAccounts(allAccounts)
        setGroups(data.groups)
      } catch {
        // Silently fail; dropdown will just have bucket options
      }
    }
    load()
  }, [])

  function validate(): boolean {
    const newErrors: Record<string, string> = {}
    if (!name.trim()) {
      newErrors.name = 'Rule name is required'
    }
    if (operands.length === 0) {
      newErrors.operands = 'Add at least one term'
    }
    if (!threshold.trim()) {
      newErrors.threshold = 'Threshold is required'
    } else if (isNaN(Number(threshold))) {
      newErrors.threshold = 'Threshold must be a number'
    }
    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!validate()) return
    setSaving(true)
    try {
      await onSave({
        name: name.trim(),
        operands,
        comparison,
        threshold: threshold.trim(),
        notify_on_recovery: notifyOnRecovery,
      })
    } catch {
      // Parent handles error
    } finally {
      setSaving(false)
    }
  }

  function addOperand() {
    setOperands((prev) => [...prev, makeDefaultOperand()])
  }

  function removeOperand(id: string) {
    setOperands((prev) => prev.filter((op) => op.id !== id))
  }

  function toggleOperator(id: string) {
    setOperands((prev) =>
      prev.map((op) =>
        op.id === id ? { ...op, operator: op.operator === '+' ? '-' : '+' } : op
      )
    )
  }

  function updateOperand(id: string, value: string) {
    const parsed = parseOperandValue(value)
    setOperands((prev) =>
      prev.map((op) =>
        op.id === id ? { ...op, ...parsed } : op
      )
    )
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="bg-white dark:bg-gray-800 rounded-xl shadow-md p-6 border border-gray-200 dark:border-gray-700"
    >
      <div className="space-y-4">
        {/* Rule Name */}
        <div>
          <label htmlFor="rule-name" className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-1">
            Rule Name
          </label>
          <input
            id="rule-name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g., Low cash warning"
            aria-describedby={errors.name ? 'rule-name-error' : undefined}
            className={`w-full px-3 py-2 border ${errors.name ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'} rounded-lg text-gray-900 dark:text-gray-100 bg-white dark:bg-gray-800 placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent`}
          />
          {errors.name && (
            <p id="rule-name-error" className="text-xs text-red-600 dark:text-red-400 mt-1">{errors.name}</p>
          )}
        </div>

        {/* When section (operands) */}
        <div>
          <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
            When
          </label>
          <div className="space-y-2">
            {operands.map((op, index) => (
              <div key={op.id} className="flex items-center gap-2">
                {/* Operator toggle (hidden on first row) */}
                {index > 0 ? (
                  <button
                    type="button"
                    onClick={() => toggleOperator(op.id)}
                    className="w-8 h-8 flex items-center justify-center rounded-md text-sm font-semibold border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                  >
                    {op.operator}
                  </button>
                ) : (
                  <div className="w-8 h-8 shrink-0" />
                )}

                {/* Operand dropdown */}
                <select
                  value={encodeOperandValue(op)}
                  onChange={(e) => updateOperand(op.id, e.target.value)}
                  className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-sm text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <optgroup label="Buckets">
                    {BUCKET_OPTIONS.map((b) => (
                      <option key={b.ref} value={`bucket:${b.ref}`}>{b.label}</option>
                    ))}
                  </optgroup>
                  {groups.length > 0 && (
                    <optgroup label="Groups">
                      {groups.map((g) => (
                        <option key={`group-${g.id}`} value={`group:${String(g.id)}:${g.name}`}>{g.name}</option>
                      ))}
                    </optgroup>
                  )}
                  {accounts.length > 0 && (
                    <optgroup label="Accounts">
                      {accounts.map((a) => (
                        <option key={`account-${a.id}`} value={`account:${a.id}:${a.name}`}>{a.name}</option>
                      ))}
                    </optgroup>
                  )}
                </select>

                {/* Remove button (hidden when only 1 operand) */}
                {operands.length > 1 ? (
                  <button
                    type="button"
                    onClick={() => removeOperand(op.id)}
                    aria-label="Remove term"
                    className="w-8 h-8 flex items-center justify-center rounded-md text-gray-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                      <path d="M18 6 6 18" />
                      <path d="m6 6 12 12" />
                    </svg>
                  </button>
                ) : (
                  <div className="w-8 h-8 shrink-0" />
                )}
              </div>
            ))}
          </div>
          {errors.operands && (
            <p className="text-xs text-red-600 dark:text-red-400 mt-1">{errors.operands}</p>
          )}
          <button
            type="button"
            onClick={addOperand}
            className="text-sm font-semibold text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 cursor-pointer transition-colors mt-2"
          >
            + Add term
          </button>
        </div>

        {/* Comparison section */}
        <div>
          <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
            is
          </label>
          <div className="flex items-center gap-2">
            <select
              value={comparison}
              onChange={(e) => setComparison(e.target.value)}
              className="w-20 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-sm text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="<">&lt;</option>
              <option value="<=">&lt;=</option>
              <option value=">">&gt;</option>
              <option value=">=">&gt;=</option>
              <option value="==">==</option>
            </select>
            <input
              type="text"
              inputMode="decimal"
              value={threshold}
              onChange={(e) => setThreshold(e.target.value)}
              placeholder="0.00"
              aria-describedby={errors.threshold ? 'threshold-error' : undefined}
              className={`flex-1 px-3 py-2 border ${errors.threshold ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'} rounded-lg bg-white dark:bg-gray-800 text-sm text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500`}
            />
          </div>
          {errors.threshold && (
            <p id="threshold-error" className="text-xs text-red-600 dark:text-red-400 mt-1">{errors.threshold}</p>
          )}
        </div>

        {/* Options: Notify on recovery toggle */}
        <div className="flex items-center justify-between">
          <span className="text-sm font-semibold text-gray-700 dark:text-gray-300">
            Notify on recovery
          </span>
          <button
            type="button"
            role="switch"
            aria-checked={notifyOnRecovery}
            aria-label="Notify on recovery"
            onClick={() => setNotifyOnRecovery(!notifyOnRecovery)}
            className={`relative inline-flex w-10 h-[22px] rounded-full transition-colors duration-150 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 ${
              notifyOnRecovery
                ? 'bg-blue-600 dark:bg-blue-500'
                : 'bg-gray-300 dark:bg-gray-600'
            }`}
          >
            <span
              className={`inline-block w-[18px] h-[18px] rounded-full bg-white shadow-sm transition-transform duration-150 ${
                notifyOnRecovery ? 'translate-x-[20px]' : 'translate-x-[2px]'
              }`}
              style={{ marginTop: '2px' }}
            />
          </button>
        </div>

        {/* Action buttons */}
        <div className="flex items-center gap-3 mt-4">
          <button
            type="submit"
            disabled={saving}
            className="bg-blue-600 text-white py-2 px-4 rounded-lg font-semibold text-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {saving ? 'Saving...' : isEdit ? 'Update Rule' : 'Save Rule'}
          </button>
          <button
            type="button"
            onClick={onCancel}
            className="py-2 px-4 rounded-lg text-sm font-semibold text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
          >
            Discard Changes
          </button>
        </div>
      </div>
    </form>
  )
}
