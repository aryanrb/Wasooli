import { useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type {
  ContractView,
  InvitationView,
  Property,
  TenancyView,
  TransactionView,
  YearlySummary,
} from '../api/types'
import { useAuth } from '../auth'
import { Layout } from '../Layout'
import { useAsync } from '../lib/useAsync'
import { formatMoney } from '../lib/money'
import {
  ComponentsEditor,
  emptyRows,
  rowsToComponents,
  type Row,
} from '../ComponentsEditor'
import { ApprovalsList, InvitationsList, type ContractCandidate } from '../sections'
import { Button, Card, Chip, ErrorText, Field, Input, Modal, Select, Spinner } from '../ui'

export function Dashboard() {
  const { user } = useAuth()
  const props = useAsync<Property[]>(() => api.get('/properties'))
  const tenancies = useAsync<TenancyView[]>(() => api.get('/tenancies'))
  const pending = useAsync<TransactionView[]>(() => api.get('/transactions/pending'))
  const invites = useAsync<InvitationView[]>(() => api.get('/invitations'))
  const analytics = useAsync<YearlySummary[]>(() => api.get('/analytics'))
  // Owner-side tenant-less active contracts across owned properties — the candidates an
  // owner can "keep" when merging a tenant's record (keep=owner). These are NOT tenancies.
  const ownerContracts = useAsync<ContractCandidate[]>(async () => {
    const ps = await api.get<Property[]>('/properties')
    const lists = await Promise.all(
      ps.map((p) =>
        api
          .get<ContractView[]>(`/properties/${p.id}/contracts`)
          .then((cs) => cs.map((c) => ({ c, pname: p.name })))
          .catch(() => [] as { c: ContractView; pname: string }[]),
      ),
    )
    return lists
      .flat()
      .filter(({ c }) => !c.tenant_user_id && c.status === 'active')
      .map(({ c, pname }) => ({ id: c.id, label: `${c.tenant_name || 'Unit'} · ${pname}` }))
  })
  const [adding, setAdding] = useState(false)

  const currencyById = new Map((props.data ?? []).map((p) => [p.id, p.currency]))
  const currencyFor = (t: TransactionView) => currencyById.get(t.property_id) ?? ''
  const thisYear = (analytics.data ?? []).find((y) => y.year === new Date().getFullYear())

  // retire = my own standalone (owner-vacant) tenancies; merge = my owner-side contracts.
  const retireCandidates: ContractCandidate[] = (tenancies.data ?? [])
    .filter((t) => t.owner_vacant && t.status === 'active')
    .map((t) => ({ id: t.id, label: `${t.tenant_name || 'Unnamed'} · ${t.property_name}` }))
  const mergeCandidates: ContractCandidate[] = ownerContracts.data ?? []

  function reloadAll() {
    invites.reload()
    props.reload()
    tenancies.reload()
    ownerContracts.reload()
  }

  return (
    <Layout title="Wasooli">
      <h2 className="text-xl font-semibold">
        Hi{user?.name ? `, ${user.name.split(' ')[0]}` : ''} 👋
      </h2>

      {(pending.data?.length ?? 0) > 0 && (
        <section className="space-y-2">
          <h3 className="text-sm font-semibold text-stone-500">Approvals</h3>
          <ApprovalsList
            items={pending.data!}
            currencyFor={currencyFor}
            onChanged={() => pending.reload()}
          />
        </section>
      )}

      {(invites.data?.length ?? 0) > 0 && (
        <section className="space-y-2">
          <h3 className="text-sm font-semibold text-stone-500">Invitations</h3>
          <InvitationsList
            items={invites.data!}
            mergeCandidates={mergeCandidates}
            retireCandidates={retireCandidates}
            onChanged={reloadAll}
          />
        </section>
      )}

      {thisYear && (
        <Card>
          <h3 className="text-sm font-semibold text-stone-500 mb-2">{thisYear.year} summary</h3>
          <div className="grid grid-cols-3 gap-2 text-center">
            <SummaryStat label="Income" value={thisYear.income} currency={firstCurrency(props.data)} tone="text-green-700" />
            <SummaryStat label="Expense" value={thisYear.expense} currency={firstCurrency(props.data)} tone="text-red-700" />
            <SummaryStat label="Net" value={thisYear.net} currency={firstCurrency(props.data)} tone="text-stone-900" />
          </div>
        </Card>
      )}

      <section className="space-y-2">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold text-stone-500">Properties</h3>
          <Button variant="secondary" onClick={() => setAdding(true)}>
            + Add property
          </Button>
        </div>
        <ErrorText>{props.error}</ErrorText>
        {props.loading ? (
          <Spinner />
        ) : (props.data?.length ?? 0) === 0 ? (
          <Card>
            <p className="text-sm text-stone-500">No properties yet. Add one to get started.</p>
          </Card>
        ) : (
          props.data!.map((p) => (
            <Link key={p.id} to={`/properties/${p.id}`}>
              <Card className="hover:border-stone-300">
                <p className="font-medium">{p.name}</p>
                <p className="text-xs text-stone-500">{p.address || 'No address'}</p>
                <p className="text-xs text-stone-400 mt-1">{p.currency}</p>
              </Card>
            </Link>
          ))
        )}
      </section>

      <section className="space-y-2">
        <h3 className="text-sm font-semibold text-stone-500">My tenancies</h3>
        <ErrorText>{tenancies.error}</ErrorText>
        {tenancies.loading ? (
          <Spinner />
        ) : (tenancies.data?.length ?? 0) === 0 ? (
          <Card>
            <p className="text-sm text-stone-500">No tenancies.</p>
          </Card>
        ) : (
          tenancies.data!.map((t) => (
            <Link key={t.id} to={`/contracts/${t.id}`}>
              <Card className="hover:border-stone-300">
                <div className="flex items-center justify-between">
                  <p className="font-medium">{t.property_name}</p>
                  <p className="font-semibold">{formatMoney(t.current_total, t.currency)}</p>
                </div>
                <div className="flex gap-1.5 mt-1">
                  <Chip kind={t.status} />
                  {t.owner_vacant && <Chip kind="solo">standalone (no owner linked)</Chip>}
                  <span className="text-xs text-stone-400 self-center">{t.cycle}</span>
                </div>
              </Card>
            </Link>
          ))
        )}
      </section>

      {adding && (
        <AddPropertyModal
          onClose={() => setAdding(false)}
          onCreated={() => {
            setAdding(false)
            props.reload()
            tenancies.reload()
          }}
        />
      )}
    </Layout>
  )
}

function firstCurrency(props: Property[] | null): string {
  return props?.[0]?.currency ?? ''
}

function SummaryStat({
  label,
  value,
  currency,
  tone,
}: {
  label: string
  value: number
  currency: string
  tone: string
}) {
  return (
    <div>
      <p className="text-xs text-stone-500">{label}</p>
      <p className={`text-sm font-semibold ${tone}`}>{formatMoney(value, currency)}</p>
    </div>
  )
}

function AddPropertyModal({ onClose, onCreated }: { onClose: () => void; onCreated: () => void }) {
  const [role, setRole] = useState<'owner' | 'tenant'>('owner')
  const [name, setName] = useState('')
  const [address, setAddress] = useState('')
  const [currency, setCurrency] = useState('PKR')
  // Tenant-path contract fields:
  const [tenantName, setTenantName] = useState('')
  const [startDate, setStartDate] = useState(new Date().toISOString().slice(0, 10))
  const [cycle, setCycle] = useState<'weekly' | 'monthly' | 'yearly'>('monthly')
  const [rows, setRows] = useState<Row[]>(emptyRows())
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    const body: Record<string, unknown> = {
      role,
      name,
      address,
      currency: currency.trim().toUpperCase(),
    }
    if (role === 'tenant') {
      const components = rowsToComponents(rows)
      if (!components) {
        setError('Add at least one rent component with a label and valid amount.')
        return
      }
      body.tenant_name = tenantName
      body.start_date = startDate
      body.cycle = cycle
      body.components = components
    }
    setBusy(true)
    setError(null)
    try {
      await api.post('/properties', body)
      onCreated()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal title="Add property" onClose={onClose}>
      <form onSubmit={submit} className="space-y-3">
        <div className="flex rounded-xl border border-stone-200 p-1 text-sm">
          <button
            type="button"
            className={`flex-1 rounded-lg py-1.5 ${role === 'owner' ? 'bg-stone-900 text-white' : 'text-stone-600'}`}
            onClick={() => setRole('owner')}
          >
            I'm the owner
          </button>
          <button
            type="button"
            className={`flex-1 rounded-lg py-1.5 ${role === 'tenant' ? 'bg-stone-900 text-white' : 'text-stone-600'}`}
            onClick={() => setRole('tenant')}
          >
            I'm the tenant
          </button>
        </div>

        <Field label="Name">
          <Input required value={name} onChange={(e) => setName(e.target.value)} />
        </Field>
        <Field label="Address">
          <Input value={address} onChange={(e) => setAddress(e.target.value)} />
        </Field>
        <Field label="Currency code">
          <Input required maxLength={3} value={currency} onChange={(e) => setCurrency(e.target.value)} />
        </Field>

        {role === 'tenant' && (
          <>
            <div className="border-t border-stone-100 pt-2 text-xs font-medium text-stone-500">
              Your tenancy terms
            </div>
            <Field label="Tenant name (optional — defaults to you)">
              <Input value={tenantName} onChange={(e) => setTenantName(e.target.value)} />
            </Field>
            <Field label="Start date">
              <Input required type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} />
            </Field>
            <Field label="Cycle">
              <Select value={cycle} onChange={(e) => setCycle(e.target.value as typeof cycle)}>
                <option value="weekly">Weekly</option>
                <option value="monthly">Monthly</option>
                <option value="yearly">Yearly</option>
              </Select>
            </Field>
            <ComponentsEditor rows={rows} onChange={setRows} />
          </>
        )}

        <ErrorText>{error}</ErrorText>
        <Button className="w-full" disabled={busy} type="submit">
          {busy ? 'Creating…' : 'Create'}
        </Button>
      </form>
    </Modal>
  )
}
