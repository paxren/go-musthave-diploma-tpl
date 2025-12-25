--
-- Удаление внешнего ключа для связи заказов с пользователями
-- Откат миграции добавления внешнего ключа
ALTER TABLE gophermart_orders
DROP CONSTRAINT IF EXISTS fk_gophermart_orders_user_id;