import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api } from '../api/client'
import type {
  AuditView,
  ContractView,
  InvitationView,
  Property,
  TransactionView,
} from '../api/types'
import { Layout } from '../Layout'
import { useAsync } from '../lib/useAsync'
import { formatMoney } from '../lib/money'
import {
  ComponentsEditor,
  emptyRows,
  rowsToComponents,
  type Row,
} from '../ComponentsEditor'
import { AuditTimeline, InviteLink, TransactionsList } from '../shared'
import { Button, Card, Chip, Empty, ErrorText, Field, Input, Modal, Select, Spinner } from '../ui'

export function PropertyPage() {
  const { id } = useParams<{ id: string }>()
  const prop = useAsync<Property>(() => api.get(`/properties/${id}`), [id])
  const contracts = useAsync<ContractView[]>(() => api.get(`/properties/${id}/contracts`), [id])
  const [modal, setModal] = useState<null | 'contract' | 'viewer'>(null)
  const [showTx, setShowTx] = useState(false)
  const [showAudit, setShowAudit] = useState(false)

  const currency = prop.data?.currency ?? ''

  return (
    <Layout title={prop.data?.name ?? 'Property'}>
      <ErrorText>{prop.error}</ErrorText>
      {prop.loading ? (
        <Spinner />
      ) : prop.data ? (
        <>
          <Card>
            <p className="font-semibold text-lg">{prop.data.name}</p>
            <p className="text-sm text-stone-500">{prop.data.address || 'No address'}</p>
            <p className="text-xs text-stone-400 mt-1">{prop.data.currency}</p>
            <div className="flex gap-2 mt-3">
              <Button variant="secondary" onClick={() => setModal('viewer')}>
                Add viewer
              </Button>
            </div>
          </Card>

          <section className="space-y-2">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-semibold text-stone-500">Contracts</h3>
              <Button variant="secondary" onClick={() => setModal('contract')}>
                + New contract
              </Button>
            </div>
            <ErrorText>{contracts.error}</ErrorText>
            {contracts.loading ? (
              <Spinner />
            ) : (contracts.data?.length ?? 0) === 0 ? (
              <Empty>No contracts yet.</Empty>
            ) : (
              contracts.data!.map((c) => (
                <Link key={c.id} to={`/contracts/${c.id}`}>
                  <Card className="hover:border-stone-300">
                    <div className="flex items-center justify-between">
                      <p className="font-medium">{c.tenant_name}</p>
                      <p className="font-semibold">{formatMoney(c.current_total, currency)}</p>
                    </div>
                    <div className="flex gap-1.5 mt-1">
                      <Chip kind={c.status} />
                      <Chip kind={c.linked ? 'linked' : 'solo'}>{c.linked ? 'linked' : 'solo'}</Chip>
                      <span className="text-xs text-stone-400 self-center">{c.cycle}</span>
                    </div>
                  </Card>
                </Link>
              ))
            )}
          </section>

          <div className="flex gap-2">
            <Button variant="secondary" onClick={() => setShowTx((v) => !v)}>
              {showTx ? 'Hide' : 'Show'} transactions
            </Button>
            <Button variant="secondary" onClick={() => setShowAudit((v) => !v)}>
              {showAudit ? 'Hide' : 'Show'} audit
            </Button>
          </div>

          {showTx && <PropertyTransactions id={id!} currency={currency} />}
          {showAudit && <PropertyAudit id={id!} />}

          {modal === 'contract' && (
            <NewContractModal
              propertyId={id!}
              onClose={() => setModal(null)}
              onCreated={() => {
                setModal(null)
                contracts.reload()
              }}
            />
          )}
          {modal === 'viewer' && (
            <AddViewerModal propertyId={id!} onClose={() => setModal(null)} />
          )}
        </>
      ) : null}
    </Layout>
  )
}

function PropertyTransactions({ id, currency }: { id: string; currency: string }) {
  const tx = useAsync<TransactionView[]>(() => api.get(`/properties/${id}/transactions`), [id])
  return (
    <section className="space-y-2">
      <h3 className="text-sm font-semibold text-stone-500">Transactions</h3>
      <ErrorText>{tx.error}</ErrorText>
      {tx.loading ? <Spinner /> : <TransactionsList items={tx.data ?? []} currency={currency} />}
    </section>
  )
}

