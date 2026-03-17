import { describe, expect, it } from 'vitest'
import {
  projectBalance,
  calculateProjection,
  type AccountProjection,
  type HoldingProjection,
  type IncomeSettings,
} from './projection'

describe('projectBalance', () => {
  it('compound interest: $10000 at 12% APY for 12 months = ~$11268.25', () => {
    const result = projectBalance(10000, 12, true, 12, 0)
    expect(result).toBeCloseTo(11268.25, 2)
  })

  it('simple interest: $10000 at 12% APY for 12 months = $11200.00', () => {
    const result = projectBalance(10000, 12, false, 12, 0)
    expect(result).toBeCloseTo(11200.0, 2)
  })

  it('compound with contributions: $10000 at 5% APY, 12 months, $500/month', () => {
    // Manual: month-by-month compound with monthly contribution
    // monthlyRate = 0.05/12 = 0.00416667
    // After 12 months: principal grows + contributions compound
    const result = projectBalance(10000, 5, true, 12, 500)
    expect(result).toBeCloseTo(16386, 0)
  })

  it('0% APY returns principal unchanged after any number of months', () => {
    expect(projectBalance(10000, 0, true, 12, 0)).toBeCloseTo(10000, 2)
    expect(projectBalance(10000, 0, false, 12, 0)).toBeCloseTo(10000, 2)
    expect(projectBalance(10000, 0, true, 120, 0)).toBeCloseTo(10000, 2)
  })

  it('0% APY with monthly contribution correctly adds contributions', () => {
    // 12 months * $500 = $6000 + $10000 = $16000
    expect(projectBalance(10000, 0, true, 12, 500)).toBeCloseTo(16000, 2)
    expect(projectBalance(10000, 0, false, 12, 500)).toBeCloseTo(16000, 2)
  })
})

describe('calculateProjection', () => {
  const baseIncome: IncomeSettings = {
    enabled: false,
    annualIncome: 0,
    monthlySavingsPct: 0,
  }

  it('sums projected values from 2 included accounts per month', () => {
    const accounts: AccountProjection[] = [
      {
        id: 'a1',
        currentBalance: 10000,
        apy: 12,
        compound: true,
        included: true,
        allocation: 50,
        hasHoldings: false,
      },
      {
        id: 'a2',
        currentBalance: 5000,
        apy: 6,
        compound: true,
        included: true,
        allocation: 50,
        hasHoldings: false,
      },
    ]

    const points = calculateProjection(accounts, [], baseIncome, 1)
    // Month 0 should be sum of current balances
    expect(points[0].value).toBeCloseTo(15000, 2)
    // Month 12 should be sum of projected balances
    const a1_12 = projectBalance(10000, 12, true, 12, 0)
    const a2_12 = projectBalance(5000, 6, true, 12, 0)
    expect(points[12].value).toBeCloseTo(a1_12 + a2_12, 2)
  })

  it('excludes accounts where included=false', () => {
    const accounts: AccountProjection[] = [
      {
        id: 'a1',
        currentBalance: 10000,
        apy: 12,
        compound: true,
        included: true,
        allocation: 100,
        hasHoldings: false,
      },
      {
        id: 'a2',
        currentBalance: 5000,
        apy: 6,
        compound: true,
        included: false,
        allocation: 0,
        hasHoldings: false,
      },
    ]

    const points = calculateProjection(accounts, [], baseIncome, 1)
    // Month 0: only a1
    expect(points[0].value).toBeCloseTo(10000, 2)
  })

  it('distributes monthly savings based on allocation percentages when income enabled', () => {
    const accounts: AccountProjection[] = [
      {
        id: 'a1',
        currentBalance: 10000,
        apy: 0,
        compound: true,
        included: true,
        allocation: 60,
        hasHoldings: false,
      },
      {
        id: 'a2',
        currentBalance: 5000,
        apy: 0,
        compound: true,
        included: true,
        allocation: 40,
        hasHoldings: false,
      },
    ]

    const income: IncomeSettings = {
      enabled: true,
      annualIncome: 120000,
      monthlySavingsPct: 20,
    }

    // monthlySavings = 120000/12 * 0.2 = $2000/month
    // a1 gets 60% = $1200/month, a2 gets 40% = $800/month
    const points = calculateProjection(accounts, [], income, 1)
    // Month 1: a1 = 10000 + 1200 = 11200, a2 = 5000 + 800 = 5800, total = 17000
    expect(points[1].value).toBeCloseTo(17000, 2)
  })

  it('projects per-holding (not account-level) for accounts with holdings', () => {
    const accounts: AccountProjection[] = [
      {
        id: 'invest',
        currentBalance: 20000, // should be skipped (hasHoldings=true)
        apy: 5,
        compound: true,
        included: true,
        allocation: 0,
        hasHoldings: true,
      },
    ]

    const holdings: HoldingProjection[] = [
      {
        id: 'h1',
        accountId: 'invest',
        currentValue: 12000,
        apy: 8,
        compound: true,
        included: true,
        allocation: 50,
      },
      {
        id: 'h2',
        accountId: 'invest',
        currentValue: 8000,
        apy: 4,
        compound: true,
        included: true,
        allocation: 50,
      },
    ]

    const points = calculateProjection(accounts, holdings, baseIncome, 1)
    // Month 0: h1 + h2 = 12000 + 8000 = 20000 (NOT account's 20000 double-counted)
    expect(points[0].value).toBeCloseTo(20000, 2)

    // Month 12: projected from holdings, not from account
    const h1_12 = projectBalance(12000, 8, true, 12, 0)
    const h2_12 = projectBalance(8000, 4, true, 12, 0)
    expect(points[12].value).toBeCloseTo(h1_12 + h2_12, 2)
  })

  it('returns totalMonths+1 points (month 0 through month N)', () => {
    const accounts: AccountProjection[] = [
      {
        id: 'a1',
        currentBalance: 1000,
        apy: 5,
        compound: true,
        included: true,
        allocation: 100,
        hasHoldings: false,
      },
    ]

    const points = calculateProjection(accounts, [], baseIncome, 5)
    expect(points).toHaveLength(5 * 12 + 1)
  })

  it('with income disabled but accounts having APY still projects growth-only', () => {
    const accounts: AccountProjection[] = [
      {
        id: 'a1',
        currentBalance: 10000,
        apy: 10,
        compound: true,
        included: true,
        allocation: 100,
        hasHoldings: false,
      },
    ]

    const income: IncomeSettings = {
      enabled: false,
      annualIncome: 120000,
      monthlySavingsPct: 20,
    }

    const points = calculateProjection(accounts, [], income, 1)
    // Should only have growth, no contribution
    const expected = projectBalance(10000, 10, true, 12, 0)
    expect(points[12].value).toBeCloseTo(expected, 2)
  })

  it('suppresses income contribution when allocation sum does not equal 100', () => {
    const accounts: AccountProjection[] = [
      {
        id: 'a1',
        currentBalance: 10000,
        apy: 5,
        compound: true,
        included: true,
        allocation: 30, // only 30%, not 100%
        hasHoldings: false,
      },
    ]

    const income: IncomeSettings = {
      enabled: true,
      annualIncome: 120000,
      monthlySavingsPct: 20,
    }

    const points = calculateProjection(accounts, [], income, 1)
    // Should be growth-only (income suppressed because allocation != 100)
    const expected = projectBalance(10000, 5, true, 12, 0)
    expect(points[12].value).toBeCloseTo(expected, 2)
  })
})
