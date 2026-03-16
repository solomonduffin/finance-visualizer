import { useState, useEffect } from 'react'
import { getSyncLog, type SyncLogEntry } from '../api/client'

function SuccessIcon() {
  return (
    <svg
      data-testid="sync-icon-success"
      xmlns="http://www.w3.org/2000/svg"
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className="w-4 h-4 shrink-0 text-green-600 dark:text-green-400"
      aria-hidden="true"
    >
      <circle cx="12" cy="12" r="10" />
      <path d="m9 12 2 2 4-4" />
    </svg>
  )
}

function PartialIcon() {
  return (
    <svg
      data-testid="sync-icon-partial"
      xmlns="http://www.w3.org/2000/svg"
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className="w-4 h-4 shrink-0 text-amber-500 dark:text-amber-400"
      aria-hidden="true"
    >
      <path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z" />
      <line x1="12" x2="12" y1="9" y2="13" />
      <line x1="12" x2="12.01" y1="17" y2="17" />
    </svg>
  )
}

function FailedIcon() {
  return (
    <svg
      data-testid="sync-icon-failed"
      xmlns="http://www.w3.org/2000/svg"
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className="w-4 h-4 shrink-0 text-red-600 dark:text-red-400"
      aria-hidden="true"
    >
      <circle cx="12" cy="12" r="10" />
      <path d="m15 9-6 6" />
      <path d="m9 9 6 6" />
    </svg>
  )
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

function formatTimestamp(isoString: string): string {
  const date = new Date(isoString)
  const datePart = date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  })
  const timePart = date.toLocaleTimeString('en-US', {
    hour: 'numeric',
    minute: '2-digit',
  })
  return `${datePart} ${timePart}`
}

function StatusIcon({ status }: { status: SyncLogEntry['status'] }) {
  switch (status) {
    case 'success':
      return <SuccessIcon />
    case 'partial':
      return <PartialIcon />
    case 'failed':
      return <FailedIcon />
  }
}

function StatusText({ entry }: { entry: SyncLogEntry }) {
  switch (entry.status) {
    case 'success':
      return (
        <span className="text-sm text-gray-600 dark:text-gray-400 ml-auto">
          {entry.accounts_fetched} accounts synced
        </span>
      )
    case 'partial':
      return (
        <span className="text-sm font-semibold text-amber-600 dark:text-amber-400 ml-auto">
          {entry.accounts_fetched} synced, {entry.accounts_failed} failed
        </span>
      )
    case 'failed':
      return (
        <span className="text-sm font-semibold text-red-600 dark:text-red-400 ml-auto">
          Sync failed
        </span>
      )
  }
}

function SyncEntry({
  entry,
  expanded,
  onToggle,
}: {
  entry: SyncLogEntry
  expanded: boolean
  onToggle: () => void
}) {
  const isExpandable = entry.status !== 'success'
  const detailId = `sync-detail-${entry.id}`

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      onToggle()
    }
  }

  return (
    <div className="border-b border-gray-200 dark:border-gray-700 last:border-0">
      <div
        className={`flex items-center gap-2 py-2 ${isExpandable ? 'cursor-pointer' : ''}`}
        {...(isExpandable
          ? {
              role: 'button',
              tabIndex: 0,
              'aria-expanded': expanded,
              'aria-controls': detailId,
              'aria-label': entry.status === 'failed' ? 'Sync failed' : `${entry.accounts_fetched} synced, ${entry.accounts_failed} failed`,
              onClick: onToggle,
              onKeyDown: handleKeyDown,
            }
          : {})}
      >
        <StatusIcon status={entry.status} />
        <span className="text-sm text-gray-500 dark:text-gray-400">
          {formatTimestamp(entry.started_at)}
        </span>
        <StatusText entry={entry} />
        {isExpandable && <ChevronIcon expanded={expanded} />}
      </div>

      {isExpandable && (
        <div
          id={detailId}
          className={`overflow-hidden transition-[max-height] duration-200 ease-in-out motion-reduce:transition-none ${
            expanded ? 'max-h-96' : 'max-h-0'
          }`}
        >
          {entry.error_text && (
            <pre className="text-xs font-mono bg-gray-100 dark:bg-gray-800 rounded p-3 mt-2 mb-2 whitespace-pre-wrap break-words">
              {entry.error_text}
            </pre>
          )}
        </div>
      )}
    </div>
  )
}

export default function SyncHistory() {
  const [entries, setEntries] = useState<SyncLogEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [expandedId, setExpandedId] = useState<number | null>(null)

  useEffect(() => {
    async function load() {
      try {
        const data = await getSyncLog()
        setEntries(data.entries)
      } catch {
        // Silently fail; component shows empty state
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [])

  function handleToggle(id: number) {
    setExpandedId((prev) => (prev === id ? null : id))
  }

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl shadow-md p-6">
      <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
        Sync History
      </h2>

      {loading ? (
        <p className="text-sm text-gray-500 dark:text-gray-400">Loading sync history...</p>
      ) : entries.length === 0 ? (
        <p className="text-sm text-gray-500 dark:text-gray-400">No sync history yet.</p>
      ) : (
        <div>
          {entries.map((entry) => (
            <SyncEntry
              key={entry.id}
              entry={entry}
              expanded={expandedId === entry.id}
              onToggle={() => handleToggle(entry.id)}
            />
          ))}
        </div>
      )}
    </div>
  )
}
