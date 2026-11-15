import { useState } from 'react'
import { api } from './api/client'
import type { Component, InvitationView, ProposedTerms, TransactionView } from './api/types'
import { formatMoney } from './lib/money'
import {
  ComponentsEditor,
  rowsToComponents,
  type Row,
} from './ComponentsEditor'
import { Button, Card, Chip, Empty, ErrorText, Field, Input, Modal, Select } from './ui'

// Shared Approvals + Invitations lists, used on both Dashboard and the Approvals tab.

export function ApprovalsList({
  items,
  currencyFor,
  onChanged,
}: {
  items: TransactionView[]
  currencyFor: (t: TransactionView) => string
  onChanged: () => void
}) {
  const [busy, setBusy] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  async function act(id: string, action: 'approve' | 'reject') {
    setBusy(id)
    setError(null)
    try {
      await api.post(`/transactions/${id}/${action}`)
      onChanged()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(null)
    }
  }

  if (!items.length) return <Empty>Nothing waiting for approval.</Empty>

  return (
    <div className="space-y-2">
      <ErrorText>{error}</ErrorText>
      {items.map((t) => (
        <Card key={t.id} className="flex items-center justify-between gap-3">
          <div className="min-w-0">
            <p className="font-medium">
              {formatMoney(t.amount, currencyFor(t))}{' '}
              <Chip kind={t.direction === 'income' ? 'paid' : 'ended'}>{t.direction}</Chip>
            </p>
            <p className="text-xs text-stone-500 truncate">
              {t.note || t.period_key || 'No note'}
            </p>
          </div>
          <div className="flex gap-2 shrink-0">
            <Button variant="secondary" disabled={busy === t.id} onClick={() => void act(t.id, 'reject')}>
              Reject
            </Button>
            <Button disabled={busy === t.id} onClick={() => void act(t.id, 'approve')}>
              Approve
            </Button>
          </div>
        </Card>
      ))}
    </div>
  )
}

// A contract the caller owns and could archive on accepting/merging.
export interface ContractCandidate {
  id: string
  label: string
}

export function InvitationsList({
  items,
  mergeCandidates,
  retireCandidates,
  onChanged,
}: {
  items: InvitationView[]
  mergeCandidates: ContractCandidate[] // owner's own tenant-less contracts (keep=owner merge)
  retireCandidates: ContractCandidate[] // caller's standalone tenancies (archive on join)
  onChanged: () => void
}) {
  const [busy, setBusy] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [countering, setCountering] = useState<InvitationView | null>(null)
  const [acceptingOwner, setAcceptingOwner] = useState<InvitationView | null>(null)
  const [acceptingTenant, setAcceptingTenant] = useState<InvitationView | null>(null)

  async function simpleAct(inv: InvitationView, action: 'accept' | 'decline') {
    setBusy(inv.id)
    setError(null)
    try {
      await api.post(`/invitations/${action}`, { token: inv.token })
      onChanged()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(null)
    }
  }

  // Accept routing: owner invites need the merge choice; tenant invites may retire
  // a standalone; viewer invites accept directly.
  function onAccept(inv: InvitationView) {
    if (inv.role === 'owner') setAcceptingOwner(inv)
    else if (inv.role === 'tenant' && retireCandidates.length) setAcceptingTenant(inv)
    else void simpleAct(inv, 'accept')
  }

  if (!items.length) return null

  return (
    <div className="space-y-2">
      <ErrorText>{error}</ErrorText>
      {items.map((inv) => (
        <Card key={inv.id} className="space-y-2">
          <div className="flex items-center justify-between gap-2">
            <p className="font-medium capitalize">{inv.role} invitation</p>
            <Chip kind={inv.status === 'countered' ? 'partial' : inv.status === 'pending' ? 'pending' : 'ended'}>
              {inv.status}
            </Chip>
          </div>
          <p className="text-xs text-stone-500">
            {inv.mine ? `sent to ${inv.to_email}` : `from ${inv.from_user}`}
          </p>
          <ProposedSummary proposed={inv.proposed} />

          {inv.your_turn ? (
            <div className="flex flex-wrap gap-2 pt-1">
              <Button disabled={busy === inv.id} onClick={() => onAccept(inv)}>
                Accept
              </Button>
              {inv.role !== 'viewer' && (
                <Button variant="secondary" disabled={busy === inv.id} onClick={() => setCountering(inv)}>
                  Counter
                </Button>
              )}
              <Button variant="secondary" disabled={busy === inv.id} onClick={() => void simpleAct(inv, 'decline')}>
                Decline
              </Button>
            </div>
          ) : (
            <p className="text-xs text-stone-400 pt-1">Waiting for the other party…</p>
          )}
        </Card>
      ))}

      {countering && (
        <CounterModal
          invite={countering}
          onClose={() => setCountering(null)}
          onDone={() => {
            setCountering(null)
            onChanged()
          }}
        />
      )}
      {acceptingOwner && (
        <AcceptOwnerModal
          invite={acceptingOwner}
          candidates={mergeCandidates}
          onClose={() => setAcceptingOwner(null)}
          onDone={() => {
            setAcceptingOwner(null)
            onChanged()
          }}
        />
      )}
      {acceptingTenant && (
        <AcceptTenantModal
          invite={acceptingTenant}
          candidates={retireCandidates}
          onClose={() => setAcceptingTenant(null)}
          onDone={() => {
            setAcceptingTenant(null)
            onChanged()
          }}
        />
      )}
    </div>
  )
}

