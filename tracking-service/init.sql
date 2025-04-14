CREATE TABLE IF NOT EXISTS tracking_info (
  order_id VARCHAR(50) PRIMARY KEY,
  courier_id VARCHAR(50),
  status VARCHAR(50),
  updated_at TIMESTAMP NOT NULL
);
