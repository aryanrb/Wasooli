import { api } from '../api/client'
import type { Property, TransactionView } from '../api/types'
import { Layout } from '../Layout'
import { useAsync } from '../lib/useAsync'
import { ApprovalsList } from '../sections'
import { ErrorText, Spinner } from '../ui'

export function Approvals() {
  const pending = useAsync<TransactionView[]>(() => api.get('/transactions/pending'))
  const props = useAsync<Property[]>(() => api.get('/properties'))

  const currencyById = new Map((props.data ?? []).map((p) => [p.id, p.currency]))
  const currencyFor = (t: TransactionView) => currencyById.get(t.property_id) ?? ''

  return (
    <Layout title="Approvals">
      <h2 className="text-xl font-semibold">Approval inbox</h2>
      <p className="text-sm text-stone-500">Payments others recorded on your contracts.</p>
      <ErrorText>{pending.error}</ErrorText>
      {pending.loading ? (
        <Spinner />
      ) : (
        <ApprovalsList
          items={pending.data ?? []}
          currencyFor={currencyFor}
          onChanged={() => pending.reload()}
        />
      )}
    </Layout>
  )
}
