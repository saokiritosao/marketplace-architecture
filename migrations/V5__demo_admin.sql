-- Пароль только для локальной демонстрации: admin123
INSERT INTO users(email, password_hash, role)
VALUES (
    'admin@marketplace.local',
    '$2a$10$ftQOZZUzpy0ebjgrAiULJ.1R1kgMfGGSpeF7vZm52Kz6HX5yuPiXC',
    'ADMIN'
)
ON CONFLICT (email) DO NOTHING;
