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
