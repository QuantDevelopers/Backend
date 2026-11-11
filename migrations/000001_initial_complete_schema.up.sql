-- Initial complete schema for QuantBackend (PostgreSQL 16)
-- Matches the exact ERD & Alembic schema specification (Card E3-1).

CREATE TABLE IF NOT EXISTS underlying_assets (
    id SERIAL PRIMARY KEY,
    symbol_fa VARCHAR(50) NOT NULL,
    name_fa VARCHAR(150) NOT NULL,
    tsetmc_id VARCHAR(50),
    isin VARCHAR(50),
    market_type VARCHAR(50) NOT NULL DEFAULT 'Bourse',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE underlying_assets ADD COLUMN IF NOT EXISTS isin VARCHAR(50);

CREATE UNIQUE INDEX IF NOT EXISTS ix_underlying_assets_symbol_fa ON underlying_assets (symbol_fa);
CREATE UNIQUE INDEX IF NOT EXISTS ix_underlying_assets_tsetmc_id ON underlying_assets (tsetmc_id);
CREATE UNIQUE INDEX IF NOT EXISTS ix_underlying_assets_isin ON underlying_assets (isin);

CREATE TABLE IF NOT EXISTS option_contracts (
    id SERIAL PRIMARY KEY,
    underlying_asset_id INTEGER NOT NULL REFERENCES underlying_assets (id) ON DELETE CASCADE,
    symbol_fa VARCHAR(50) NOT NULL,
    name_fa VARCHAR(150) NOT NULL,
    contract_type VARCHAR(10) NOT NULL,
    strike_price NUMERIC(15, 2) NOT NULL,
    contract_size INTEGER NOT NULL DEFAULT 1000,
    expiration_date DATE NOT NULL,
    begin_date DATE,
    tsetmc_id VARCHAR(50),
    isin VARCHAR(50),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT check_contract_type CHECK (contract_type IN ('CALL', 'PUT'))
);

CREATE INDEX IF NOT EXISTS ix_option_contracts_underlying_asset_id ON option_contracts (underlying_asset_id);
CREATE UNIQUE INDEX IF NOT EXISTS ix_option_contracts_symbol_fa ON option_contracts (symbol_fa);
CREATE INDEX IF NOT EXISTS ix_option_contracts_contract_type ON option_contracts (contract_type);
CREATE INDEX IF NOT EXISTS ix_option_contracts_strike_price ON option_contracts (strike_price);
CREATE INDEX IF NOT EXISTS ix_option_contracts_expiration_date ON option_contracts (expiration_date);
CREATE UNIQUE INDEX IF NOT EXISTS ix_option_contracts_tsetmc_id ON option_contracts (tsetmc_id);
CREATE UNIQUE INDEX IF NOT EXISTS ix_option_contracts_isin ON option_contracts (isin);
CREATE INDEX IF NOT EXISTS ix_option_contracts_is_active ON option_contracts (is_active);
CREATE INDEX IF NOT EXISTS ix_option_ua_expiry ON option_contracts (underlying_asset_id, expiration_date);
CREATE INDEX IF NOT EXISTS ix_option_ua_type_strike ON option_contracts (underlying_asset_id, contract_type, strike_price);

CREATE TABLE IF NOT EXISTS daily_history (
    id BIGSERIAL PRIMARY KEY,
    instrument_type VARCHAR(20) NOT NULL,
    instrument_id INTEGER NOT NULL,
    trade_date DATE NOT NULL,
    open_price NUMERIC(15, 2) NOT NULL,
    high_price NUMERIC(15, 2) NOT NULL,
    low_price NUMERIC(15, 2) NOT NULL,
    close_price NUMERIC(15, 2) NOT NULL,
    yesterday_price NUMERIC(15, 2),
    settlement_price NUMERIC(15, 2),
    volume BIGINT NOT NULL DEFAULT 0,
    value NUMERIC(20, 2) NOT NULL DEFAULT 0.0,
    trade_count INTEGER NOT NULL DEFAULT 0,
    open_interest INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT check_daily_instrument_type CHECK (instrument_type IN ('UNDERLYING', 'OPTION')),
    CONSTRAINT uq_daily_instrument_date UNIQUE (instrument_type, instrument_id, trade_date)
);

CREATE INDEX IF NOT EXISTS ix_daily_history_instrument_id ON daily_history (instrument_id);
CREATE INDEX IF NOT EXISTS ix_daily_history_instrument_type ON daily_history (instrument_type);
CREATE INDEX IF NOT EXISTS ix_daily_history_trade_date ON daily_history (trade_date);
CREATE INDEX IF NOT EXISTS ix_daily_inst_date ON daily_history (instrument_type, instrument_id, trade_date);

CREATE TABLE IF NOT EXISTS ticks_raw (
    id BIGSERIAL PRIMARY KEY,
    instrument_type VARCHAR(20) NOT NULL,
    instrument_id INTEGER NOT NULL,
    trade_date DATE NOT NULL,
    trade_time TIME NOT NULL,
    sequence_number INTEGER,
    price NUMERIC(15, 2) NOT NULL,
    volume BIGINT NOT NULL,
    raw_payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT check_raw_instrument_type CHECK (instrument_type IN ('UNDERLYING', 'OPTION'))
);

CREATE INDEX IF NOT EXISTS ix_ticks_raw_instrument_id ON ticks_raw (instrument_id);
CREATE INDEX IF NOT EXISTS ix_ticks_raw_instrument_type ON ticks_raw (instrument_type);
CREATE INDEX IF NOT EXISTS ix_ticks_raw_trade_date ON ticks_raw (trade_date);
CREATE INDEX IF NOT EXISTS ix_ticks_raw_trade_time ON ticks_raw (trade_time);
CREATE INDEX IF NOT EXISTS ix_raw_ticks_lookup ON ticks_raw (instrument_type, instrument_id, trade_date, trade_time);

CREATE TABLE IF NOT EXISTS ticks_clean (
    id BIGSERIAL PRIMARY KEY,
    instrument_type VARCHAR(20) NOT NULL,
    instrument_id INTEGER NOT NULL,
    trade_date DATE NOT NULL,
    trade_timestamp TIMESTAMPTZ NOT NULL,
    sequence_number INTEGER NOT NULL,
    price NUMERIC(15, 2) NOT NULL,
    volume BIGINT NOT NULL,
    is_canceled BOOLEAN NOT NULL DEFAULT FALSE,
    is_outlier BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT check_clean_instrument_type CHECK (instrument_type IN ('UNDERLYING', 'OPTION')),
    CONSTRAINT uq_clean_tick_seq UNIQUE (instrument_type, instrument_id, trade_date, sequence_number)
);

CREATE INDEX IF NOT EXISTS ix_ticks_clean_instrument_id ON ticks_clean (instrument_id);
CREATE INDEX IF NOT EXISTS ix_ticks_clean_instrument_type ON ticks_clean (instrument_type);
CREATE INDEX IF NOT EXISTS ix_ticks_clean_trade_date ON ticks_clean (trade_date);
CREATE INDEX IF NOT EXISTS ix_ticks_clean_trade_timestamp ON ticks_clean (trade_timestamp);
CREATE INDEX IF NOT EXISTS ix_ticks_clean_sequence_number ON ticks_clean (sequence_number);
CREATE INDEX IF NOT EXISTS ix_clean_ticks_ts ON ticks_clean (instrument_type, instrument_id, trade_timestamp);

CREATE TABLE IF NOT EXISTS data_quality_logs (
    id BIGSERIAL PRIMARY KEY,
    instrument_type VARCHAR(20) NOT NULL,
    instrument_id INTEGER NOT NULL,
    trade_date DATE NOT NULL,
    check_category VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,
    gap_count INTEGER NOT NULL DEFAULT 0,
    details JSONB,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT check_quality_status CHECK (status IN ('PASSED', 'WARNING', 'FAILED'))
);

CREATE INDEX IF NOT EXISTS ix_data_quality_logs_check_category ON data_quality_logs (check_category);
CREATE INDEX IF NOT EXISTS ix_data_quality_logs_instrument_id ON data_quality_logs (instrument_id);
CREATE INDEX IF NOT EXISTS ix_data_quality_logs_instrument_type ON data_quality_logs (instrument_type);
CREATE INDEX IF NOT EXISTS ix_data_quality_logs_status ON data_quality_logs (status);
CREATE INDEX IF NOT EXISTS ix_data_quality_logs_trade_date ON data_quality_logs (trade_date);
CREATE INDEX IF NOT EXISTS ix_quality_inst_date ON data_quality_logs (instrument_type, instrument_id, trade_date);
