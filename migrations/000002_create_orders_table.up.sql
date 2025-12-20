--
-- Создание таблицы заказов
-- Таблица для хранения заказов и списаний пользователей
CREATE TABLE gophermart_orders (
    id VARCHAR(255) PRIMARY KEY,
    user_id INTEGER NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('ORDER', 'WITHDRAW')),
    status VARCHAR(20) NOT NULL CHECK (status IN ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED')),
    value BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создание индексов для быстрого поиска
CREATE INDEX idx_gophermart_orders_user_id ON gophermart_orders(user_id);
CREATE INDEX idx_gophermart_orders_status ON gophermart_orders(status);
CREATE INDEX idx_gophermart_orders_type ON gophermart_orders(type);
CREATE INDEX idx_gophermart_orders_user_status ON gophermart_orders(user_id, status);