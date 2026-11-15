# CLAUDE.md — wasooli (backend)

Context for AI agents working in this repo. A lightweight rental-income ledger API.

## Stack
Go 1.26 · Echo v4 · PostgreSQL 17 · sqlc · Tern migrations (embedded, auto-applied on boot) · zap · pgx/v5 · Google OAuth2 + JWT session cookies.

## Structure (do not flatten)
- `main.go` — wiring + graceful shutdown; embeds `migrations/`.
- `config/` — env-loaded Config (DB, JWT, Google OAuth, FrontendURL).
- `internal/handlers/` — thin: parse request → call service → write response.
- `internal/services/` — business logic; owns validation, DB access, authorization, errors.
- `internal/auth/` — Google OAuth sign-in + JWT session cookie + auth middleware.
- `internal/db/` — pool + migration runner; `internal/db/sqlc/` is GENERATED.
- `internal/types/` — request DTOs with `Validate()`, money/`Component`, derived `Period` logic.
- `internal/httperr/` — typed errors carrying HTTP status.
- `migrations/` — numbered Tern SQL (`up ---- create above / drop below ---- down`).
- `sql/` — sqlc query source.

## Domain model (8 tables)
users · properties(owner) · contracts · contract_terms(rent breakdown + history) ·
transactions(the ledger) · invitations · access_grants(viewers) · audit_log.

Key modeling decisions:
- **Roles are per-property, never global.** owner = owns a property; tenant =
  `contracts.tenant_user_id`; viewer = `access_grants`. A user can be any of these on
  different properties. Nothing is chosen at signup.
- **Owner vs tenant at creation.** A property is created as an owner (managed property)
  or a tenant (standalone tenancy record: `properties.owner_vacant=true`, with one
  auto-created contract where the creator is the tenant). A standalone tenant holds the
  owner seat as a placeholder, so they record on their own ledger freely; when a real
  owner joins (`ClaimPropertyOwner`), `owner_vacant` flips false and the contract becomes
  linked (approvals kick in).
- **"Sync" = one shared record.** A linked contract is a single row both parties read.
  `transactions.approval_status` (pending|approved|rejected) gates whether an entry
  counts. No replication/sync engine.
- **Access is split property-wide vs contract-level.** `requirePropertyWide` (real owner
  or viewer) guards listing all contracts / property transactions / audit. A tenant is
  scoped to their own contract via `getContractWithAccess` and reaches it through
  `ListTenancies`, never the property's contract list. `requireOwnerSide` = holds the
  owner seat (real owner OR tenant placeholder) for editing own terms.
- **Invitations negotiate (free back-and-forth).** `proposed` jsonb carries the terms;
  `last_actor` is who set them, so only the other party may `accept`/`counter`. Roles:
  tenant (owner→tenant, contract), owner (standalone tenant→owner, property+contract),
  viewer. On finalize, agreed terms are applied to the canonical contract.
- **Linking two standalone records archives the non-chosen one.** The owner picks
  `keep: owner|tenant`; the losing contract gets `status='archived'`, `archived_into`
  set — never deleted. Archived contracts drop out of all lists.
- **Billing periods are derived, never stored.** `types.GeneratePeriods(cycle, start, until)`.
  A payment carries `period_key` (e.g. `2026-03`). Outstanding = term total − approved
  payments for that key. Rent changes = a new `contract_terms` row; the term in effect
  at a period's date defines its due amount (`termTotalAt`).
- **Money is `int64` minor units.** Never floats. Store/compute in paisa/cents.

## Conventions
1. **Handler/service split.** No DB calls or business logic in handlers. No SQL in services.
2. **One `Service`, not per-entity.** Authorization (property access checks) and audit
   logging are cross-cutting, so they live on a single `services.Service` with files split
   by domain. This is a deliberate deviation from a per-entity-service template.
3. **Authorization in the service layer** (there is no RLS). Every read/write goes through
   `requireAccess` (owner/viewer/tenant) or `requireOwner`. Add the check before touching data.
4. **Context first.** Pass `context.Context` as the first arg to anything doing I/O.
5. **sqlc only.** Add queries to `sql/*.sql`, run `make generate`. Never edit `internal/db/sqlc/`.
   jsonb columns come back as `[]byte` → return them via `json.RawMessage` in a view struct
   (see `services/views.go`) so they don't base64-encode.
6. **Errors.** Services return `*httperr.Error`. Handlers return the error; the server's
   error handler renders JSON and logs 5xx. Map `pgx.ErrNoRows` → `httperr.NotFound`.
7. **Logging.** Structured zap only. Log at boundaries, not hot loops.
8. **Migrations.** One concern per file; always write a working `down`; never edit a merged
   migration — add a new one. They auto-apply on boot and via `make migrate`.
9. **Config.** New env var → add to `.env.example` in the same change.

## Auth flow
`GET /api/v1/auth/google/login` → Google consent → `GET /auth/google/callback` upserts the
user, signs a JWT, sets the `wasooli_session` httpOnly cookie, redirects to `FRONTEND_URL`.
All other routes require that cookie (`auth.Middleware`). `auth.UserID(c)` reads the caller.

## Add an endpoint
1. Add queries to `sql/<entity>.sql`; `make generate`.
2. Add logic + authorization to `internal/services/<entity>.go`.
3. Add a thin handler in `internal/handlers/<entity>.go`.
4. Register the route in `handler.go`'s `RegisterRoutes`.
5. Add a test for any non-trivial logic.

## Run / test locally
```
make db-up          # postgres on :5432 (docker) — or use a local postgres
cp .env.example .env # set JWT_SECRET and Google creds
make run            # migrations auto-apply on boot
make test
```

## Deferred (intentional TODOs — see `ponytail:` comments)
Overpayment carry-forward (credit is shown, not auto-applied), multi-currency, deposits,
month-end anchor drift (Jan 31 → Mar 3 via AddDate), email delivery of invitations
(the invite token is returned in the API response; the frontend builds the accept link).
