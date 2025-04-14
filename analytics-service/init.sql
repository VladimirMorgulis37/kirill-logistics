CREATE TABLE IF NOT EXISTS order_stats (
  id SERIAL PRIMARY KEY,
  total_orders INTEGER,
  new_orders INTEGER,
  completed_orders INTEGER,
  calculated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
