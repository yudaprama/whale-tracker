-- ============================================
-- Whale Tracker - Schema v2.3 (Clean - No Redundancy)
-- Focus: Balance Tracking + Latest Price + Telegram Alert
-- Database: DuckDB
-- ============================================

-- -------------------------------------------------
-- 1. WHALES: Master list whale yang di-track
-- -------------------------------------------------
CREATE OR REPLACE TABLE whales (
  address VARCHAR(42) PRIMARY KEY,
  label VARCHAR(255) NOT NULL,
  telegram_chat_id VARCHAR(50),
  active BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT NOW()
);

-- -------------------------------------------------
-- 2. TOKENS: Master list token
-- -------------------------------------------------
CREATE OR REPLACE TABLE tokens (
  address VARCHAR(42) PRIMARY KEY,
  symbol VARCHAR(32) NOT NULL,
  decimals INT NOT NULL,
  category VARCHAR(50),
  coingecko_id VARCHAR(50)
);

-- -------------------------------------------------
-- 3. TOKEN_PRICES: Latest price only (ONE row per token)
--    Single source of truth untuk harga
-- -------------------------------------------------
CREATE OR REPLACE TABLE token_prices (
  token_address VARCHAR(42) PRIMARY KEY,
  price_usd DOUBLE NOT NULL,
  source VARCHAR(50),
  updated_at TIMESTAMP DEFAULT NOW(),

  FOREIGN KEY (token_address) REFERENCES tokens(address)
);

-- -------------------------------------------------
-- 4. BALANCES: Current balance per whale per token
--    HANYA simpan balance, price diambil dari token_prices
-- -------------------------------------------------
CREATE OR REPLACE TABLE balances (
  whale_address VARCHAR(42) NOT NULL,
  token_address VARCHAR(42) NOT NULL,

  -- Balance data ONLY
  balance_decimal DOUBLE NOT NULL,

  -- Previous data (untuk deteksi perubahan)
  prev_balance_decimal DOUBLE,
  change_percent DOUBLE,

  -- Metadata
  last_updated TIMESTAMP NOT NULL,

  PRIMARY KEY (whale_address, token_address),
  FOREIGN KEY (whale_address) REFERENCES whales(address),
  FOREIGN KEY (token_address) REFERENCES tokens(address)
);

-- -------------------------------------------------
-- VIEW: Latest holdings (auto-calculate USD value)
-- -------------------------------------------------
CREATE OR REPLACE VIEW latest_holdings AS
SELECT
  w.label as whale,
  t.symbol as token,
  t.category,
  b.balance_decimal,
  tp.price_usd as token_price,
  (b.balance_decimal * tp.price_usd) as usd_value,
  b.change_percent,
  b.last_updated
FROM balances b
JOIN whales w ON b.whale_address = w.address
JOIN tokens t ON b.token_address = t.address
JOIN token_prices tp ON b.token_address = tp.token_address
WHERE b.balance_decimal > 0
ORDER BY usd_value DESC;

-- -------------------------------------------------
-- VIEW: Whales dengan perubahan besar (alert candidates)
-- -------------------------------------------------
CREATE OR REPLACE VIEW big_changes AS
SELECT
  w.label as whale,
  w.telegram_chat_id,
  t.symbol as token,
  b.balance_decimal,
  b.prev_balance_decimal,
  b.change_percent,
  tp.price_usd as token_price,
  (b.balance_decimal * tp.price_usd) as usd_value,
  b.last_updated
FROM balances b
JOIN whales w ON b.whale_address = w.address
JOIN tokens t ON b.token_address = t.address
JOIN token_prices tp ON b.token_address = tp.token_address
WHERE b.change_percent IS NOT NULL
  AND ABS(b.change_percent) >= 10
ORDER BY ABS(b.change_percent) DESC;
