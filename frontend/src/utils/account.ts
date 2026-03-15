import type { AccountItem } from '../api/client'

/**
 * Returns the display name for an account.
 * - If display_name is set, returns it (no org prefix).
 * - If display_name is null and org_name exists, returns "OrgName - Name".
 * - If display_name is null and org_name is empty, returns name.
 */
export function getAccountDisplayName(
  _account: Pick<AccountItem, 'display_name' | 'org_name' | 'name'>
): string {
  // Stub: not implemented yet
  return ''
}
