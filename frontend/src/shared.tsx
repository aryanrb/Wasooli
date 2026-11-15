import type { AuditView, InvitationView, TransactionView } from './api/types'
import { formatMoney } from './lib/money'
import { relativeTime } from './lib/time'
import { Card, Chip, Empty } from './ui'

export function InviteLink({ invite }: { invite: InvitationView }) {
  const url = `${window.location.origin}/invite/accept?token=${invite.token}`
  return (
    <div className="rounded-xl bg-stone-100 p-3 text-xs break-all">
      <p className="text-stone-500 mb-1">Accept link for {invite.to_email}:</p>
      <a className="text-indigo-700 underline" href={url}>
        {url}
      </a>
    </div>
  )
}

export function TransactionsList({
  items,
  currency,
}: {
  items: TransactionView[]
  currency: string
}) {
  if (!items.length) return <Empty>No transactions.</Empty>
  return (
    <div className="space-y-2">
      {items.map((t) => (
        <Card key={t.id} className="flex items-center justify-between gap-3">
          <div className="min-w-0">
            <p className="font-medium">
              <span className={t.direction === 'income' ? 'text-green-700' : 'text-red-700'}>
                {t.direction === 'income' ? '+' : '−'}
                {formatMoney(t.amount, currency)}
              </span>
            </p>
            <p className="text-xs text-stone-500 truncate">
              {t.note || t.period_key || t.direction} · {t.paid_at}
            </p>
          </div>
          <Chip kind={t.approval_status} />
        </Card>
      ))}
    </div>
  )
}

export function AuditTimeline({ items }: { items: AuditView[] }) {
  if (!items.length) return <Empty>No activity yet.</Empty>
  return (
    <ol className="space-y-3 border-l border-stone-200 pl-4">
      {items.map((a) => (
        <li key={a.id} className="relative">
          <span className="absolute -left-[21px] top-1.5 h-2 w-2 rounded-full bg-stone-400" />
          <p className="text-sm">
            <span className="font-medium">{a.action.replace(/_/g, ' ')}</span>
            {a.actor_name ? <span className="text-stone-500"> by {a.actor_name}</span> : null}
          </p>
          <p className="text-xs text-stone-400">{relativeTime(a.created_at)}</p>
        </li>
      ))}
    </ol>
  )
}
