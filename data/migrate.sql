BEGIN TRANSACTION;

DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS settings;

CREATE TABLE accounts (
  id text PRIMARY KEY,
  balance text NOT NULL,
  public_key text NOT NULL,
  private_key json,
  password text, -- Password must not be empty.
  last_update timestamp with time zone
);

CREATE INDEX IF NOT EXISTS account_public_key ON accounts(public_key);
CREATE INDEX IF NOT EXISTS account_last_update ON accounts(last_update);

CREATE TABLE transactions (
  id text PRIMARY KEY,
  hash text NOT NULL,
  "from" text NOT NULL,
  "to" text,
  amount text NOT NULL,
  status bigint,
  receipt_block bigint,
  timestamp  bigint
);

CREATE INDEX IF NOT EXISTS tx_hash ON transactions(hash);
CREATE INDEX IF NOT EXISTS tx_from ON transactions ("from");
CREATE INDEX IF NOT EXISTS tx_to ON transactions ("to");

CREATE TABLE settings (
  key text PRIMARY KEY,
  value text NOT NULL
);

END TRANSACTION;