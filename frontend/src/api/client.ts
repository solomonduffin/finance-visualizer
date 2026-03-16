/**
 * Typed fetch wrapper for API calls.
 * All requests use credentials: 'include' for HttpOnly cookie-based auth.
 */

export interface LoginSuccess {
  ok: true
}

export interface LoginError {
  error: string
}

export type LoginResult = LoginSuccess | LoginError

/**
 * POST /api/auth/login
 * Returns {ok: true} on success, {error: string} on failure.
 * Throws on network error.
 */
export async function login(password: string): Promise<LoginResult> {
  const res = await fetch('/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ password }),
  })

  if (res.status === 200) {
    return { ok: true }
  }

  if (res.status === 429) {
    return { error: 'rate_limited' }
  }

  const body = await res.json().catch(() => ({ error: 'invalid password' }))
  return { error: body.error ?? 'invalid password' }
}

/**
 * GET /api/health
 * Returns true if authenticated (200), false otherwise (401).
 */
export async function checkAuth(): Promise<boolean> {
  try {
    const res = await fetch('/api/health', {
      credentials: 'include',
    })
    return res.status === 200
  } catch {
    return false
  }
}

export interface SettingsResponse {
  configured: boolean
  last_sync_at: string | null
  last_sync_status: string | null
  growth_badge_enabled: boolean
}

/**
 * GET /api/settings
 * Returns current SimpleFIN configuration and last sync status.
 */
export async function getSettings(): Promise<SettingsResponse> {
  const res = await fetch('/api/settings', { credentials: 'include' })
  return res.json()
}

/**
 * POST /api/settings
 * Saves the SimpleFIN access URL. Triggers an immediate first sync if new URL.
 */
export async function saveSettings(accessUrl: string): Promise<{ ok: boolean }> {
  const res = await fetch('/api/settings', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ access_url: accessUrl }),
  })
  return res.json()
}

export interface SyncResponse {
  ok: boolean
  restored?: string[]
}

/**
 * POST /api/sync/now
 * Triggers an on-demand sync. Returns restored account names if any were unhidden.
 */
export async function triggerSync(): Promise<SyncResponse> {
  const res = await fetch('/api/sync/now', {
    method: 'POST',
    credentials: 'include',
  })
  return res.json()
}

// ─── Dashboard Endpoints ────────────────────────────────────────────────────

export interface SummaryResponse {
  liquid: string
  savings: string
  investments: string
  last_synced_at: string | null
}

export interface AccountItem {
  id: string
  name: string              // effective name (COALESCE from backend)
  original_name: string     // raw SimpleFIN name
  balance: string
  account_type: string      // effective type (after override)
  org_name: string
  display_name: string | null
  hidden_at: string | null
  account_type_override: string | null
}

export interface AccountsResponse {
  liquid: AccountItem[]
  savings: AccountItem[]
  investments: AccountItem[]
  other: AccountItem[]
}

export interface UpdateAccountRequest {
  display_name?: string | null
  hidden?: boolean
  account_type_override?: string | null
}

/**
 * PATCH /api/accounts/:id
 * Updates account metadata (display name, hidden state, type override).
 */
export async function updateAccount(id: string, data: UpdateAccountRequest): Promise<AccountItem> {
  const res = await fetch(`/api/accounts/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(`Failed to update account: ${res.status}`)
  return res.json()
}

export interface HistoryPoint {
  date: string
  balance: string
}

export interface BalanceHistoryResponse {
  liquid: HistoryPoint[]
  savings: HistoryPoint[]
  investments: HistoryPoint[]
}

/**
 * GET /api/summary
 * Returns total balances by panel category.
 */
export async function getSummary(): Promise<SummaryResponse> {
  const res = await fetch('/api/summary', { credentials: 'include' })
  return res.json()
}

/**
 * GET /api/accounts
 * Returns accounts grouped by panel category.
 * Pass includeHidden=true to include hidden accounts (for Settings page).
 */
export async function getAccounts(includeHidden = false): Promise<AccountsResponse> {
  const url = includeHidden ? '/api/accounts?include_hidden=true' : '/api/accounts'
  const res = await fetch(url, { credentials: 'include' })
  return res.json()
}

/**
 * GET /api/balance-history?days=N
 * Returns balance history for the past N days (default 30).
 */
export async function getBalanceHistory(days?: number): Promise<BalanceHistoryResponse> {
  const url = days !== undefined ? `/api/balance-history?days=${days}` : '/api/balance-history'
  const res = await fetch(url, { credentials: 'include' })
  return res.json()
}

// ---- Sync Log ----

export interface SyncLogEntry {
  id: number
  started_at: string
  finished_at: string | null
  accounts_fetched: number
  accounts_failed: number
  error_text: string | null
  status: 'success' | 'partial' | 'failed'
}

export interface SyncLogResponse {
  entries: SyncLogEntry[]
}

/**
 * GET /api/sync-log
 * Returns the last 7 sync log entries with derived status and sanitized error text.
 */
export async function getSyncLog(): Promise<SyncLogResponse> {
  const res = await fetch('/api/sync-log', { credentials: 'include' })
  return res.json()
}

// ---- Growth ----

export interface GrowthData {
  current_total: string
  prior_total: string
  dollar_change: string
  pct_change: string
}

export interface GrowthResponse {
  liquid: GrowthData | null
  savings: GrowthData | null
  investments: GrowthData | null
  growth_badge_enabled: boolean
}

/**
 * GET /api/growth
 * Returns per-panel 30-day growth data with decimal arithmetic.
 */
export async function getGrowth(): Promise<GrowthResponse> {
  const res = await fetch('/api/growth', { credentials: 'include' })
  return res.json()
}

// ---- Settings Toggle ----

/**
 * PUT /api/settings/growth-badge
 * Persists the growth badge enabled/disabled toggle.
 */
export async function saveGrowthBadgeSetting(value: boolean): Promise<{ ok: boolean }> {
  const res = await fetch('/api/settings/growth-badge', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ value: String(value) }),
  })
  if (!res.ok) throw new Error(`Failed to save setting: ${res.status}`)
  return res.json()
}
