-- ============================================
-- Sample Data untuk Whale Tracker v2.3 (Clean - No Redundancy)
-- ============================================

-- -------------------------------------------------
-- Insert Whales
-- -------------------------------------------------
INSERT INTO whales (address, label, telegram_chat_id) VALUES
  ('0x3f5CE5FBFe3E9af3971dD833D26bA9b5C936f0bE', 'Binance Cold Wallet', NULL),
  ('0x2B8f1b796eE9Ad21Ce014197E58dAeeB8666D091', 'Bitfinex Cold Wallet', NULL),
  ('0x71660c4005BA85c37ccec55d0C4493E66Fe775d3', 'Coinbase', NULL),
  ('0xdf0a15a997de4294e692995e239c0b86eec9d0b6', 'Kraken Cold Wallet', NULL),
  ('0x47ac0Fb4F2D84898e4D9E7b4DaB3C24507a6D503', 'Unknown Whale #1', NULL);

-- -------------------------------------------------
-- Insert Tokens (dengan coingecko_id)
-- -------------------------------------------------
INSERT INTO tokens (address, symbol, decimals, category, coingecko_id) VALUES
  ('0x0000000000000000000000000000000000000000', 'ETH', 18, 'native', 'ethereum'),
  ('0xdac17f958d2ee523a2206206994597c13d831ec7', 'USDT', 6, 'stablecoin', 'tether'),
  ('0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48', 'USDC', 6, 'stablecoin', 'usd-coin'),
  ('0x6b175474e89094c44da98b954eedeac495271d0f', 'DAI', 18, 'stablecoin', 'dai'),
  ('0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0', 'wstETH', 18, 'eth_derivative', 'wrapped-steth'),
  ('0xae78736cd615f374d3085123a210448e74fc6393', 'rETH', 18, 'eth_derivative', 'rocket-pool-eth'),
  ('0x1f9840a85d5af5bf1d1762f925bdaddc4201f984', 'UNI', 18, 'defi', 'uniswap'),
  ('0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2', 'WETH', 18, 'eth_derivative', 'weth');

-- -------------------------------------------------
-- Insert Token Prices (Latest only - ONE row per token)
-- -------------------------------------------------
INSERT INTO token_prices (token_address, price_usd, source, updated_at) VALUES
  ('0x0000000000000000000000000000000000000000', 3000.0, 'coingecko', '2026-01-04 10:00:00'),
  ('0xdac17f958d2ee523a2206206994597c13d831ec7', 1.0, 'coingecko', '2026-01-04 10:00:00'),
  ('0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48', 1.0, 'coingecko', '2026-01-04 10:00:00'),
  ('0x6b175474e89094c44da98b954eedeac495271d0f', 1.0, 'coingecko', '2026-01-04 10:00:00'),
  ('0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0', 3200.0, 'coingecko', '2026-01-04 10:00:00'),
  ('0xae78736cd615f374d3085123a210448e74fc6393', 3100.0, 'coingecko', '2026-01-04 10:00:00'),
  ('0x1f9840a85d5af5bf1d1762f925bdaddc4201f984', 8.0, 'coingecko', '2026-01-04 10:00:00'),
  ('0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2', 3000.0, 'coingecko', '2026-01-04 10:00:00');

-- -------------------------------------------------
-- Insert Balances (HANYA balance decimal, USD dihitung on-the-fly)
-- -------------------------------------------------
-- Binance Cold Wallet holdings (multi-token)
INSERT INTO balances (whale_address, token_address, balance_decimal,
                     prev_balance_decimal, change_percent, last_updated) VALUES
  ('0x3f5CE5FBFe3E9af3971dD833D26bA9b5C936f0bE', '0x0000000000000000000000000000000000000000',
   50000.0, 42500.0, 17.6, '2026-01-04 10:00:00'),
  ('0x3f5CE5FBFe3E9af3971dD833D26bA9b5C936f0bE', '0xdac17f958d2ee523a2206206994597c13d831ec7',
   10000.0, 9500.0, 5.3, '2026-01-04 10:00:00'),
  ('0x3f5CE5FBFe3E9af3971dD833D26bA9b5C936f0bE', '0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48',
   50000.0, 48000.0, 4.2, '2026-01-04 10:00:00'),
  ('0x3f5CE5FBFe3E9af3971dD833D26bA9b5C936f0bE', '0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0',
   3125.0, 3000.0, 4.2, '2026-01-04 10:00:00');

-- Bitfinex Cold Wallet holdings
INSERT INTO balances (whale_address, token_address, balance_decimal,
                     prev_balance_decimal, change_percent, last_updated) VALUES
  ('0x2B8f1b796eE9Ad21Ce014197E58dAeeB8666D091', '0x0000000000000000000000000000000000000000',
   30000.0, 31000.0, -3.2, '2026-01-04 10:00:00'),
  ('0x2B8f1b796eE9Ad21Ce014197E58dAeeB8666D091', '0xdac17f958d2ee523a2206206994597c13d831ec7',
   5000.0, 5000.0, 0.0, '2026-01-04 10:00:00');

-- Unknown Whale holdings (baru pertama kali track, tidak ada prev)
INSERT INTO balances (whale_address, token_address, balance_decimal,
                     prev_balance_decimal, change_percent, last_updated) VALUES
  ('0x47ac0Fb4F2D84898e4D9E7b4DaB3C24507a6D503', '0x0000000000000000000000000000000000000000',
   1000.0, NULL, NULL, '2026-01-04 10:00:00');
