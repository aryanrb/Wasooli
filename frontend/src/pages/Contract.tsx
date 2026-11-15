import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { ApiError, api } from '../api/client'
import type {
  AuditView,
  ContractView,
  PeriodStatus,
  Property,
  TenancyView,
  TransactionView,
} from '../api/types'
import { Layout } from '../Layout'
import { useAsync } from '../lib/useAsync'
import { formatMoney, parseMoney } from '../lib/money'
import {
  ComponentsEditor,
  emptyRows,
  rowsToComponents,
  type Row,
} from '../ComponentsEditor'
import { AuditTimeline, InviteLink, TransactionsList } from '../shared'
import { Button, Card, Chip, Empty, ErrorText, Field, Input, Modal, Select, Spinner } from '../ui'
import type { InvitationView } from '../api/types'

// Tolerate 403 (tenant hitting a property-wide route): return null instead of erroring.
async function allow403<T>(p: Promise<T>): Promise<T | null> {
  try {
    return await p
  } catch (e) {
    if (e instanceof ApiError && e.status === 403) return null
    throw e
  }
}

export function ContractPage() {
  const { id } = useParams<{ id: string }>()
  const contract = useAsync<ContractView>(() => api.get(`/contracts/${id}`), [id])
  const periods = useAsync<PeriodStatus[]>(() => api.get(`/contracts/${id}/periods`), [id])
  const tx = useAsync<TransactionView[]>(() => api.get(`/contracts/${id}/transactions`), [id])
  const audit = useAsync<AuditView[]>(() => api.get(`/contracts/${id}/audit`), [id])
  // Tenant entry point; tells us currency + owner_vacant for the tenant side.
  const tenancies = useAsync<TenancyView[]>(() => api.get('/tenancies'), [id])

  const [payPeriod, setPayPeriod] = useState<PeriodStatus | null>(null)
  const [raising, setRaising] = useState(false)
  const [ending, setEnding] = useState(false)
  const [editing, setEditing] = useState(false)
  const [invite, setInvite] = useState<null | 'tenant' | 'owner'>(null)
  const [actionError, setActionError] = useState<string | null>(null)

  const c = contract.data
  const myTenancy = (tenancies.data ?? []).find((t) => t.id === id)
  const isTenant = !!myTenancy
  // Owner can fetch the property (currency + owner_vacant); tenant gets 403 → null.
  const prop = useAsync(
    () => (c && !isTenant ? allow403(api.get<Property>(`/properties/${c.property_id}`)) : Promise.resolve(null)),
    [c?.property_id, isTenant],
  )
  const currency = myTenancy?.currency ?? prop.data?.currency ?? ''
  const ownerVacant = myTenancy?.owner_vacant ?? prop.data?.owner_vacant ?? false

  function reloadAll() {
    periods.reload()
    tx.reload()
    audit.reload()
    contract.reload()
    tenancies.reload()
  }

  async function endContract() {
    setActionError(null)
    try {
      await api.post(`/contracts/${id}/end`)
      setEnding(false)
      reloadAll()
    } catch (e) {
      setActionError(e instanceof Error ? e.message : String(e))
    }
  }

  return (
    <Layout title={c?.tenant_name ?? 'Contract'}>
      <ErrorText>{contract.error}</ErrorText>
      {contract.loading || !c ? (
        <Spinner />
      ) : (
        <>
          <Card>
            <div className="flex items-center justify-between">
              <p className="font-semibold text-lg">{c.tenant_name}</p>
              <Chip kind={c.status} />
            </div>
            <div className="flex gap-1.5 mt-1 mb-3">
              <Chip kind={c.linked ? 'linked' : 'solo'}>{c.linked ? 'linked' : 'solo'}</Chip>
              <span className="text-xs text-stone-400 self-center">
                {c.cycle} · since {c.start_date}
              </span>
            </div>
            <div className="border-t border-stone-100 pt-2 space-y-1">
              {c.current_components.map((comp, i) => (
                <div key={i} className="flex justify-between text-sm">
                  <span className="text-stone-500">{comp.label}</span>
                  <span>{formatMoney(comp.amount, currency)}</span>
                </div>
              ))}
              <div className="flex justify-between text-sm font-semibold pt-1 border-t border-stone-100">
                <span>Total / {c.cycle.replace('ly', '')}</span>
                <span>{formatMoney(c.current_total, currency)}</span>
              </div>
            </div>
            {c.status === 'active' && (
              <div className="flex flex-wrap gap-2 mt-3">
                <Button variant="secondary" onClick={() => setEditing(true)}>
                  Edit contract
                </Button>
                <Button variant="secondary" onClick={() => setRaising(true)}>
                  Raise rent
                </Button>
                {/* Owner side: invite a tenant to link. Standalone tenant: invite the owner. */}
                {!isTenant && (
                  <Button variant="secondary" onClick={() => setInvite('tenant')}>
                    Invite tenant
                  </Button>
                )}
                {isTenant && ownerVacant && (
                  <Button variant="secondary" onClick={() => setInvite('owner')}>
                    Invite owner
                  </Button>
                )}
                <Button variant="danger" onClick={() => setEnding(true)}>
                  End contract
                </Button>
              </div>
            )}
            <ErrorText>{actionError}</ErrorText>
          </Card>

          <section className="space-y-2">
            <h3 className="text-sm font-semibold text-stone-500">Periods</h3>
            <ErrorText>{periods.error}</ErrorText>
            {periods.loading ? (
              <Spinner />
            ) : (periods.data?.length ?? 0) === 0 ? (
              <Empty>No periods yet.</Empty>
            ) : (
              periods.data!.map((p) => (
                <Card key={p.period.key}>
                  <div className="flex items-center justify-between">
                    <p className="font-medium">{p.period.key}</p>
                    <Chip kind={p.status} />
                  </div>
                  <div className="grid grid-cols-3 gap-2 text-center mt-2 text-sm">
                    <div>
                      <p className="text-xs text-stone-500">Due</p>
                      <p>{formatMoney(p.due, currency)}</p>
                    </div>
                    <div>
                      <p className="text-xs text-stone-500">Paid</p>
                      <p>{formatMoney(p.paid, currency)}</p>
                    </div>
                    <div>
                      <p className="text-xs text-stone-500">Outstanding</p>
                      <p className={p.outstanding > 0 ? 'text-amber-700 font-medium' : ''}>
                        {formatMoney(p.outstanding, currency)}
                      </p>
                    </div>
                  </div>
                  {c.status === 'active' && p.status !== 'paid' && (
                    <Button
                      variant="secondary"
                      className="w-full mt-3"
                      onClick={() => setPayPeriod(p)}
                    >
                      Record payment
                    </Button>
                  )}
                </Card>
              ))
            )}
          </section>

          <section className="space-y-2">
            <h3 className="text-sm font-semibold text-stone-500">Transactions</h3>
            <ErrorText>{tx.error}</ErrorText>
            {tx.loading ? <Spinner /> : <TransactionsList items={tx.data ?? []} currency={currency} />}
          </section>

          <section className="space-y-2">
            <h3 className="text-sm font-semibold text-stone-500">Audit</h3>
            <ErrorText>{audit.error}</ErrorText>
            {audit.loading ? <Spinner /> : <AuditTimeline items={audit.data ?? []} />}
          </section>

          {payPeriod && (
            <RecordPaymentModal
              contract={c}
              period={payPeriod}
              currency={currency}
              onClose={() => setPayPeriod(null)}
              onDone={() => {
                setPayPeriod(null)
                reloadAll()
              }}
            />
          )}
          {raising && (
            <RaiseRentModal
              contractId={c.id}
              onClose={() => setRaising(false)}
              onDone={() => {
                setRaising(false)
                reloadAll()
              }}
            />
          )}
          {editing && (
            <EditContractModal
              contract={c}
              onClose={() => setEditing(false)}
              onDone={() => {
                setEditing(false)
                reloadAll()
              }}
            />
          )}
          {invite && (
            <InviteModal
              contractId={c.id}
              role={invite}
              onClose={() => setInvite(null)}
              onDone={reloadAll}
            />
          )}
          {ending && (
            <Modal title="End contract?" onClose={() => setEnding(false)}>
              <p className="text-sm text-stone-600 mb-4">
                This archives the tenancy. It cannot be undone.
              </p>
              <div className="flex gap-2">
                <Button variant="secondary" className="flex-1" onClick={() => setEnding(false)}>
                  Cancel
                </Button>
                <Button variant="danger" className="flex-1" onClick={() => void endContract()}>
                  End contract
                </Button>
              </div>
            </Modal>
          )}
        </>
      )}
    </Layout>
  )
}

