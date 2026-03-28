-- db/migrations/005_seed_data.sql
-- Seed data for development and testing.
-- Inserts 3 customers, 6 users, 3 services, 5 user assignments,
-- 14 field definitions, 8 assets, 6 licenses, and 4 customer services with predictable UUIDs.

-- +goose Up

-- customers
INSERT INTO customers (id, name, contact_email, notes) VALUES
    ('00000000-0000-0000-0000-000000000101', 'Müller AG', 'info@mueller-ag.ch', 'Manufacturing company in Zürich, 25 employees. Main contact: Hans Müller.'),
    ('00000000-0000-0000-0000-000000000102', 'Schmidt & Partner', 'office@schmidt-partner.ch', 'Law firm in Bern, 10 employees. Premium support contract.'),
    ('00000000-0000-0000-0000-000000000103', 'TechStart GmbH', 'admin@techstart.ch', 'Tech startup in Basel, 40 employees. Rapid growth, frequent hardware orders.')
ON CONFLICT (id) DO NOTHING;

-- users (customer_staff)
INSERT INTO users (id, type, first_name, last_name) VALUES
    ('00000000-0000-0000-0000-000000000201', 'customer_staff', 'Hans', 'Müller'),
    ('00000000-0000-0000-0000-000000000202', 'customer_staff', 'Anna', 'Weber'),
    ('00000000-0000-0000-0000-000000000203', 'customer_staff', 'Klaus', 'Schmidt'),
    ('00000000-0000-0000-0000-000000000204', 'customer_staff', 'Lisa', 'Berger')
ON CONFLICT (id) DO NOTHING;

-- users (internal_staff)
INSERT INTO users (id, type, first_name, last_name) VALUES
    ('00000000-0000-0000-0000-000000000205', 'internal_staff', 'Max', 'Fischer'),
    ('00000000-0000-0000-0000-000000000206', 'internal_staff', 'Sarah', 'Koch')
ON CONFLICT (id) DO NOTHING;

-- services
INSERT INTO services (id, name, description) VALUES
    ('00000000-0000-0000-0000-000000000301', 'Managed IT Support', 'Full IT infrastructure management including monitoring, patching, and helpdesk.'),
    ('00000000-0000-0000-0000-000000000302', 'Cloud Backup', 'Daily encrypted backups to cloud storage with 30-day retention.'),
    ('00000000-0000-0000-0000-000000000303', 'Security Audit', 'Quarterly security assessment including vulnerability scanning and compliance review.')
ON CONFLICT (id) DO NOTHING;

-- user_assignments (only customer_staff users; Anna Weber assigned to 2 customers)
INSERT INTO user_assignments (id, user_id, customer_id, role, email, phone, notes) VALUES
    ('00000000-0000-0000-0000-000000000401', '00000000-0000-0000-0000-000000000201', '00000000-0000-0000-0000-000000000101', 'IT-Ansprechpartner', 'h.mueller@mueller-ag.ch', '+41 44 123 45 67', 'Primary IT contact, available Mon-Fri.'),
    ('00000000-0000-0000-0000-000000000402', '00000000-0000-0000-0000-000000000202', '00000000-0000-0000-0000-000000000101', 'Buchhaltung', 'a.weber@mueller-ag.ch', '+41 44 123 45 68', 'Handles license purchases and invoices.'),
    ('00000000-0000-0000-0000-000000000403', '00000000-0000-0000-0000-000000000203', '00000000-0000-0000-0000-000000000102', 'Managing Partner', 'k.schmidt@schmidt-partner.ch', '+41 31 987 65 43', 'Decision maker for all IT purchases.'),
    ('00000000-0000-0000-0000-000000000404', '00000000-0000-0000-0000-000000000204', '00000000-0000-0000-0000-000000000103', 'CTO', 'l.berger@techstart.ch', '+41 61 555 12 34', 'Technical lead, approves all infrastructure changes.'),
    ('00000000-0000-0000-0000-000000000405', '00000000-0000-0000-0000-000000000202', '00000000-0000-0000-0000-000000000103', 'IT Consultant', 'a.weber@techstart.ch', '+41 61 555 12 35', 'External consultant, on-site Tuesdays.')