function PropertyAudit({ id }: { id: string }) {
  const audit = useAsync<AuditView[]>(() => api.get(`/properties/${id}/audit`), [id])
  return (
    <section className="space-y-2">
      <h3 className="text-sm font-semibold text-stone-500">Audit</h3>
      <ErrorText>{audit.error}</ErrorText>
      {audit.loading ? <Spinner /> : <AuditTimeline items={audit.data ?? []} />}
    </section>
  )
}

function AddViewerModal({ propertyId, onClose }: { propertyId: string; onClose: () => void }) {
  const [email, setEmail] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [done, setDone] = useState<string | null>(null)
  const [invite, setInvite] = useState<InvitationView | null>(null)
  const [busy, setBusy] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    setDone(null)
    setInvite(null)
    try {
      // Registered users are added directly; otherwise fall back to a viewer invitation.
      try {
        const res = await api.post<{ status: string }>(`/properties/${propertyId}/viewers`, { email })
        setDone(`Viewer added (${res.status}).`)
      } catch {
        const inv = await api.post<InvitationView>('/invitations', {
          to_email: email,
          role: 'viewer',
          property_id: propertyId,
        })
        setInvite(inv)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal title="Add viewer" onClose={onClose}>
      <form onSubmit={submit} className="space-y-3">
        <Field label="Email">
          <Input required type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
        </Field>
        <p className="text-xs text-stone-400">
          If they already have an account they get read access immediately; otherwise an invitation
          link is generated.
        </p>
        <ErrorText>{error}</ErrorText>
        {done && <p className="text-sm text-green-700">{done}</p>}
        {invite && <InviteLink invite={invite} />}
        <Button className="w-full" disabled={busy} type="submit">
          {busy ? 'Adding…' : 'Add viewer'}
        </Button>
      </form>
    </Modal>
  )
}

function NewContractModal({
  propertyId,
  onClose,
  onCreated,
}: {
  propertyId: string
  onClose: () => void
  onCreated: () => void
}) {
  const [tenantName, setTenantName] = useState('')
  const [tenantEmail, setTenantEmail] = useState('')
  const [startDate, setStartDate] = useState(new Date().toISOString().slice(0, 10))
  const [cycle, setCycle] = useState<'weekly' | 'monthly' | 'yearly'>('monthly')
  const [rows, setRows] = useState<Row[]>(emptyRows())
  const [error, setError] = useState<string | null>(null)
  const [invite, setInvite] = useState<InvitationView | null>(null)
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
      const body: Record<string, unknown> = {
        tenant_name: tenantName,
        start_date: startDate,
        cycle,
        components,
      }
      if (tenantEmail.trim()) body.tenant_email = tenantEmail.trim()
      const res = await api.post<{ contract: ContractView; invite?: InvitationView }>(
        `/properties/${propertyId}/contracts`,
        body,
      )
      if (res.invite) {
        setInvite(res.invite) // keep modal open so owner can copy the accept link
      } else {
        onCreated()
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  if (invite) {
    return (
      <Modal title="Contract created" onClose={onCreated}>
        <p className="text-sm text-stone-600 mb-3">
          Share this accept link with the tenant so they can link their account:
        </p>
        <InviteLink invite={invite} />
        <Button className="w-full mt-4" onClick={onCreated}>
          Done
        </Button>
      </Modal>
    )
  }

  return (
    <Modal title="New contract" onClose={onClose}>
      <form onSubmit={submit} className="space-y-3">
        <Field label="Tenant name">
          <Input required value={tenantName} onChange={(e) => setTenantName(e.target.value)} />
        </Field>
        <Field label="Tenant email (optional — sends an invite)">
          <Input type="email" value={tenantEmail} onChange={(e) => setTenantEmail(e.target.value)} />
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
        <ErrorText>{error}</ErrorText>
        <Button className="w-full" disabled={busy} type="submit">
          {busy ? 'Creating…' : 'Create contract'}
        </Button>
      </form>
    </Modal>
  )
}
