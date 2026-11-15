-- Users: one account per person. Role (owner/tenant/viewer) is per-property,
-- never stored here. google_sub links the Google identity; email is the key.
CREATE TABLE users (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email      text NOT NULL UNIQUE,
    name       text NOT NULL DEFAULT '',
    google_sub text,
    avatar_url text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Properties: owned by one user. Single currency per property.
CREATE TABLE properties (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id   uuid NOT NULL REFERENCES users(id),
    name       text NOT NULL,
    address    text NOT NULL DEFAULT '',
    currency   text NOT NULL DEFAULT 'NPR',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_properties_owner ON properties(owner_id);

-- Contracts: a tenancy on a property. tenant_user_id NULL = solo (tenant not an
-- app user). cycle drives the derived billing periods; anchor is the day-of-cycle.
CREATE TABLE contracts (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id    uuid NOT NULL REFERENCES properties(id),
    tenant_user_id uuid REFERENCES users(id),
    tenant_name    text NOT NULL DEFAULT '',
    start_date     date NOT NULL,
    cycle          text NOT NULL,                  -- weekly | monthly | yearly
    anchor         integer NOT NULL DEFAULT 1,     -- day-of-month / day-of-week / month-of-year
    status         text NOT NULL DEFAULT 'active', -- active | ended
    ended_at       timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_contracts_property ON contracts(property_id);
CREATE INDEX idx_contracts_tenant ON contracts(tenant_user_id);

-- Contract terms: rent breakdown AND its history. A rent change inserts a new row;
-- the term in effect at a period's date defines that period's due amount.
CREATE TABLE contract_terms (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    contract_id    uuid NOT NULL REFERENCES contracts(id),
    effective_from date NOT NULL,
    components     jsonb NOT NULL DEFAULT '[]',    -- [{"label":"rent","amount":1000000}]
    total          bigint NOT NULL,                -- minor units (e.g. paisa/cents)
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_terms_contract ON contract_terms(contract_id, effective_from);

-- Transactions: the ledger. Income (rent received) or expense. Tagged to a
-- contract + period_key for rent; approval_status gates whether it counts.
CREATE TABLE transactions (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id     uuid NOT NULL REFERENCES properties(id),
    contract_id     uuid REFERENCES contracts(id),
    period_key      text,                           -- e.g. 2026-03 (monthly), 2026-W12, 2026
    recorded_by     uuid NOT NULL REFERENCES users(id),
    direction       text NOT NULL,                  -- income | expense
    amount          bigint NOT NULL,                -- minor units
    components      jsonb NOT NULL DEFAULT '[]',
    note            text NOT NULL DEFAULT '',
    paid_at         date NOT NULL,
    approval_status text NOT NULL DEFAULT 'approved', -- pending | approved | rejected
    approved_by     uuid REFERENCES users(id),
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_tx_property ON transactions(property_id);
CREATE INDEX idx_tx_contract_period ON transactions(contract_id, period_key);

-- Invitations: link owner<->tenant or grant viewer access. Either side may send.
CREATE TABLE invitations (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    from_user   uuid NOT NULL REFERENCES users(id),
    to_email    text NOT NULL,
    property_id uuid REFERENCES properties(id),
    contract_id uuid REFERENCES contracts(id),
    role        text NOT NULL,                      -- tenant | owner | viewer
    token       text NOT NULL UNIQUE,
    status      text NOT NULL DEFAULT 'pending',    -- pending | accepted | declined
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_inv_email ON invitations(to_email);

-- Access grants: read-only viewers (parents, partners) on a property.
CREATE TABLE access_grants (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users(id),
    property_id uuid NOT NULL REFERENCES properties(id),
    role        text NOT NULL DEFAULT 'viewer',
    granted_by  uuid NOT NULL REFERENCES users(id),
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, property_id)
);

-- Audit log: contract changes, approvals/rejections, contract end/restart.
CREATE TABLE audit_log (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type text NOT NULL,
    entity_id   uuid NOT NULL,
    action      text NOT NULL,
    actor       uuid NOT NULL REFERENCES users(id),
    details     jsonb NOT NULL DEFAULT '{}',
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_entity ON audit_log(entity_type, entity_id);

---- create above / drop below ----

DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS access_grants;
DROP TABLE IF EXISTS invitations;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS contract_terms;
DROP TABLE IF EXISTS contracts;
DROP TABLE IF EXISTS properties;
DROP TABLE IF EXISTS users;
