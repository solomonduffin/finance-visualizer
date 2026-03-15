/**
 * Formats a string balance value as a USD currency string.
 * e.g., "4230.50" -> "$4,230.50"
 * Handles negative values for credit cards: "-500.25" -> "-$500.25"
 */
export function formatCurrency(value: string): string {
  return Number(value).toLocaleString('en-US', { style: 'currency', currency: 'USD' })
}
