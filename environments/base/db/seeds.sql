-- Seed data for development

-- Create test users
INSERT INTO users (email, name, password) VALUES
    ('admin@example.com', 'Admin User', '$2a$10$example.hash'),
    ('test@example.com', 'Test User', '$2a$10$example.hash'),
    ('demo@example.com', 'Demo User', '$2a$10$example.hash')
ON CONFLICT (email) DO NOTHING;

-- Create some audit entries
INSERT INTO audit_log (user_id, action, entity_type, entity_id, details) VALUES
    (1, 'login', 'session', NULL, '{"ip": "127.0.0.1"}'::jsonb),
    (2, 'create', 'project', 1, '{"name": "Test Project"}'::jsonb),
    (3, 'update', 'project', 1, '{"field": "name", "value": "Updated Project"}'::jsonb)
ON CONFLICT DO NOTHING;

-- Kopplingar mellan kommunens resurser och MATCHis banor.
-- Matchi-anläggningen "Baldershallen" (id 531) har hallerna 1-5 som
-- "Idrottshall 1-5", squashbanorna som "1. Squash"-"5. Squash" och
-- coworkingrummet som "Coworking".
INSERT INTO court_mappings (kommun_resource_id, kommun_resource_name, matchi_court_id, matchi_court_name) VALUES
    (628, 'Baldershallen 1', 11344, '1. Idrottshall'),
    (629, 'Baldershallen 2', 11345, '2. Idrottshall'),
    (630, 'Baldershallen 3', 12486, '3. Idrottshall'),
    (631, 'Baldershallen 4', 12487, '4. Idrottshall'),
    (632, 'Baldershallen 5', 12488, '5. Idrottshall'),
    (633, 'Baldershallen Squashbana 1', 3742, '1. Squash'),
    (634, 'Baldershallen Squashbana 2', 11594, '2. Squash'),
    (635, 'Baldershallen Squashbana 3', 11595, '3. Squash'),
    (636, 'Baldershallen Squashbana 4', 11596, '4. Squash'),
    (637, 'Baldershallen Squashbana 5', 11598, '5. Squash'),
    (751, 'Baldershallen Coworking-rum', 16157, 'Coworking (Endast klubben)')
ON CONFLICT (kommun_resource_id, matchi_court_id) DO NOTHING;
