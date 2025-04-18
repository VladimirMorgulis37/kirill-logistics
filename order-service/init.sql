CREATE TABLE IF NOT EXISTS orders (
    id VARCHAR(50) PRIMARY KEY,
    sender_name VARCHAR(255) NOT NULL,
    recipient_name VARCHAR(255) NOT NULL,
    address_from VARCHAR(255),
    address_to VARCHAR(255),
    status VARCHAR(50),
    created_at TIMESTAMP NOT NULL,
    weight DOUBLE PRECISION,
    length DOUBLE PRECISION,
    width DOUBLE PRECISION,
    height DOUBLE PRECISION,
    urgency INT
);

-- 1. Создание таблицы курьеров
CREATE TABLE IF NOT EXISTS couriers (
  id VARCHAR(50) PRIMARY KEY,
  name VARCHAR(255) NOT NULL
);

-- 2. Добавление ссылки на курьера в таблицу заказов
ALTER TABLE orders
  ADD COLUMN IF NOT EXISTS courier_id VARCHAR(50)
    REFERENCES couriers(id);
