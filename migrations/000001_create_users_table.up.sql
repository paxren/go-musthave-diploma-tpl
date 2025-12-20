--
-- Создание таблицы пользователей
-- Таблица для хранения учетных записей пользователей системы
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    login VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создание индекса для быстрого поиска по логину
CREATE INDEX idx_users_login ON users(login);