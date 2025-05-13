CREATE TABLE IF NOT EXISTS general_stats (
  total_orders INTEGER DEFAULT 0,
  active_orders INTEGER DEFAULT 0,
  completed_orders INTEGER DEFAULT 0,
  average_completion_time_seconds INTEGER DEFAULT 0
);

INSERT INTO general_stats (total_orders)
SELECT 0
WHERE NOT EXISTS (SELECT 1 FROM general_stats);
