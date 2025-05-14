CREATE TABLE IF NOT EXISTS orders (
    id VARCHAR(50) PRIMARY KEY,
    sender_name VARCHAR(255) NOT NULL,
    recipient_name VARCHAR(255) NOT NULL,
    email TEXT,
    address_from VARCHAR(255),
    address_to VARCHAR(255),
    status VARCHAR(50),
    created_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    weight DOUBLE PRECISION,
    length DOUBLE PRECISION,
    width DOUBLE PRECISION,
    height DOUBLE PRECISION,
    urgency INT
);

-- 1. Создание таблицы курьеров
CREATE TABLE IF NOT EXISTS couriers (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  phone TEXT,
  vehicle_type TEXT,
  status TEXT DEFAULT 'available',
  latitude DOUBLE PRECISION DEFAULT 0,
  longitude DOUBLE PRECISION DEFAULT 0,
  active_order_id TEXT
);

-- 2. Добавление ссылки на курьера в таблицу заказов
ALTER TABLE orders
  ADD COLUMN IF NOT EXISTS courier_id VARCHAR(50)
    REFERENCES couriers(id);
  ALTER COLUMN courier_id DROP NOT NULL;