function RecordPaymentModal({
  contract,
  period,
  currency,
  onClose,
  onDone,
}: {
  contract: ContractView
  period: PeriodStatus
  currency: string
  onClose: () => void
  onDone: () => void
}) {
  // Default to the outstanding amount, in major units.
  const [amount, setAmount] = useState((period.outstanding / 100).toFixed(2))
  const [paidAt, setPaidAt] = useState(new Date().toISOString().slice(0, 10))
  const [note, setNote] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    const minor = parseMoney(amount)
    if (Number.isNaN(minor) || minor <= 0) {
      setError('Enter a valid amount.')
      return
    }
    setBusy(true)
    setError(null)
    try {
      await api.post('/transactions', {
        property_id: contract.property_id,
        contract_id: contract.id,
        period_key: period.period.key,
        direction: 'income',
        amount: minor,
        note,
        paid_at: paidAt,
      })
      onDone()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal title={`Record payment · ${period.period.key}`} onClose={onClose}>
      <form onSubmit={submit} className="space-y-3">
        <p className="text-sm text-stone-500">
          Outstanding: {formatMoney(period.outstanding, currency)}
        </p>
        <Field label={`Amount (${currency})`}>
          <Input
            required
            type="number"
            step="0.01"
            min="0"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
        </Field>
        <Field label="Paid at">
          <Input required type="date" value={paidAt} onChange={(e) => setPaidAt(e.target.value)} />
        </Field>
        <Field label="Note (optional)">
          <Input value={note} onChange={(e) => setNote(e.target.value)} />
        </Field>
        <ErrorText>{error}</ErrorText>
        <Button className="w-full" disabled={busy} type="submit">
          {busy ? 'Saving…' : 'Record payment'}
        </Button>
      </form>
    </Modal>
  )
}

