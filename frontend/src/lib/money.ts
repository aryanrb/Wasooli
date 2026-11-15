// Money is always integer minor units. Format divides by 100; parse multiplies
// by 100 and rounds. Single source of truth for both directions.

export function formatMoney(minor: number, currency: string): string {
  const major = (minor / 100).toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })
  return `${currency} ${major}`
}

// Parse a user-entered major-unit string into integer minor units.
// Returns NaN for invalid input so callers can validate.
export function parseMoney(input: string): number {
  const n = Number(input)
  if (!Number.isFinite(n)) return NaN
  return Math.round(n * 100)
}
