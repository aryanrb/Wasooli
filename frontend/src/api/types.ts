// Mirrors the backend view structs. Money fields are integer minor units.

export interface User {
  id: string
  email: string
  name: string
  avatar_url: string
}

export interface Property {
  id: string
  owner_id: string
  name: string
  address: string
  currency: string
  owner_vacant: boolean // true = a tenant's standalone record, no owner linked yet
  created_at: string
}

export interface Component {
  label: string
  amount: number // minor units
}

export interface ContractView {
  id: string
  property_id: string
  tenant_user_id: string | null
  tenant_name: string
  linked: boolean
  start_date: string
  cycle: 'weekly' | 'monthly' | 'yearly'
  anchor: string
  status: 'active' | 'ended'
  created_at: string
  current_components: Component[]
  current_total: number
}

// A tenant's own contract, enriched with the property context they can't fetch directly.
export interface TenancyView extends ContractView {
  property_name: string
  currency: string
  owner_vacant: boolean
}

export interface TermView {
  id: string
  effective_from: string
  components: Component[]
  total: number
}

export interface TransactionView {
  id: string
  property_id: string
  contract_id: string | null
  period_key: string
  recorded_by: string
  direction: 'income' | 'expense'
  amount: number
  components: Component[] | null
  note: string
  paid_at: string
  approval_status: 'pending' | 'approved' | 'rejected'
  created_at: string
}

export interface ProposedTerms {
  tenant_name: string
  start_date: string
  cycle: 'weekly' | 'monthly' | 'yearly'
  anchor: string
  components: Component[]
}

export interface InvitationView {
  id: string
  from_user: string
  to_email: string
  property_id: string | null
  contract_id: string | null
  role: 'tenant' | 'owner' | 'viewer'
  token: string
  status: 'pending' | 'countered' | 'accepted' | 'declined'
  proposed: Partial<ProposedTerms> // empty {} for viewer
  mine: boolean // caller sent it
  your_turn: boolean // caller may act now
  created_at: string
}

export interface AuditView {
  id: string
  action: string
  actor: string
  actor_name: string
  details: Record<string, unknown>
  created_at: string
}

export interface PeriodStatus {
  period: { key: string; start: string; end: string }
  due: number
  paid: number
  outstanding: number
  status: 'due' | 'partial' | 'paid'
}

export interface YearlySummary {
  year: number
  income: number
  expense: number
  net: number
}
