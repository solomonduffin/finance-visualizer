import { useState, useEffect, useCallback, type FormEvent } from 'react'
import { getSettings, saveSettings, triggerSync, saveGrowthBadgeSetting, getEmailConfig, saveEmailConfig, sendTestEmail, type SettingsResponse } from '../api/client'
import { timeAgo } from '../utils/time'
import AccountsSection from '../components/AccountsSection'
import SyncHistory from '../components/SyncHistory'
import DashboardPreferences from '../components/DashboardPreferences'
import { Toast } from '../components/Toast'

interface SettingsProps {
  onNavigateDashboard: () => void
}

export default function Settings({ onNavigateDashboard }: SettingsProps) {
  const [accessUrl, setAccessUrl] = useState('')
  const [settings, setSettings] = useState<SettingsResponse | null>(null)
  const [saving, setSaving] = useState(false)
  const [syncing, setSyncing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [syncMessage, setSyncMessage] = useState<string | null>(null)
  const [toasts, setToasts] = useState<string[]>([])
  const [growthBadgeEnabled, setGrowthBadgeEnabled] = useState<boolean>(true)
  const [emailConfig, setEmailConfig] = useState({
    host: '', port: '587', username: '', password: '', from: '', to: ''
  })
  const [emailConfigured, setEmailConfigured] = useState(false)
  const [emailSaving, setEmailSaving] = useState(false)
  const [emailTesting, setEmailTesting] = useState(false)
  const [emailFeedback, setEmailFeedback] = useState<{type: 'success' | 'error', message: string} | null>(null)

  async function loadSettings() {
    try {
      const s = await getSettings()
      setSettings(s)
      setGrowthBadgeEnabled(s.growth_badge_enabled)
    } catch {
      setError('Failed to load settings.')
    }
  }

  useEffect(() => {
    loadSettings()
    loadEmailConfig()
  }, [])

  async function loadEmailConfig() {
    try {
      const cfg = await getEmailConfig()
      if (cfg.configured) {
        setEmailConfigured(true)
        setEmailConfig(prev => ({
          ...prev,
          host: cfg.host ?? '',
          port: cfg.port ?? '587',
          username: cfg.username ?? '',
          from: cfg.from ?? '',
          to: cfg.to ?? '',
          password: '', // API never returns password
        }))
      }
    } catch {
      // Silently fail -- email config is optional
    }
  }

  async function handleTestEmail() {
    setEmailTesting(true)
    setEmailFeedback(null)
    try {
      await sendTestEmail(emailConfig)
      setEmailFeedback({ type: 'success', message: 'Test email sent successfully' })
      setTimeout(() => setEmailFeedback(null), 5000)
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Failed to send test email'
      setEmailFeedback({ type: 'error', message })
    } finally {
      setEmailTesting(false)
    }
  }

  async function handleSaveEmailConfig() {
    setEmailSaving(true)
    setEmailFeedback(null)
    try {
      await saveEmailConfig(emailConfig)
      setEmailFeedback({ type: 'success', message: 'Email configuration saved' })
      setEmailConfigured(true)
      setTimeout(() => setEmailFeedback(null), 5000)
    } catch {
      setEmailFeedback({ type: 'error', message: 'Failed to save email configuration' })
    } finally {
      setEmailSaving(false)
    }
  }

  async function handleSave(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    if (!accessUrl.trim()) return
    setError(null)
    setSaving(true)
    try {
      await saveSettings(accessUrl.trim())
      setAccessUrl('')
      await loadSettings()
    } catch {
      setError('Failed to save settings.')
    } finally {
      setSaving(false)
    }
  }

  async function handleSyncNow() {
    if (!settings?.configured || syncing) return
    setError(null)
    setSyncing(true)
    setSyncMessage(null)
    try {
      const result = await triggerSync()
      setSyncMessage('Sync triggered')

      // Show toast for each restored account
      if (result.restored && result.restored.length > 0) {
        const restoredMsg = result.restored.length === 1
          ? `Account "${result.restored[0]}" was restored by sync`
          : `${result.restored.length} accounts restored by sync: ${result.restored.join(', ')}`
        setToasts((prev) => [...prev, restoredMsg])
      }

      setTimeout(async () => {
        await loadSettings()
        setSyncMessage(null)
        setSyncing(false)
      }, 2500)
    } catch {
      setError('Failed to trigger sync.')
      setSyncing(false)
    }
  }

  async function handleToggleGrowthBadge(newValue: boolean) {
    setGrowthBadgeEnabled(newValue)
    try {
      await saveGrowthBadgeSetting(newValue)
    } catch {
      setGrowthBadgeEnabled(!newValue) // Revert
      setToasts((prev) => [...prev, 'Failed to save preference'])
    }
  }

  const handleDismissToast = useCallback((index: number) => {
    setToasts((prev) => prev.filter((_, i) => i !== index))
  }, [])

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 py-12 px-4">
      <div className="max-w-lg mx-auto">
        {/* Header */}
        <div className="mb-6 flex items-center justify-between">
          <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100">Settings</h1>
          <button
            type="button"
            onClick={onNavigateDashboard}
            className="text-sm text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 font-medium"
          >
            &larr; Back to Dashboard
          </button>
        </div>

        {/* SimpleFIN Configuration Card */}
        <div className="bg-white dark:bg-gray-900 rounded-xl shadow-md p-6 mb-6">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">SimpleFIN Connection</h2>

          <form onSubmit={handleSave} noValidate>
            <div className="mb-4">
              <label
                htmlFor="access-url"
                className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
              >
                Setup Token or Access URL
              </label>
              <input
                id="access-url"
                type="text"
                value={accessUrl}
                onChange={(e) => setAccessUrl(e.target.value)}
                placeholder="Paste setup token or access URL"
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-900 dark:text-gray-100 bg-white dark:bg-gray-800 placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
              <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                Get a setup token from SimpleFIN Bridge, or paste an existing access URL.
              </p>
            </div>

            {error && (
              <p role="alert" className="text-sm text-red-600 dark:text-red-400 mb-4">
                {error}
              </p>
            )}

            <button
              type="submit"
              disabled={saving || !accessUrl.trim()}
              className="w-full bg-blue-600 text-white py-2 px-4 rounded-lg font-medium hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {saving ? 'Saving\u2026' : 'Save'}
            </button>
          </form>
        </div>

        {/* Status Card */}
        <div className="bg-white dark:bg-gray-900 rounded-xl shadow-md p-6 mb-6">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">Sync Status</h2>

          {settings === null ? (
            <p className="text-sm text-gray-500 dark:text-gray-400">Loading\u2026</p>
          ) : (
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <span
                  className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                    settings.configured
                      ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                      : 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'
                  }`}
                >
                  {settings.configured ? 'Configured' : 'Not configured'}
                </span>
              </div>

              {settings.configured && settings.last_sync_at && (
                <div className="text-sm text-gray-600 dark:text-gray-400">
                  <span className="font-medium">Last sync:</span>{' '}
                  {timeAgo(settings.last_sync_at)}
                </div>
              )}

              {settings.configured && settings.last_sync_status && (
                <div className="text-sm text-gray-600 dark:text-gray-400">
                  <span className="font-medium">Status:</span>{' '}
                  <span
                    className={
                      settings.last_sync_status === 'success'
                        ? 'text-green-700 dark:text-green-400'
                        : 'text-red-600 dark:text-red-400'
                    }
                  >
                    {settings.last_sync_status === 'success'
                      ? 'Success'
                      : settings.last_sync_status}
                  </span>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Actions Card */}
        <div className="bg-white dark:bg-gray-900 rounded-xl shadow-md p-6 mb-6">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">Actions</h2>

          {syncMessage && (
            <p className="text-sm text-green-700 dark:text-green-400 mb-3">{syncMessage}</p>
          )}

          <button
            type="button"
            onClick={handleSyncNow}
            disabled={!settings?.configured || syncing}
            className="w-full bg-gray-800 dark:bg-gray-700 text-white py-2 px-4 rounded-lg font-medium hover:bg-gray-900 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-gray-700 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {syncing ? 'Syncing\u2026' : 'Sync Now'}
          </button>
        </div>

        {/* Accounts Management Section */}
        {settings?.configured && (
          <div className="mb-6">
            <AccountsSection />
          </div>
        )}

        {/* Sync History */}
        {settings?.configured && (
          <div className="mb-6">
            <SyncHistory />
          </div>
        )}

        {/* Email Configuration */}
        <div className="bg-white dark:bg-gray-900 rounded-xl shadow-md p-6 mb-6">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">Email Configuration</h2>
          <div className="space-y-4">
            {/* SMTP Host */}
            <div>
              <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-1">SMTP Host</label>
              <input
                type="text"
                value={emailConfig.host}
                onChange={e => setEmailConfig(prev => ({...prev, host: e.target.value}))}
                placeholder="smtp.example.com"
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-900 dark:text-gray-100 bg-white dark:bg-gray-800 placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
            {/* Port */}
            <div>
              <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-1">Port</label>
              <input
                type="text"
                inputMode="numeric"
                value={emailConfig.port}
                onChange={e => setEmailConfig(prev => ({...prev, port: e.target.value}))}
                placeholder="587"
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-900 dark:text-gray-100 bg-white dark:bg-gray-800 placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
            {/* Username */}
            <div>
              <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-1">Username</label>
              <input
                type="text"
                value={emailConfig.username}
                onChange={e => setEmailConfig(prev => ({...prev, username: e.target.value}))}
                placeholder="user@example.com"
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-900 dark:text-gray-100 bg-white dark:bg-gray-800 placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
            {/* Password */}
            <div>
              <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-1">Password</label>
              <input
                type="password"
                value={emailConfig.password}
                onChange={e => setEmailConfig(prev => ({...prev, password: e.target.value}))}
                placeholder={emailConfigured ? '********' : ''}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-900 dark:text-gray-100 bg-white dark:bg-gray-800 placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Encrypted at rest. Re-enter to change.</p>
            </div>
            {/* From Address */}
            <div>
              <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-1">From Address</label>
              <input
                type="email"
                value={emailConfig.from}
                onChange={e => setEmailConfig(prev => ({...prev, from: e.target.value}))}
                placeholder="alerts@example.com"
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-900 dark:text-gray-100 bg-white dark:bg-gray-800 placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
            {/* To Address */}
            <div>
              <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-1">To Address</label>
              <input
                type="email"
                value={emailConfig.to}
                onChange={e => setEmailConfig(prev => ({...prev, to: e.target.value}))}
                placeholder="you@example.com"
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-900 dark:text-gray-100 bg-white dark:bg-gray-800 placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
          </div>

          {/* Action buttons */}
          <div className="flex items-center gap-3 mt-4">
            <button
              onClick={handleTestEmail}
              disabled={emailTesting || !emailConfig.host}
              className="py-2 px-4 rounded-lg text-sm font-semibold border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {emailTesting ? 'Sending...' : 'Test Email'}
            </button>
            <button
              onClick={handleSaveEmailConfig}
              disabled={emailSaving || !emailConfig.host}
              className="bg-blue-600 text-white py-2 px-4 rounded-lg font-semibold text-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {emailSaving ? 'Saving...' : 'Save Email Config'}
            </button>
          </div>

          {/* Inline feedback */}
          {emailFeedback && (
            <p className={`text-sm mt-2 ${emailFeedback.type === 'success' ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'}`}>
              {emailFeedback.message}
            </p>
          )}
        </div>

        {/* Dashboard Preferences */}
        <div className="mb-6">
          <DashboardPreferences
            initialValue={growthBadgeEnabled}
            onToggle={handleToggleGrowthBadge}
          />
        </div>
      </div>

      {/* Toast notifications */}
      {toasts.map((msg, i) => (
        <Toast
          key={`${msg}-${i}`}
          message={msg}
          onDismiss={() => handleDismissToast(i)}
        />
      ))}
    </div>
  )
}
