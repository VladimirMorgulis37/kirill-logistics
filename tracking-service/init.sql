CREATE TABLE IF NOT EXISTS tracking_info (
  order_id VARCHAR(50) PRIMARY KEY,
  courier_id VARCHAR(50),
  status VARCHAR(50),
  latitude DOUBLE PRECISION,
  longitude DOUBLE PRECISION,
  updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS courier_tracking (
  courier_id    TEXT PRIMARY KEY,
  status        TEXT NOT NULL,
  latitude      DOUBLE PRECISION NOT NULL,
  longitude     DOUBLE PRECISION NOT NULL,
  updated_at    TIMESTAMP WITHOUT TIME ZONE NOT NULL
);
