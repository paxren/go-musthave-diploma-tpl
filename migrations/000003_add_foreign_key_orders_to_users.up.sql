--
-- Добавление внешнего ключа для связи заказов с пользователями
-- Устанавливает ссылочную целостность между таблицами orders и users
ALTER TABLE gophermart_orders
ADD CONSTRAINT fk_gophermart_orders_user_id
FOREIGN KEY (user_id) REFERENCES gophermart_users(id) ON DELETE CASCADE;

-- Добавление комментария для документации
COMMENT ON CONSTRAINT fk_gophermart_orders_user_id ON gophermart_orders IS 'Связь заказа с пользователем';