ON CONFLICT (id) DO NOTHING;

-- category_field_definitions (category_id resolved by name subquery)
-- Laptop fields
INSERT INTO category_field_definitions (id, category_id, name, field_type, required, sort_order) VALUES
    ('00000000-0000-0000-0000-000000000801', (SELECT id FROM hardware_categories WHERE name = 'Laptop'), 'Hostname', 'text', false, 1),
    ('00000000-0000-0000-0000-000000000802', (SELECT id FROM hardware_categories WHERE name = 'Laptop'), 'Operating System', 'text', false, 2),
    ('00000000-0000-0000-0000-000000000803', (SELECT id FROM hardware_categories WHERE name = 'Laptop'), 'RAM (GB)', 'number', false, 3),
    ('00000000-0000-0000-0000-000000000804', (SELECT id FROM hardware_categories WHERE name = 'Laptop'), 'Storage (GB)', 'number', false, 4)
ON CONFLICT (id) DO NOTHING;
-- Server fields
INSERT INTO category_field_definitions (id, category_id, name, field_type, required, sort_order) VALUES
    ('00000000-0000-0000-0000-000000000805', (SELECT id FROM hardware_categories WHERE name = 'Server'), 'Hostname', 'text', false, 1),
    ('00000000-0000-0000-0000-000000000806', (SELECT id FROM hardware_categories WHERE name = 'Server'), 'Operating System', 'text', false, 2),
    ('00000000-0000-0000-0000-000000000807', (SELECT id FROM hardware_categories WHERE name = 'Server'), 'RAM (GB)', 'number', false, 3),
    ('00000000-0000-0000-0000-000000000808', (SELECT id FROM hardware_categories WHERE name = 'Server'), 'CPU Cores', 'number', false, 4)
ON CONFLICT (id) DO NOTHING;
-- Printer fields
INSERT INTO category_field_definitions (id, category_id, name, field_type, required, sort_order) VALUES
    ('00000000-0000-0000-0000-000000000809', (SELECT id FROM hardware_categories WHERE name = 'Printer'), 'Print Technology', 'text', false, 1),
    ('00000000-0000-0000-0000-000000000810', (SELECT id FROM hardware_categories WHERE name = 'Printer'), 'Color', 'boolean', false, 2)
ON CONFLICT (id) DO NOTHING;
-- Monitor fields
INSERT INTO category_field_definitions (id, category_id, name, field_type, required, sort_order) VALUES
    ('00000000-0000-0000-0000-000000000811', (SELECT id FROM hardware_categories WHERE name = 'Monitor'), 'Resolution', 'text', false, 1),
    ('00000000-0000-0000-0000-000000000812', (SELECT id FROM hardware_categories WHERE name = 'Monitor'), 'Panel Size (inches)', 'number', false, 2)
ON CONFLICT (id) DO NOTHING;
-- Network Device fields
INSERT INTO category_field_definitions (id, category_id, name, field_type, required, sort_order) VALUES
    ('00000000-0000-0000-0000-000000000813', (SELECT id FROM hardware_categories WHERE name = 'Network Device'), 'Firmware Version', 'text', false, 1),
    ('00000000-0000-0000-0000-000000000814', (SELECT id FROM hardware_categories WHERE name = 'Network Device'), 'Port Count', 'number', false, 2)
ON CONFLICT (id) DO NOTHING;

