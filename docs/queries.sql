-- ============================================
-- Useful Queries for Whale Tracker v2.3 (Clean - No Redundancy)
-- ============================================

-- -------------------------------------------------
-- QUERY 1: Get ALL tokens held by whale (USD dihitung on-the-fly)
-- -------------------------------------------------
SELECT * FROM latest_holdings WHERE whale = 'Binance Cold Wallet';

-- -------------------------------------------------
-- QUERY 2: Total portfolio per whale (USD dihitung on-the-fly)
-- -------------------------------------------------
SELECT
  w.label as whale,
  SUM(b.balance_decimal * tp.price_usd) as total_portfolio_usd,
  COUNT(*) as token_count,
  SUM(CASE WHEN t.category = 'stablecoin' THEN b.balance_decimal * tp.price_usd ELSE 0 END) as stablecoin_value,
  SUM(CASE WHEN t.symbol = 'ETH' THEN b.balance_decimal * tp.price_usd ELSE 0 END) as eth_value
FROM balances b
JOIN whales w ON b.whale_address = w.address
JOIN tokens t ON b.token_address = t.address
JOIN token_prices tp ON b.token_address = tp.token_address
WHERE b.balance_decimal > 0
GROUP BY w.label
ORDER BY total_portfolio_usd DESC;

-- -------------------------------------------------
-- QUERY 3: Get big changes (alert candidates)
-- -------------------------------------------------
SELECT * FROM big_changes;

-- -------------------------------------------------
-- QUERY 4: Latest prices (semua token)
-- -------------------------------------------------
SELECT
  t.symbol as token,
  tp.price_usd,
  tp.updated_at
FROM token_prices tp
JOIN tokens t ON tp.token_address = t.address
ORDER BY tp.price_usd DESC;

-- -------------------------------------------------
-- QUERY 5: Top tokens by whale holdings
-- -------------------------------------------------
SELECT
  t.symbol as token,
  t.category,
  COUNT(DISTINCT b.whale_address) as holder_count,
  SUM(b.balance_decimal * tp.price_usd) as total_usd_held,
  AVG(b.balance_decimal * tp.price_usd) as avg_usd_per_whale
FROM balances b
JOIN tokens t ON b.token_address = t.address
JOIN token_prices tp ON b.token_address = tp.token_address
WHERE b.balance_decimal > 0
GROUP BY t.symbol, t.category
ORDER BY total_usd_held DESC;

-- -------------------------------------------------
-- QUERY 6: Whale vs Whale comparison
-- -------------------------------------------------
WITH whale_totals AS (
  SELECT
    w.label as whale,
    SUM(b.balance_decimal * tp.price_usd) as total_value
  FROM balances b
  JOIN whales w ON b.whale_address = w.address
  JOIN token_prices tp ON b.token_address = tp.token_address
  WHERE b.balance_decimal > 0
  GROUP BY w.label
)
SELECT
  a.whale as whale_a,
  b.whale as whale_b,
  a.total_value as value_a,
  b.total_value as value_b,
  (a.total_value - b.total_value) as difference,
  ((a.total_value - b.total_value) / b.total_value * 100) as percent_diff
FROM whale_totals a
CROSS JOIN whale_totals b
WHERE a.whale < b.whale
ORDER BY ABS(percent_diff) DESC;

-- -------------------------------------------------
-- QUERY 7: All whales with their top token
-- -------------------------------------------------
SELECT
  w.label as whale,
  FIRST(t.symbol ORDER BY b.balance_decimal * tp.price_usd DESC) as top_token,
  FIRST(b.balance_decimal * tp.price_usd ORDER BY b.balance_decimal * tp.price_usd DESC) as top_token_value,
  SUM(b.balance_decimal * tp.price_usd) as total_portfolio
FROM balances b
JOIN whales w ON b.whale_address = w.address
JOIN tokens t ON b.token_address = t.address
JOIN token_prices tp ON b.token_address = tp.token_address
WHERE b.balance_decimal > 0
GROUP BY w.label
ORDER BY total_portfolio DESC;