function ProposedSummary({ proposed }: { proposed: Partial<ProposedTerms> }) {
  const comps = proposed.components ?? []
  if (!proposed.tenant_name && !comps.length && !proposed.start_date) return null
  const total = comps.reduce((s, c) => s + c.amount, 0)
  return (
    <div className="rounded-xl bg-stone-50 p-2 text-xs text-stone-600 space-y-0.5">
      {proposed.tenant_name && <p>Tenant: {proposed.tenant_name}</p>}
      {proposed.start_date && (
        <p>
          {proposed.start_date} · {proposed.cycle}
        </p>
      )}
      {comps.map((c, i) => (
        <p key={i} className="flex justify-between">
          <span>{c.label}</span>
          <span>{(c.amount / 100).toFixed(2)}</span>
        </p>
      ))}
      {comps.length > 0 && (
        <p className="flex justify-between font-medium border-t border-stone-200 pt-0.5">
          <span>Total</span>
          <span>{(total / 100).toFixed(2)}</span>
        </p>
      )}
    </div>
  )
}

function componentsToRows(comps: Component[] | undefined): Row[] {
  if (!comps || !comps.length) return [{ label: '', amount: '' }]
  return comps.map((c) => ({ label: c.label, amount: (c.amount / 100).toFixed(2) }))
}

function CounterModal({
  invite,
  onClose,
  onDone,
}: {
  invite: InvitationView
  onClose: () => void
  onDone: () => void
}) {
  const p = invite.proposed
  const [tenantName, setTenantName] = useState(p.tenant_name ?? '')
  const [startDate, setStartDate] = useState(p.start_date ?? new Date().toISOString().slice(0, 10))
  const [cycle, setCycle] = useState<'weekly' | 'monthly' | 'yearly'>(p.cycle ?? 'monthly')
  const [anchor, setAnchor] = useState(p.anchor ?? '')
  const [rows, setRows] = useState<Row[]>(componentsToRows(p.components))
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
      await api.post('/invitations/counter', {
        token: invite.token,
        proposed: { tenant_name: tenantName, start_date: startDate, cycle, anchor, components },
      })
      onDone()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal title="Counter-propose terms" onClose={onClose}>
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
        <ComponentsEditor rows={rows} onChange={setRows} />
        <ErrorText>{error}</ErrorText>
        <Button className="w-full" disabled={busy} type="submit">
          {busy ? 'Sending…' : 'Send counter'}
        </Button>
      </form>
    </Modal>
  )
}

function AcceptOwnerModal({
  invite,
  candidates,
  onClose,
  onDone,
}: {
  invite: InvitationView
  candidates: ContractCandidate[]
  onClose: () => void
  onDone: () => void
}) {
  const [keep, setKeep] = useState<'tenant' | 'owner'>('tenant')
  const [mergeId, setMergeId] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    if (keep === 'owner' && !mergeId) {
      setError('Pick one of your standalone contracts to keep.')
      return
    }
    setBusy(true)
    setError(null)
    try {
      const body: Record<string, unknown> = { token: invite.token, keep }
      // keep=owner requires the kept contract; keep=tenant may optionally archive one.
      if (mergeId) body.merge_contract_id = mergeId
      await api.post('/invitations/accept', body)
      onDone()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal title="Accept — become owner" onClose={onClose}>
      <form onSubmit={submit} className="space-y-3">
        <fieldset className="space-y-2">
          <legend className="text-xs font-medium text-stone-500 mb-1">Which records win?</legend>
          <label className="flex gap-2 items-start">
            <input type="radio" name="keep" checked={keep === 'tenant'} onChange={() => setKeep('tenant')} />
            <span className="text-sm">
              Keep the tenant's record (their transactions stay). Optionally archive one of mine.
            </span>
          </label>
          <label className="flex gap-2 items-start">
            <input type="radio" name="keep" checked={keep === 'owner'} onChange={() => setKeep('owner')} />
            <span className="text-sm">Keep my own contract; archive the tenant's into it.</span>
          </label>
        </fieldset>

        {keep === 'owner' && candidates.length === 0 && (
          <p className="text-xs text-amber-600">
            You have no owner-side records to keep — choose "keep the tenant's record", or create
            your own property first and try again.
          </p>
        )}

        {(keep === 'owner' || candidates.length > 0) && (
          <Field label={keep === 'owner' ? 'My contract to keep (required)' : 'My contract to archive (optional)'}>
            <Select value={mergeId} onChange={(e) => setMergeId(e.target.value)}>
              <option value="">{keep === 'owner' ? 'Select…' : 'None'}</option>
              {candidates.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.label}
                </option>
              ))}
            </Select>
          </Field>
        )}

        <ErrorText>{error}</ErrorText>
        <Button className="w-full" disabled={busy} type="submit">
          {busy ? 'Accepting…' : 'Accept'}
        </Button>
      </form>
    </Modal>
  )
}

function AcceptTenantModal({
  invite,
  candidates,
  onClose,
  onDone,
}: {
  invite: InvitationView
  candidates: ContractCandidate[]
  onClose: () => void
  onDone: () => void
}) {
  const [retireId, setRetireId] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      const body: Record<string, unknown> = { token: invite.token }
      if (retireId) body.retire_contract_id = retireId
      await api.post('/invitations/accept', body)
      onDone()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal title="Accept tenancy" onClose={onClose}>
      <form onSubmit={submit} className="space-y-3">
        <p className="text-sm text-stone-600">Joining this contract with the current terms.</p>
        <Field label="Archive one of my standalone records? (optional)">
          <Select value={retireId} onChange={(e) => setRetireId(e.target.value)}>
            <option value="">None</option>
            {candidates.map((c) => (
              <option key={c.id} value={c.id}>
                {c.label}
              </option>
            ))}
          </Select>
        </Field>
        <ErrorText>{error}</ErrorText>
        <Button className="w-full" disabled={busy} type="submit">
          {busy ? 'Accepting…' : 'Accept'}
        </Button>
      </form>
    </Modal>
  )
}
