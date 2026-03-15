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

/**
 * POST /api/sync/now
 * Triggers an on-demand background sync.
 */
export async function triggerSync(): Promise<{ ok: boolean }> {
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
  name: string
  balance: string
  account_type: string
}

export interface AccountsResponse {
  liquid: AccountItem[]
  savings: AccountItem[]
  investments: AccountItem[]
  other: AccountItem[]
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
 */
export async function getAccounts(): Promise<AccountsResponse> {
  const res = await fetch('/api/accounts', { credentials: 'include' })
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