-- assets (category_id resolved by name subquery)
INSERT INTO assets (id, customer_id, category_id, name, description, metadata, field_values) VALUES
    -- Müller AG: Laptops + Server
    ('00000000-0000-0000-0000-000000000501', '00000000-0000-0000-0000-000000000101', (SELECT id FROM hardware_categories WHERE name = 'Laptop'), 'ThinkPad T14s #1', 'Hans Müller primary workstation', '{"serial": "PF-4A2B8C", "warranty_until": "2027-06-15"}',
     '{"00000000-0000-0000-0000-000000000801": "PC-HM01", "00000000-0000-0000-0000-000000000802": "Windows 11 Pro", "00000000-0000-0000-0000-000000000803": 16, "00000000-0000-0000-0000-000000000804": 512}'),
    ('00000000-0000-0000-0000-000000000502', '00000000-0000-0000-0000-000000000101', (SELECT id FROM hardware_categories WHERE name = 'Laptop'), 'ThinkPad T14s #2', 'Anna Weber primary workstation', '{"serial": "PF-7D3E9F", "warranty_until": "2027-06-15"}',
     '{"00000000-0000-0000-0000-000000000801": "PC-AW02", "00000000-0000-0000-0000-000000000802": "Windows 11 Pro", "00000000-0000-0000-0000-000000000803": 16, "00000000-0000-0000-0000-000000000804": 256}'),
    ('00000000-0000-0000-0000-000000000503', '00000000-0000-0000-0000-000000000101', (SELECT id FROM hardware_categories WHERE name = 'Server'), 'PowerEdge R750', 'On-premise file server', '{"serial": "SV-1122334", "rack_position": "A3-U12", "warranty_until": "2028-01-30"}',
     '{"00000000-0000-0000-0000-000000000805": "SRV-FS01", "00000000-0000-0000-0000-000000000806": "Windows Server 2022", "00000000-0000-0000-0000-000000000807": 64, "00000000-0000-0000-0000-000000000808": 16}'),
    -- Schmidt & Partner: Monitor + Printer
    ('00000000-0000-0000-0000-000000000504', '00000000-0000-0000-0000-000000000102', (SELECT id FROM hardware_categories WHERE name = 'Monitor'), 'Dell U2723QE', 'Klaus Schmidt office monitor', '{"serial": "MN-8877665"}',
     '{"00000000-0000-0000-0000-000000000811": "3840x2160", "00000000-0000-0000-0000-000000000812": 27}'),
    ('00000000-0000-0000-0000-000000000505', '00000000-0000-0000-0000-000000000102', (SELECT id FROM hardware_categories WHERE name = 'Printer'), 'HP LaserJet Pro M404', 'Shared office printer, 2nd floor', '{"serial": "PR-5544332", "ip_address": "192.168.1.50"}',
     '{"00000000-0000-0000-0000-000000000809": "Laser", "00000000-0000-0000-0000-000000000810": false}'),
    -- TechStart GmbH: Server + Network Device + Laptop
    ('00000000-0000-0000-0000-000000000506', '00000000-0000-0000-0000-000000000103', (SELECT id FROM hardware_categories WHERE name = 'Server'), 'ProLiant DL380', 'Development server', '{"serial": "SV-9988776", "rack_position": "B1-U8"}',
     '{"00000000-0000-0000-0000-000000000805": "SRV-DEV01", "00000000-0000-0000-0000-000000000806": "Ubuntu 24.04 LTS", "00000000-0000-0000-0000-000000000807": 128, "00000000-0000-0000-0000-000000000808": 32}'),
    ('00000000-0000-0000-0000-000000000507', '00000000-0000-0000-0000-000000000103', (SELECT id FROM hardware_categories WHERE name = 'Network Device'), 'UniFi Dream Machine Pro', 'Main office router/firewall', '{"serial": "NW-1239876", "ip_address": "192.168.0.1"}',
     '{"00000000-0000-0000-0000-000000000813": "3.2.7", "00000000-0000-0000-0000-000000000814": 8}'),
    ('00000000-0000-0000-0000-000000000508', '00000000-0000-0000-0000-000000000103', (SELECT id FROM hardware_categories WHERE name = 'Laptop'), 'MacBook Pro 16"', 'Lisa Berger primary workstation', '{"serial": "LP-APPLE01", "warranty_until": "2027-11-20"}',
     '{"00000000-0000-0000-0000-000000000801": "MAC-LB01", "00000000-0000-0000-0000-000000000802": "macOS Sequoia", "00000000-0000-0000-0000-000000000803": 32, "00000000-0000-0000-0000-000000000804": 1024}')
