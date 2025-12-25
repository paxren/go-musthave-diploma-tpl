--
-- Создание таблицы пользователей
-- Таблица для хранения учетных записей пользователей системы
CREATE TABLE gophermart_users (
    id SERIAL PRIMARY KEY,
    login VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создание индекса для быстрого поиска по логину
CREATE INDEX idx_gophermart_users_login ON gophermart_users(login);