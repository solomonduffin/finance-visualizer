import { useState, useEffect } from 'react'
import {
  getAlerts,
  createAlert,
  updateAlertRule,
  toggleAlert,
  deleteAlert,
  type AlertRule,
  type CreateAlertRequest,
} from '../api/client'
import AlertRuleForm from '../components/AlertRuleForm'
import AlertRuleCard from '../components/AlertRuleCard'

export default function Alerts() {
  const [rules, setRules] = useState<AlertRule[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showBuilder, setShowBuilder] = useState(false)
  const [editingRuleId, setEditingRuleId] = useState<number | null>(null)

  useEffect(() => {
    loadRules()
  }, [])

  async function loadRules() {
    setLoading(true)
    setError(null)
    try {
      const data = await getAlerts()
      setRules(data)
    } catch {
      setError('Failed to load alerts')
    } finally {
      setLoading(false)
    }
  }

  async function handleCreate(data: CreateAlertRequest) {
    const newRule = await createAlert(data)
    setRules((prev) => [newRule, ...prev])
    setShowBuilder(false)
  }

  async function handleUpdate(data: CreateAlertRequest) {
    if (editingRuleId == null) return
    const updated = await updateAlertRule(editingRuleId, data)
    setRules((prev) => prev.map((r) => (r.id === editingRuleId ? updated : r)))
    setEditingRuleId(null)
  }

  function handleEdit(rule: AlertRule) {
    setEditingRuleId(rule.id)
    setShowBuilder(false)
  }

  async function handleToggle(id: number, enabled: boolean) {
    // Optimistic update
    setRules((prev) =>
      prev.map((r) => (r.id === id ? { ...r, enabled } : r))
    )
    try {
      const updated = await toggleAlert(id, enabled)
      setRules((prev) => prev.map((r) => (r.id === id ? updated : r)))
    } catch {
      // Revert on error
      setRules((prev) =>
        prev.map((r) => (r.id === id ? { ...r, enabled: !enabled } : r))
      )
    }
  }

  async function handleDelete(id: number) {
    try {
      await deleteAlert(id)
      setRules((prev) => prev.filter((r) => r.id !== id))
    } catch {
      // Could show error toast but keeping it simple
    }
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      <div className="max-w-3xl mx-auto px-4 py-6">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100">Alerts</h1>
          {!showBuilder && (
            <button
              type="button"
              onClick={() => {
                setShowBuilder(true)
                setEditingRuleId(null)
              }}
              className="bg-blue-600 text-white py-2 px-4 rounded-lg text-sm font-semibold hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 transition-colors"
            >
              + New Alert
            </button>
          )}
        </div>

        {/* Builder form */}
        {showBuilder && (
          <div className="mb-4">
            <AlertRuleForm
              onSave={handleCreate}
              onCancel={() => setShowBuilder(false)}
            />
          </div>
        )}

        {/* Content states */}
        {loading ? (
          <div className="space-y-4">
            {[1, 2, 3].map((i) => (
              <div key={i} className="h-24 bg-gray-200 dark:bg-gray-700 rounded-xl animate-pulse" />
            ))}
          </div>
        ) : error ? (
          <div className="text-center py-12">
            <p className="text-sm text-red-600 dark:text-red-400 mb-4">{error}</p>
            <button
              type="button"
              onClick={loadRules}
              className="bg-blue-600 text-white py-2 px-4 rounded-lg text-sm font-semibold hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 transition-colors"
            >
              Retry Loading
            </button>
          </div>
        ) : rules.length === 0 && !showBuilder ? (
          <div className="flex items-center justify-center min-h-[400px] px-4">
            <div className="bg-white dark:bg-gray-800 rounded-xl shadow-md p-10 max-w-md w-full text-center">
              <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">
                No alert rules yet
              </h2>
              <p className="text-sm text-gray-500 dark:text-gray-400 mb-6">
                Create your first alert to get notified when your balances cross a threshold.
              </p>
              <button
                type="button"
                onClick={() => setShowBuilder(true)}
                className="inline-block bg-blue-600 hover:bg-blue-700 text-white font-semibold py-2 px-6 rounded-lg transition-colors"
              >
                Create Alert
              </button>
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            {rules.map((rule) => (
              <AlertRuleCard
                key={rule.id}
                rule={rule}
                onToggle={handleToggle}
                onEdit={handleEdit}
                onDelete={handleDelete}
                isEditing={editingRuleId === rule.id}
                editForm={
                  editingRuleId === rule.id ? (
                    <AlertRuleForm
                      initialData={{
                        name: rule.name,
                        operands: rule.operands,
                        comparison: rule.comparison,
                        threshold: rule.threshold,
                        notify_on_recovery: rule.notify_on_recovery,
                      }}
                      onSave={handleUpdate}
                      onCancel={() => setEditingRuleId(null)}
                      isEdit
                    />
                  ) : undefined
                }
              />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