ON CONFLICT (id) DO NOTHING;

-- licenses (user_assignment_id NULL where not applicable; customer_id must match assignment's customer_id)
INSERT INTO licenses (id, customer_id, user_assignment_id, product_name, license_key, quantity, valid_from, valid_until) VALUES
    ('00000000-0000-0000-0000-000000000601', '00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000401', 'Microsoft 365 Business', 'M365-MUE-2024-ABCD', 25, '2025-01-01', '2026-12-31'),
    ('00000000-0000-0000-0000-000000000602', '00000000-0000-0000-0000-000000000101', NULL, 'Adobe Creative Cloud', 'ACC-MUE-2025-EFGH', 5, '2025-06-01', '2026-05-31'),
    ('00000000-0000-0000-0000-000000000603', '00000000-0000-0000-0000-000000000102', '00000000-0000-0000-0000-000000000403', 'Microsoft 365 Business', 'M365-SCH-2025-IJKL', 10, '2025-03-01', '2026-02-28'),
    ('00000000-0000-0000-0000-000000000604', '00000000-0000-0000-0000-000000000102', NULL, 'Abacus Business', 'ABA-SCH-2025-MNOP', 3, '2025-01-01', '2025-12-31'),
    ('00000000-0000-0000-0000-000000000605', '00000000-0000-0000-0000-000000000103', '00000000-0000-0000-0000-000000000404', 'JetBrains All Products', 'JB-TECH-2025-QRST', 15, '2025-04-01', '2026-03-31'),
    ('00000000-0000-0000-0000-000000000606', '00000000-0000-0000-0000-000000000103', NULL, 'Slack Business+', 'SLK-TECH-2025-UVWX', 40, '2025-01-01', '2026-12-31')
ON CONFLICT (id) DO NOTHING;

-- customer_services
INSERT INTO customer_services (id, customer_id, service_id, customizations, notes) VALUES
    ('00000000-0000-0000-0000-000000000701', '00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000301', '{"sla": "8x5", "response_time_hours": 4}', 'Standard business hours support.'),
    ('00000000-0000-0000-0000-000000000702', '00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000302', '{"retention_days": 30, "storage_gb": 500}', 'Daily backups, 500 GB included.'),
    ('00000000-0000-0000-0000-000000000703', '00000000-0000-0000-0000-000000000102', '00000000-0000-0000-0000-000000000301', '{"sla": "24x7", "response_time_hours": 1}', 'Premium 24/7 support — law firm requires guaranteed uptime.'),
    ('00000000-0000-0000-0000-000000000704', '00000000-0000-0000-0000-000000000102', '00000000-0000-0000-0000-000000000303', '{"frequency": "quarterly", "include_pentest": true}', 'DSG compliance requirement, quarterly audits with penetration testing.')
ON CONFLICT (id) DO NOTHING;

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

DELETE FROM category_field_definitions WHERE id IN (
    '00000000-0000-0000-0000-000000000801',
    '00000000-0000-0000-0000-000000000802',
    '00000000-0000-0000-0000-000000000803',
    '00000000-0000-0000-0000-000000000804',
    '00000000-0000-0000-0000-000000000805',
    '00000000-0000-0000-0000-000000000806',
    '00000000-0000-0000-0000-000000000807',
    '00000000-0000-0000-0000-000000000808',
    '00000000-0000-0000-0000-000000000809',
    '00000000-0000-0000-0000-000000000810',
    '00000000-0000-0000-0000-000000000811',
    '00000000-0000-0000-0000-000000000812',
    '00000000-0000-0000-0000-000000000813',
    '00000000-0000-0000-0000-000000000814'
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
