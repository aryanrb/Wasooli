// Relative time for audit timelines. Good enough without a date library.
export function relativeTime(iso: string): string {
  const then = new Date(iso).getTime()
  if (Number.isNaN(then)) return iso
  const diff = Date.now() - then
  const abs = Math.abs(diff)
  const min = 60_000,
    hr = 60 * min,
    day = 24 * hr
  const fmt = (n: number, unit: string) => `${n} ${unit}${n === 1 ? '' : 's'} ${diff >= 0 ? 'ago' : 'from now'}`
  if (abs < min) return 'just now'
  if (abs < hr) return fmt(Math.round(abs / min), 'min')
  if (abs < day) return fmt(Math.round(abs / hr), 'hour')
  if (abs < 30 * day) return fmt(Math.round(abs / day), 'day')
  return new Date(iso).toLocaleDateString()
}
