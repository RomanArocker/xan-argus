# Database Seed Data Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create a goose migration that seeds realistic example data into all tables, so the app has useful demo content on first Docker start.

**Architecture:** A single goose SQL migration (`005_seed_data.sql`) that inserts example data with fixed UUIDs. Uses `ON CONFLICT DO NOTHING` so re-running is safe. Inserts follow FK dependency order: independent tables first, then dependent tables. The migration runs automatically on startup via the existing `database.RunMigrations()` call.

**Tech Stack:** PostgreSQL (goose SQL migration)

---

## File Structure

| Action | File | Purpose |
|--------|------|---------|
| Create | `db/migrations/005_seed_data.sql` | Goose migration with all seed data |

## Data Design

### Fixed UUIDs (for FK references)

We use fixed UUIDs so FKs can reference them reliably. Pattern: `00000000-0000-0000-0000-00000000XXYY` where XX = table, YY = row number.

| Table | UUID Prefix | Count |
|-------|-------------|-------|
| customers | `...0101` – `...0103` | 3 |
| users (customer_staff) | `...0201` – `...0204` | 4 |
| users (internal_staff) | `...0205` – `...0206` | 2 |
| services | `...0301` – `...0303` | 3 |
| user_assignments | `...0401` – `...0405` | 5 |
| assets | `...0501` – `...0508` | 8 |
| licenses | `...0601` – `...0606` | 6 |
| customer_services | `...0701` – `...0704` | 4 |

### Example Data Theme

An IT services company managing three German SMB customers:
- **Müller GmbH** — small manufacturing firm, 2 staff, laptops + servers
- **Schmidt & Partner** — law firm, 1 staff, monitors + printers
- **TechStart AG** — startup, 2 staff, mixed hardware, many licenses

---

## Task 1: Create seed migration — independent tables

**Files:**
- Create: `db/migrations/005_seed_data.sql`

- [ ] **Step 1: Create migration file with customers, users, services**

Write `db/migrations/005_seed_data.sql` with:

```sql
-- db/migrations/005_seed_data.sql

-- +goose Up

-- ============================================================
-- Seed data: realistic demo content for first-time setup
-- Uses fixed UUIDs and ON CONFLICT DO NOTHING for idempotency
-- ============================================================

-- Customers
INSERT INTO customers (id, name, contact_email, notes) VALUES
    ('00000000-0000-0000-0000-000000000101', 'Müller GmbH', 'info@mueller-gmbh.de', 'Manufacturing company, 25 employees. Main contact: Hans Müller.'),
    ('00000000-0000-0000-0000-000000000102', 'Schmidt & Partner', 'office@schmidt-partner.de', 'Law firm, 10 employees. Premium support contract.'),
    ('00000000-0000-0000-0000-000000000103', 'TechStart AG', 'admin@techstart.de', 'Tech startup, 40 employees. Rapid growth, frequent hardware orders.')
ON CONFLICT DO NOTHING;

-- Users (customer_staff)
INSERT INTO users (id, type, first_name, last_name) VALUES
    ('00000000-0000-0000-0000-000000000201', 'customer_staff', 'Hans', 'Müller'),
    ('00000000-0000-0000-0000-000000000202', 'customer_staff', 'Anna', 'Weber'),
    ('00000000-0000-0000-0000-000000000203', 'customer_staff', 'Klaus', 'Schmidt'),
    ('00000000-0000-0000-0000-000000000204', 'customer_staff', 'Lisa', 'Berger')
ON CONFLICT DO NOTHING;

-- Users (internal_staff)
INSERT INTO users (id, type, first_name, last_name) VALUES
    ('00000000-0000-0000-0000-000000000205', 'internal_staff', 'Max', 'Fischer'),
    ('00000000-0000-0000-0000-000000000206', 'internal_staff', 'Sarah', 'Koch')
ON CONFLICT DO NOTHING;

-- Services
INSERT INTO services (id, name, description) VALUES
    ('00000000-0000-0000-0000-000000000301', 'Managed IT Support', 'Full IT infrastructure management including monitoring, patching, and helpdesk.'),
    ('00000000-0000-0000-0000-000000000302', 'Cloud Backup', 'Daily encrypted backups to cloud storage with 30-day retention.'),
    ('00000000-0000-0000-0000-000000000303', 'Security Audit', 'Quarterly security assessment including vulnerability scanning and compliance review.')
ON CONFLICT DO NOTHING;
```

- [ ] **Step 2: Verify syntax**

Run: `docker compose up --build`
Expected: App starts, migration 005 applied, no errors.

- [ ] **Step 3: Commit**

```bash
git add db/migrations/005_seed_data.sql
git commit -m "feat: add seed migration — customers, users, services"
```