function RaiseRentModal({
  contractId,
  onClose,
  onDone,
}: {
  contractId: string
  onClose: () => void
  onDone: () => void
}) {
  const [effectiveFrom, setEffectiveFrom] = useState(new Date().toISOString().slice(0, 10))
  const [rows, setRows] = useState<Row[]>(emptyRows())
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    const components = rowsToComponents(rows)
    if (!components) {
      setError('Add at least one component with a label and valid amount.')
      return
    }
    setBusy(true)
    setError(null)
    try {
      await api.post(`/contracts/${contractId}/terms`, {
        effective_from: effectiveFrom,
        components,
      })
      onDone()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal title="Raise rent" onClose={onClose}>
      <form onSubmit={submit} className="space-y-3">
        <Field label="Effective from">
          <Input
            required
            type="date"
            value={effectiveFrom}
            onChange={(e) => setEffectiveFrom(e.target.value)}
          />
        </Field>
        <ComponentsEditor rows={rows} onChange={setRows} />
        <ErrorText>{error}</ErrorText>
        <Button className="w-full" disabled={busy} type="submit">
          {busy ? 'Saving…' : 'Apply new rent'}
        </Button>
      </form>
    </Modal>
  )
}

function EditContractModal({
  contract,
  onClose,
  onDone,
}: {
  contract: ContractView
  onClose: () => void
  onDone: () => void
}) {
  const [tenantName, setTenantName] = useState(contract.tenant_name)
  const [cycle, setCycle] = useState(contract.cycle)
  const [anchor, setAnchor] = useState(contract.anchor)
  const [startDate, setStartDate] = useState(contract.start_date)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await api.patch(`/contracts/${contract.id}`, {
        tenant_name: tenantName,
        cycle,
        anchor,
        start_date: startDate,
      })
      onDone()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal title="Edit contract" onClose={onClose}>
      <form onSubmit={submit} className="space-y-3">
        <Field label="Tenant name">
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
        <Field label="Anchor (optional)">
          <Input value={anchor} onChange={(e) => setAnchor(e.target.value)} />
        </Field>
        <ErrorText>{error}</ErrorText>
        <Button className="w-full" disabled={busy} type="submit">
          {busy ? 'Saving…' : 'Save changes'}
        </Button>
      </form>
    </Modal>
  )
}

function InviteModal({
  contractId,
  role,
  onClose,
  onDone,
}: {
  contractId: string
  role: 'tenant' | 'owner'
  onClose: () => void
  onDone: () => void
}) {
  const [email, setEmail] = useState('')
  const [invite, setInvite] = useState<InvitationView | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      const inv = await api.post<InvitationView>('/invitations', {
        to_email: email,
        role,
        contract_id: contractId,
      })
      setInvite(inv)
      onDone()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  const title = role === 'tenant' ? 'Invite tenant' : 'Invite owner'

  if (invite) {
    return (
      <Modal title={title} onClose={onClose}>
        <p className="text-sm text-stone-600 mb-3">Share this link so they can respond:</p>
        <InviteLink invite={invite} />
        <Button className="w-full mt-4" onClick={onClose}>
          Done
        </Button>
      </Modal>
    )
  }

  return (
    <Modal title={title} onClose={onClose}>
      <form onSubmit={submit} className="space-y-3">
        <p className="text-sm text-stone-500">
          {role === 'tenant'
            ? 'Invite an app user to link to this contract. Terms can be negotiated before it is accepted.'
            : 'Invite the real owner to take over this standalone tenancy record.'}
        </p>
        <Field label="Email">
          <Input required type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
        </Field>
        <ErrorText>{error}</ErrorText>
        <Button className="w-full" disabled={busy} type="submit">
          {busy ? 'Sending…' : 'Send invitation'}
        </Button>
      </form>
    </Modal>
  )
}
