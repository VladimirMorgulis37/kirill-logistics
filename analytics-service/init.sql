CREATE TABLE IF NOT EXISTS general_stats (
  total_orders INTEGER DEFAULT 0,
  active_orders INTEGER DEFAULT 0,
  completed_orders INTEGER DEFAULT 0,
  average_completion_time_seconds INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS courier_stats (
    courier_id TEXT PRIMARY KEY,
    courier_name TEXT,
    completed_orders INTEGER NOT NULL DEFAULT 0,
    total_revenue NUMERIC NOT NULL DEFAULT 0,
    average_delivery_time_sec NUMERIC NOT NULL DEFAULT 0
);

INSERT INTO general_stats (total_orders)
SELECT 0
WHERE NOT EXISTS (SELECT 1 FROM general_stats);