---

## Task 2: Add user assignments

**Files:**
- Modify: `db/migrations/005_seed_data.sql`

- [ ] **Step 1: Append user_assignments to migration**

Add after the services block:

```sql
-- User Assignments (only customer_staff users → enforced by trigger)
INSERT INTO user_assignments (id, user_id, customer_id, role, email, phone, notes) VALUES
    -- Müller GmbH: Hans (IT contact) + Anna (accounting)
    ('00000000-0000-0000-0000-000000000401', '00000000-0000-0000-0000-000000000201', '00000000-0000-0000-0000-000000000101', 'IT-Ansprechpartner', 'h.mueller@mueller-gmbh.de', '+49 89 1234567', 'Primary IT contact, available Mon-Fri.'),
    ('00000000-0000-0000-0000-000000000402', '00000000-0000-0000-0000-000000000202', '00000000-0000-0000-0000-000000000101', 'Buchhaltung', 'a.weber@mueller-gmbh.de', '+49 89 1234568', 'Handles license purchases and invoices.'),
    -- Schmidt & Partner: Klaus (managing partner)
    ('00000000-0000-0000-0000-000000000403', '00000000-0000-0000-0000-000000000203', '00000000-0000-0000-0000-000000000102', 'Managing Partner', 'k.schmidt@schmidt-partner.de', '+49 30 9876543', 'Decision maker for all IT purchases.'),
    -- TechStart AG: Lisa (CTO) + Anna (also assigned here as freelance consultant)
    ('00000000-0000-0000-0000-000000000404', '00000000-0000-0000-0000-000000000204', '00000000-0000-0000-0000-000000000103', 'CTO', 'l.berger@techstart.de', '+49 40 5551234', 'Technical lead, approves all infrastructure changes.'),
    ('00000000-0000-0000-0000-000000000405', '00000000-0000-0000-0000-000000000202', '00000000-0000-0000-0000-000000000103', 'IT Consultant', 'a.weber@techstart.de', '+49 40 5551235', 'External consultant, on-site Tuesdays.')
ON CONFLICT DO NOTHING;
```

Note: Anna Weber (0202) is assigned to both Müller GmbH and TechStart AG — this demonstrates multi-assignment.

- [ ] **Step 2: Rebuild and verify**

Run: `docker compose up --build`
Expected: No errors, user assignments created.

- [ ] **Step 3: Commit**

```bash
git add db/migrations/005_seed_data.sql
git commit -m "feat: add seed user assignments with multi-customer example"
```

---

## Task 3: Add assets

**Files:**
- Modify: `db/migrations/005_seed_data.sql`

- [ ] **Step 1: Append assets to migration**

Hardware categories from migration 003 don't have fixed UUIDs, so we use a subquery. Add:

```sql
-- Assets
-- Note: category_id references hardware_categories seeded in migration 003 (by name)
INSERT INTO assets (id, customer_id, category_id, name, description, metadata, field_values) VALUES
    -- Müller GmbH: 3 assets
    ('00000000-0000-0000-0000-000000000501',
     '00000000-0000-0000-0000-000000000101',
     (SELECT id FROM hardware_categories WHERE name = 'Laptop'),
     'ThinkPad T14s #1', 'Hans Müller primary workstation',
     '{"serial": "PF-4A2B8C", "warranty_until": "2027-06-15"}', '{}'),
    ('00000000-0000-0000-0000-000000000502',
     '00000000-0000-0000-0000-000000000101',
     (SELECT id FROM hardware_categories WHERE name = 'Laptop'),
     'ThinkPad T14s #2', 'Anna Weber primary workstation',
     '{"serial": "PF-7D3E9F", "warranty_until": "2027-06-15"}', '{}'),
    ('00000000-0000-0000-0000-000000000503',
     '00000000-0000-0000-0000-000000000101',
     (SELECT id FROM hardware_categories WHERE name = 'Server'),
     'PowerEdge R750', 'On-premise file server',
     '{"serial": "SV-1122334", "rack_position": "A3-U12", "warranty_until": "2028-01-30"}', '{}'),
    -- Schmidt & Partner: 2 assets
    ('00000000-0000-0000-0000-000000000504',
     '00000000-0000-0000-0000-000000000102',
     (SELECT id FROM hardware_categories WHERE name = 'Monitor'),
     'Dell U2723QE', 'Klaus Schmidt office monitor',
     '{"serial": "MN-8877665"}', '{}'),
    ('00000000-0000-0000-0000-000000000505',
     '00000000-0000-0000-0000-000000000102',
     (SELECT id FROM hardware_categories WHERE name = 'Printer'),
     'HP LaserJet Pro M404', 'Shared office printer, 2nd floor',
     '{"serial": "PR-5544332", "ip_address": "192.168.1.50"}', '{}'),
    -- TechStart AG: 3 assets
    ('00000000-0000-0000-0000-000000000506',
     '00000000-0000-0000-0000-000000000103',
     (SELECT id FROM hardware_categories WHERE name = 'Server'),
     'ProLiant DL380', 'Development server',
     '{"serial": "SV-9988776", "rack_position": "B1-U8"}', '{}'),
    ('00000000-0000-0000-0000-000000000507',
     '00000000-0000-0000-0000-000000000103',
     (SELECT id FROM hardware_categories WHERE name = 'Network Device'),
     'UniFi Dream Machine Pro', 'Main office router/firewall',
     '{"serial": "NW-1239876", "ip_address": "192.168.0.1"}', '{}'),
    ('00000000-0000-0000-0000-000000000508',
     '00000000-0000-0000-0000-000000000103',
     (SELECT id FROM hardware_categories WHERE name = 'Laptop'),
     'MacBook Pro 16"', 'Lisa Berger primary workstation',
     '{"serial": "LP-APPLE01", "warranty_until": "2027-11-20"}', '{}')
ON CONFLICT DO NOTHING;
```

