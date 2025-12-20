--
-- Добавление внешнего ключа для связи заказов с пользователями
-- Устанавливает ссылочную целостность между таблицами orders и users
ALTER TABLE orders 
ADD CONSTRAINT fk_orders_user_id 
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- Добавление комментария для документации
COMMENT ON CONSTRAINT fk_orders_user_id ON orders IS 'Связь заказа с пользователем';