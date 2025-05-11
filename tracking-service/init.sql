CREATE TABLE IF NOT EXISTS courier_tracking (
  courier_id    TEXT PRIMARY KEY,
  latitude      DOUBLE PRECISION NOT NULL,
  longitude     DOUBLE PRECISION NOT NULL,
  updated_at    TIMESTAMP WITHOUT TIME ZONE NOT NULL
);