- [ ] **Step 2: Rebuild and verify**

Run: `docker compose up --build`
Expected: No errors, 8 assets created with correct category references.

- [ ] **Step 3: Commit**

```bash
git add db/migrations/005_seed_data.sql
git commit -m "feat: add seed assets across all customers"
```

---

## Task 4: Add licenses

**Files:**
- Modify: `db/migrations/005_seed_data.sql`

- [ ] **Step 1: Append licenses to migration**

```sql
-- Licenses
INSERT INTO licenses (id, customer_id, user_assignment_id, product_name, license_key, quantity, valid_from, valid_until) VALUES
    -- Müller GmbH: 2 licenses (one assigned to Hans)
    ('00000000-0000-0000-0000-000000000601',
     '00000000-0000-0000-0000-000000000101',
     '00000000-0000-0000-0000-000000000401',
     'Microsoft 365 Business', 'M365-MUE-2024-ABCD', 25, '2025-01-01', '2026-12-31'),
    ('00000000-0000-0000-0000-000000000602',
     '00000000-0000-0000-0000-000000000101',
     NULL,
     'Adobe Creative Cloud', 'ACC-MUE-2025-EFGH', 5, '2025-06-01', '2026-05-31'),
    -- Schmidt & Partner: 2 licenses (one assigned to Klaus)
    ('00000000-0000-0000-0000-000000000603',
     '00000000-0000-0000-0000-000000000102',
     '00000000-0000-0000-0000-000000000403',
     'Microsoft 365 Business', 'M365-SCH-2025-IJKL', 10, '2025-03-01', '2026-02-28'),
    ('00000000-0000-0000-0000-000000000604',
     '00000000-0000-0000-0000-000000000102',
     NULL,
     'DATEV Mittelstand', 'DAT-SCH-2025-MNOP', 3, '2025-01-01', '2025-12-31'),
    -- TechStart AG: 2 licenses (one assigned to Lisa)
    ('00000000-0000-0000-0000-000000000605',
     '00000000-0000-0000-0000-000000000103',
     '00000000-0000-0000-0000-000000000404',
     'JetBrains All Products', 'JB-TECH-2025-QRST', 15, '2025-04-01', '2026-03-31'),
    ('00000000-0000-0000-0000-000000000606',
     '00000000-0000-0000-0000-000000000103',
     NULL,
     'Slack Business+', 'SLK-TECH-2025-UVWX', 40, '2025-01-01', '2026-12-31')
ON CONFLICT DO NOTHING;
```

Note: License customer consistency trigger requires `user_assignment.customer_id` to match `licenses.customer_id`. All assignments above are correct (e.g., assignment 0401 belongs to customer 0101, license 0601 also belongs to customer 0101).

- [ ] **Step 2: Rebuild and verify**

Run: `docker compose up --build`
Expected: No errors, 6 licenses created. Trigger validation passes.

- [ ] **Step 3: Commit**

```bash
git add db/migrations/005_seed_data.sql
git commit -m "feat: add seed licenses with assignment references"
```

---

## Task 5: Add customer_services and +goose Down

**Files:**
- Modify: `db/migrations/005_seed_data.sql`

- [ ] **Step 1: Append customer_services and down migration**

