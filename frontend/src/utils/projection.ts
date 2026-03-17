export interface AccountProjection {
  id: string
  currentBalance: number
  apy: number // annual percentage (e.g., 5.0 for 5%)
  compound: boolean
  included: boolean
  allocation: number // percentage of income allocated (0-100), for accounts without holdings
  hasHoldings: boolean // true if this account's projection comes from holdings, not account-level
}

export interface HoldingProjection {
  id: string
  accountId: string
  currentValue: number
  apy: number
  compound: boolean
  included: boolean
  allocation: number // percentage of income allocated (0-100)
}

export interface IncomeSettings {
  enabled: boolean
  annualIncome: number
  monthlySavingsPct: number // percentage (e.g., 20 for 20%)
}

export interface ProjectionPoint {
  date: string // YYYY-MM-DD
  value: number // projected net worth
}

/**
 * Projects a single balance forward by a given number of months.
 * Supports compound (month-by-month) and simple (linear) interest,
 * with optional monthly contributions.
 */
export function projectBalance(
  principal: number,
  apy: number,
  compound: boolean,
  months: number,
  monthlyContribution: number = 0,
): number {
  const monthlyRate = apy / 100 / 12

  if (compound) {
    let balance = principal
    for (let m = 0; m < months; m++) {
      balance = balance * (1 + monthlyRate) + monthlyContribution
    }
    return balance
  } else {
    // Simple: principal grows linearly, contributions accumulate linearly
    return principal + principal * monthlyRate * months + monthlyContribution * months
  }
}

/**
 * Calculates a portfolio-level projection across all included accounts and holdings.
 *
 * Key behaviors:
 * - Accounts with hasHoldings=true are SKIPPED (their value comes from the holdings loop)
 * - Income contribution is distributed by allocation percentage
 * - If allocation sum is not ~100%, income contribution is suppressed (growth-only)
 * - Returns totalMonths+1 points (month 0 through month N)
 */
export function calculateProjection(
  accounts: AccountProjection[],
  holdings: HoldingProjection[],
  income: IncomeSettings,
  horizonYears: number,
): ProjectionPoint[] {
  const totalMonths = horizonYears * 12
  const monthlySavings = income.enabled
    ? (income.annualIncome / 12) * (income.monthlySavingsPct / 100)
    : 0

  // Validate allocation sum = 100% when income is enabled
  // Sum allocations from: accounts without holdings + all holdings
  const allocationSum = [
    ...accounts.filter((a) => a.included && !a.hasHoldings).map((a) => a.allocation),
    ...holdings.filter((h) => h.included).map((h) => h.allocation),
  ].reduce((sum, pct) => sum + pct, 0)

  const useIncome = income.enabled && monthlySavings > 0 && Math.abs(allocationSum - 100) < 0.01

  const now = new Date()
  const points: ProjectionPoint[] = []

  for (let m = 0; m <= totalMonths; m++) {
    const date = new Date(now.getFullYear(), now.getMonth() + m, now.getDate())
    let total = 0

    // Accounts WITHOUT holdings (project at account level)
    for (const acct of accounts) {
      if (!acct.included || acct.hasHoldings) continue
      const contribution = useIncome ? monthlySavings * (acct.allocation / 100) : 0
      total += projectBalance(acct.currentBalance, acct.apy, acct.compound, m, contribution)
    }

    // Holdings (for investment accounts WITH holdings -- project per-holding)
    for (const h of holdings) {
      if (!h.included) continue
      const contribution = useIncome ? monthlySavings * (h.allocation / 100) : 0
      total += projectBalance(h.currentValue, h.apy, h.compound, m, contribution)
    }

    points.push({
      date: date.toISOString().split('T')[0],
      value: Math.round(total * 100) / 100, // round to cents
    })
  }

  return points
}
