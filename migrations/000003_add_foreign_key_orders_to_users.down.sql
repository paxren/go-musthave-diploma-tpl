--
-- Удаление внешнего ключа для связи заказов с пользователями
-- Откат миграции добавления внешнего ключа
ALTER TABLE orders 
DROP CONSTRAINT IF EXISTS fk_orders_user_id;