```sql
-- Customer Services
INSERT INTO customer_services (id, customer_id, service_id, customizations, notes) VALUES
    -- Müller GmbH: Managed IT + Cloud Backup
    ('00000000-0000-0000-0000-000000000701',
     '00000000-0000-0000-0000-000000000101',
     '00000000-0000-0000-0000-000000000301',
     '{"sla": "8x5", "response_time_hours": 4}',
     'Standard business hours support.'),
    ('00000000-0000-0000-0000-000000000702',
     '00000000-0000-0000-0000-000000000101',
     '00000000-0000-0000-0000-000000000302',
     '{"retention_days": 30, "storage_gb": 500}',
     'Daily backups, 500 GB included.'),
    -- Schmidt & Partner: Managed IT + Security Audit
    ('00000000-0000-0000-0000-000000000703',
     '00000000-0000-0000-0000-000000000102',
     '00000000-0000-0000-0000-000000000301',
     '{"sla": "24x7", "response_time_hours": 1}',
     'Premium 24/7 support — law firm requires guaranteed uptime.'),
    ('00000000-0000-0000-0000-000000000704',
     '00000000-0000-0000-0000-000000000102',
     '00000000-0000-0000-0000-000000000303',
     '{"frequency": "quarterly", "include_pentest": true}',
     'GDPR compliance requirement, quarterly audits with penetration testing.')
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM customer_services WHERE id IN (
    '00000000-0000-0000-0000-000000000701',
    '00000000-0000-0000-0000-000000000702',
    '00000000-0000-0000-0000-000000000703',
    '00000000-0000-0000-0000-000000000704'
);
DELETE FROM licenses WHERE id IN (
    '00000000-0000-0000-0000-000000000601',
    '00000000-0000-0000-0000-000000000602',
    '00000000-0000-0000-0000-000000000603',
    '00000000-0000-0000-0000-000000000604',
    '00000000-0000-0000-0000-000000000605',
    '00000000-0000-0000-0000-000000000606'
);
DELETE FROM assets WHERE id IN (
    '00000000-0000-0000-0000-000000000501',
    '00000000-0000-0000-0000-000000000502',
    '00000000-0000-0000-0000-000000000503',
    '00000000-0000-0000-0000-000000000504',
    '00000000-0000-0000-0000-000000000505',
    '00000000-0000-0000-0000-000000000506',
    '00000000-0000-0000-0000-000000000507',
    '00000000-0000-0000-0000-000000000508'
);
DELETE FROM user_assignments WHERE id IN (
    '00000000-0000-0000-0000-000000000401',
    '00000000-0000-0000-0000-000000000402',
    '00000000-0000-0000-0000-000000000403',
    '00000000-0000-0000-0000-000000000404',
    '00000000-0000-0000-0000-000000000405'
);
DELETE FROM users WHERE id IN (
    '00000000-0000-0000-0000-000000000201',
    '00000000-0000-0000-0000-000000000202',
    '00000000-0000-0000-0000-000000000203',
    '00000000-0000-0000-0000-000000000204',
    '00000000-0000-0000-0000-000000000205',
    '00000000-0000-0000-0000-000000000206'
);
DELETE FROM services WHERE id IN (
    '00000000-0000-0000-0000-000000000301',
    '00000000-0000-0000-0000-000000000302',
    '00000000-0000-0000-0000-000000000303'
);
DELETE FROM customers WHERE id IN (
    '00000000-0000-0000-0000-000000000101',
    '00000000-0000-0000-0000-000000000102',
    '00000000-0000-0000-0000-000000000103'
);
```

- [ ] **Step 2: Final rebuild and full verification**

Run: `docker compose up --build`
Expected: App starts cleanly, migration 005 applied. Verify data:
```bash
docker compose exec db psql -U xanargus -c "SELECT count(*) FROM customers;"
# Expected: 3
docker compose exec db psql -U xanargus -c "SELECT count(*) FROM users;"
# Expected: 6
docker compose exec db psql -U xanargus -c "SELECT count(*) FROM assets;"
# Expected: 8
docker compose exec db psql -U xanargus -c "SELECT count(*) FROM licenses;"
# Expected: 6
```

- [ ] **Step 3: Commit**

```bash
git add db/migrations/005_seed_data.sql
git commit -m "feat: complete seed migration with customer_services and rollback"
```

---

## Summary

| Table | Seed Count | Notes |
|-------|-----------|-------|
| customers | 3 | German SMBs: manufacturing, law, startup |
| users | 6 | 4 customer_staff + 2 internal_staff |
| services | 3 | Managed IT, Cloud Backup, Security Audit |
| user_assignments | 5 | Includes multi-customer assignment (Anna) |
| assets | 8 | Mixed categories, realistic metadata |
| licenses | 6 | Some with user assignment, some without |
| customer_services | 4 | With JSONB customizations |
| **Total** | **35 rows** | |