-- -------------------------------------------------
-- QUERY 8: Whales with specific token
-- -------------------------------------------------
SELECT
  w.label as whale,
  b.balance_decimal,
  (b.balance_decimal * tp.price_usd) as usd_value,
  b.change_percent
FROM balances b
JOIN whales w ON b.whale_address = w.address
JOIN tokens t ON b.token_address = t.address
JOIN token_prices tp ON b.token_address = tp.token_address
WHERE t.symbol = 'USDT'
  AND b.balance_decimal > 0
ORDER BY usd_value DESC;

-- -------------------------------------------------
-- QUERY 9: Whales by category (stablecoin vs other)
-- -------------------------------------------------
SELECT
  w.label as whale,
  SUM(CASE WHEN t.category = 'stablecoin' THEN b.balance_decimal * tp.price_usd ELSE 0 END) as stablecoin_usd,
  SUM(CASE WHEN t.category != 'stablecoin' THEN b.balance_decimal * tp.price_usd ELSE 0 END) as other_usd,
  SUM(b.balance_decimal * tp.price_usd) as total_usd,
  (SUM(CASE WHEN t.category = 'stablecoin' THEN b.balance_decimal * tp.price_usd ELSE 0 END) /
   SUM(b.balance_decimal * tp.price_usd) * 100) as stablecoin_percent
FROM balances b
JOIN whales w ON b.whale_address = w.address
JOIN tokens t ON b.token_address = t.address
JOIN token_prices tp ON b.token_address = tp.token_address
WHERE b.balance_decimal > 0
GROUP BY w.label
ORDER BY total_usd DESC;

-- -------------------------------------------------
-- QUERY 10: Token distribution across all whales
-- -------------------------------------------------
SELECT
  t.symbol as token,
  t.category,
  SUM(b.balance_decimal * tp.price_usd) as total_usd,
  COUNT(DISTINCT b.whale_address) as whale_count,
  AVG(b.balance_decimal) as avg_balance_per_whale
FROM balances b
JOIN tokens t ON b.token_address = t.address
JOIN token_prices tp ON b.token_address = tp.token_address
WHERE b.balance_decimal > 0
GROUP BY t.symbol, t.category
ORDER BY total_usd DESC;

-- -------------------------------------------------
-- QUERY 11: Balances without price (missing token_prices)
-- -------------------------------------------------
SELECT
  w.label as whale,
  t.symbol as token,
  b.balance_decimal
FROM balances b
JOIN whales w ON b.whale_address = w.address
JOIN tokens t ON b.token_address = t.address
LEFT JOIN token_prices tp ON b.token_address = tp.token_address
WHERE tp.price_usd IS NULL;

-- -------------------------------------------------
-- QUERY 12: Tokens without price
-- -------------------------------------------------
SELECT
  t.symbol,
  t.coingecko_id
FROM tokens t
LEFT JOIN token_prices tp ON t.address = tp.token_address
WHERE tp.price_usd IS NULL;

-- -------------------------------------------------
-- QUERY 13: Update balance dengan prev_balance tracking
-- -------------------------------------------------
-- Contoh: UPDATE saat balance berubah
-- Step 1: Simpan old balance
-- Step 2: Update new balance + calculate change_percent

-- Untuk whale '0x...' dan token '0x...':
INSERT INTO balances (whale_address, token_address, balance_decimal, prev_balance_decimal, change_percent, last_updated)
VALUES ('0x...', '0x...', 60000.0,
        (SELECT balance_decimal FROM balances WHERE whale_address = '0x...' AND token_address = '0x...'),
        ((60000.0 - (SELECT balance_decimal FROM balances WHERE whale_address = '0x...' AND token_address = '0x...')) /
         (SELECT balance_decimal FROM balances WHERE whale_address = '0x...' AND token_address = '0x...') * 100),
        NOW())
ON CONFLICT (whale_address, token_address) DO UPDATE SET
  balance_decimal = excluded.balance_decimal,
  prev_balance_decimal = excluded.prev_balance_decimal,
  change_percent = excluded.change_percent,
  last_updated = excluded.last_updated;
