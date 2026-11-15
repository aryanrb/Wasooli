-- Tenant-created standalone records: the owner seat stays vacant until a real owner
-- joins. owner_id still holds the creator (so they can manage their own record), but
-- owner_vacant=true means "this is a tenant's standalone ledger", keeping it out of
-- the owner-facing property list and off the linked-approval path.
ALTER TABLE properties ADD COLUMN owner_vacant boolean NOT NULL DEFAULT false;

-- When two standalone records are linked, the non-chosen contract is archived (not
-- deleted) and points at the contract that superseded it.
ALTER TABLE contracts ADD COLUMN archived_into uuid REFERENCES contracts(id);

-- Invitation negotiation (free back-and-forth): proposed carries the terms under
-- discussion; last_actor is who set the current proposal, so only the OTHER party
-- may accept or counter it.
ALTER TABLE invitations ADD COLUMN proposed jsonb NOT NULL DEFAULT '{}';
ALTER TABLE invitations ADD COLUMN last_actor uuid REFERENCES users(id);

---- create above / drop below ----

ALTER TABLE invitations DROP COLUMN last_actor;
ALTER TABLE invitations DROP COLUMN proposed;
ALTER TABLE contracts DROP COLUMN archived_into;
ALTER TABLE properties DROP COLUMN owner_vacant;
