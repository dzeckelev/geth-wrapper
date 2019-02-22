BEGIN TRANSACTION;

DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS outputs;

DROP TYPE IF EXISTS tx_status;

CREATE TYPE tx_status AS ENUM ('failed','successful');

CREATE TABLE accounts (
  id text PRIMARY KEY,
  balance text NOT NULL,
  public_key text NOT NULL
);

CREATE INDEX IF NOT EXISTS account_public_key ON accounts(public_key);

CREATE TABLE transactions (
  id text PRIMARY KEY,
  hash text NOT NULL,
  "from" text NOT NULL,
  "to" text NOT NULL,
  amount text NOT NULL,
  status tx_status,
  block bigint,
  timestamp  bigint,
  marked  bool,
  confirmations bigint NOT NULL
);

CREATE TABLE outputs (
  id text PRIMARY KEY,
  hash text NOT NULL,
  account text NOT NULL
);

CREATE INDEX IF NOT EXISTS tx_hash ON transactions(hash);
CREATE INDEX IF NOT EXISTS tx_from ON transactions ("from");
CREATE INDEX IF NOT EXISTS tx_to ON transactions ("to");

CREATE TABLE settings (
  key text PRIMARY KEY,
  value text NOT NULL
);

END TRANSACTION;