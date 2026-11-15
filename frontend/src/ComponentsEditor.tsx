import { parseMoney } from './lib/money'
import { Button, Input, Label } from './ui'

// A row's amount is held as the raw major-unit string; converted to minor on submit.
export interface Row {
  label: string
  amount: string
}

export function emptyRows(): Row[] {
  return [{ label: '', amount: '' }]
}

// Convert editor rows to API components, dropping blanks. Returns null if any
// filled row has an invalid amount.
export function rowsToComponents(rows: Row[]): { label: string; amount: number }[] | null {
  const out: { label: string; amount: number }[] = []
  for (const r of rows) {
    if (!r.label.trim() && !r.amount.trim()) continue
    const minor = parseMoney(r.amount)
    if (!r.label.trim() || Number.isNaN(minor)) return null
    out.push({ label: r.label.trim(), amount: minor })
  }
  return out.length ? out : null
}

export function ComponentsEditor({
  rows,
  onChange,
}: {
  rows: Row[]
  onChange: (rows: Row[]) => void
}) {
  const update = (i: number, patch: Partial<Row>) =>
    onChange(rows.map((r, idx) => (idx === i ? { ...r, ...patch } : r)))
  const add = () => onChange([...rows, { label: '', amount: '' }])
  const remove = (i: number) => onChange(rows.length > 1 ? rows.filter((_, idx) => idx !== i) : rows)

  return (
    <div>
      <Label>Rent components</Label>
      <div className="space-y-2">
        {rows.map((r, i) => (
          <div key={i} className="flex gap-2">
            <Input
              placeholder="Label (e.g. Base rent)"
              value={r.label}
              onChange={(e) => update(i, { label: e.target.value })}
            />
            <Input
              className="w-28"
              type="number"
              step="0.01"
              min="0"
              placeholder="0.00"
              value={r.amount}
              onChange={(e) => update(i, { amount: e.target.value })}
            />
            <Button
              type="button"
              variant="ghost"
              onClick={() => remove(i)}
              aria-label="Remove component"
            >
              ✕
            </Button>
          </div>
        ))}
      </div>
      <Button type="button" variant="ghost" className="mt-1 px-2" onClick={add}>
        + Add component
      </Button>
    </div>
  )
}
