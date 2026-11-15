import { api } from '../api/client'
import type { Property, YearlySummary } from '../api/types'
import { Layout } from '../Layout'
import { useAsync } from '../lib/useAsync'
import { formatMoney } from '../lib/money'
import { Card, Empty, ErrorText, Spinner } from '../ui'

export function Analytics() {
  const data = useAsync<YearlySummary[]>(() => api.get('/analytics'))
  const props = useAsync<Property[]>(() => api.get('/properties'))
  const currency = props.data?.[0]?.currency ?? ''

  const years = data.data ?? []
  // Scale bars to the largest income/expense across all years.
  const max = Math.max(1, ...years.flatMap((y) => [y.income, y.expense]))

  return (
    <Layout title="Analytics">
      <h2 className="text-xl font-semibold">Yearly summary</h2>
      <ErrorText>{data.error}</ErrorText>
      {data.loading ? (
        <Spinner />
      ) : years.length === 0 ? (
        <Empty>No data yet.</Empty>
      ) : (
        <>
          {years.map((y) => (
            <Card key={y.year} className="space-y-2">
              <div className="flex items-center justify-between">
                <h3 className="font-semibold">{y.year}</h3>
                <span className={`text-sm font-medium ${y.net >= 0 ? 'text-green-700' : 'text-red-700'}`}>
                  net {formatMoney(y.net, currency)}
                </span>
              </div>
              <Bar label="Income" value={y.income} max={max} currency={currency} color="bg-green-500" />
              <Bar label="Expense" value={y.expense} max={max} currency={currency} color="bg-red-500" />
            </Card>
          ))}

          <Card>
            <h3 className="text-sm font-semibold text-stone-500 mb-2">Table</h3>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left text-stone-500">
                    <th className="py-1 pr-2">Year</th>
                    <th className="py-1 pr-2 text-right">Income</th>
                    <th className="py-1 pr-2 text-right">Expense</th>
                    <th className="py-1 text-right">Net</th>
                  </tr>
                </thead>
                <tbody>
                  {years.map((y) => (
                    <tr key={y.year} className="border-t border-stone-100">
                      <td className="py-1 pr-2">{y.year}</td>
                      <td className="py-1 pr-2 text-right">{formatMoney(y.income, currency)}</td>
                      <td className="py-1 pr-2 text-right">{formatMoney(y.expense, currency)}</td>
                      <td className="py-1 text-right">{formatMoney(y.net, currency)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </Card>
        </>
      )}
    </Layout>
  )
}

function Bar({
  label,
  value,
  max,
  currency,
  color,
}: {
  label: string
  value: number
  max: number
  currency: string
  color: string
}) {
  const pct = Math.round((value / max) * 100)
  return (
    <div>
      <div className="flex justify-between text-xs text-stone-500 mb-0.5">
        <span>{label}</span>
        <span>{formatMoney(value, currency)}</span>
      </div>
      <div className="h-3 rounded-full bg-stone-100 overflow-hidden">
        <div className={`h-full rounded-full ${color}`} style={{ width: `${pct}%` }} />
      </div>
    </div>
  )
}
