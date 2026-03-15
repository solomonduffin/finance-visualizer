import { describe, expect, it } from 'vitest'
import { getAccountDisplayName } from './account'

describe('getAccountDisplayName', () => {
  it('returns display_name when set', () => {
    const result = getAccountDisplayName({
      display_name: 'My Checking',
      org_name: 'Chase',
      name: 'Chase Checking 1234',
    })
    expect(result).toBe('My Checking')
  })

  it('returns "OrgName \u2013 Name" when display_name is null and org_name exists', () => {
    const result = getAccountDisplayName({
      display_name: null,
      org_name: 'Chase',
      name: 'Checking 1234',
    })
    expect(result).toBe('Chase \u2013 Checking 1234')
  })

  it('returns name when display_name is null and org_name is empty', () => {
    const result = getAccountDisplayName({
      display_name: null,
      org_name: '',
      name: 'Checking 1234',
    })
    expect(result).toBe('Checking 1234')
  })
})
