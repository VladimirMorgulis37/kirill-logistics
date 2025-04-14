CREATE TABLE IF NOT EXISTS orders (
  id VARCHAR(50) PRIMARY KEY,
  sender_name VARCHAR(255) NOT NULL,
  recipient_name VARCHAR(255) NOT NULL,
  address_from VARCHAR(255),
  address_to VARCHAR(255),
  status VARCHAR(50),
  created_at TIMESTAMP NOT NULL
);
