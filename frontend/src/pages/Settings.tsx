import { useState, useEffect, type FormEvent } from 'react'
import { getSettings, saveSettings, triggerSync, type SettingsResponse } from '../api/client'
import { timeAgo } from '../utils/time'

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

  async function loadSettings() {
    try {
      const s = await getSettings()
      setSettings(s)
    } catch {
      setError('Failed to load settings.')
    }
  }

  useEffect(() => {
    loadSettings()
  }, [])

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
      await triggerSync()
      setSyncMessage('Sync triggered')
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

  return (
    <div className="min-h-screen bg-gray-50 py-12 px-4">
      <div className="max-w-lg mx-auto">
        {/* Header */}
        <div className="mb-6 flex items-center justify-between">
          <h1 className="text-2xl font-semibold text-gray-900">Settings</h1>
          <button
            type="button"
            onClick={onNavigateDashboard}
            className="text-sm text-blue-600 hover:text-blue-800 font-medium"
          >
            &larr; Back to Dashboard
          </button>
        </div>

        {/* SimpleFIN Configuration Card */}
        <div className="bg-white rounded-xl shadow-md p-6 mb-6">
          <h2 className="text-lg font-medium text-gray-900 mb-4">SimpleFIN Connection</h2>

          <form onSubmit={handleSave} noValidate>
            <div className="mb-4">
              <label
                htmlFor="access-url"
                className="block text-sm font-medium text-gray-700 mb-1"
              >
                Setup Token or Access URL
              </label>
              <input
                id="access-url"
                type="text"
                value={accessUrl}
                onChange={(e) => setAccessUrl(e.target.value)}
                placeholder="Paste setup token or access URL"
                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-gray-900 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
              <p className="mt-1 text-xs text-gray-500">
                Get a setup token from SimpleFIN Bridge, or paste an existing access URL.
              </p>
            </div>

            {error && (
              <p role="alert" className="text-sm text-red-600 mb-4">
                {error}
              </p>
            )}

            <button
              type="submit"
              disabled={saving || !accessUrl.trim()}
              className="w-full bg-blue-600 text-white py-2 px-4 rounded-lg font-medium hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {saving ? 'Saving…' : 'Save'}
            </button>
          </form>
        </div>

        {/* Status Card */}
        <div className="bg-white rounded-xl shadow-md p-6 mb-6">
          <h2 className="text-lg font-medium text-gray-900 mb-4">Sync Status</h2>

          {settings === null ? (
            <p className="text-sm text-gray-500">Loading…</p>
          ) : (
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <span
                  className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                    settings.configured
                      ? 'bg-green-100 text-green-800'
                      : 'bg-gray-100 text-gray-600'
                  }`}
                >
                  {settings.configured ? 'Configured' : 'Not configured'}
                </span>
              </div>

              {settings.configured && settings.last_sync_at && (
                <div className="text-sm text-gray-600">
                  <span className="font-medium">Last sync:</span>{' '}
                  {timeAgo(settings.last_sync_at)}
                </div>
              )}

              {settings.configured && settings.last_sync_status && (
                <div className="text-sm text-gray-600">
                  <span className="font-medium">Status:</span>{' '}
                  <span
                    className={
                      settings.last_sync_status === 'success'
                        ? 'text-green-700'
                        : 'text-red-600'
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
        <div className="bg-white rounded-xl shadow-md p-6">
          <h2 className="text-lg font-medium text-gray-900 mb-4">Actions</h2>

          {syncMessage && (
            <p className="text-sm text-green-700 mb-3">{syncMessage}</p>
          )}

          <button
            type="button"
            onClick={handleSyncNow}
            disabled={!settings?.configured || syncing}
            className="w-full bg-gray-800 text-white py-2 px-4 rounded-lg font-medium hover:bg-gray-900 focus:outline-none focus:ring-2 focus:ring-gray-700 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {syncing ? 'Syncing…' : 'Sync Now'}
          </button>
        </div>
      </div>
    </div>
  )
}
