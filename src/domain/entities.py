from dataclasses import dataclass
from datetime import date, datetime, time


@dataclass
class UnderlyingAssetEntity:
    id: int | None
    symbol_fa: str
    name_fa: str
    tsetmc_id: str | None
    isin: str | None = None
    market_type: str = "Bourse"
    is_active: bool = True
    created_at: datetime | None = None
    updated_at: datetime | None = None


@dataclass
class OptionContractEntity:
    id: int | None
    underlying_asset_id: int
    symbol_fa: str
    name_fa: str
    contract_type: str  # 'CALL' or 'PUT'
    strike_price: float
    contract_size: int
    expiration_date: date
    begin_date: date | None = None
    tsetmc_id: str | None = None
    isin: str | None = None
    is_active: bool = True
    status: str = "ACTIVE"
    created_at: datetime | None = None
    updated_at: datetime | None = None


@dataclass
class DailyHistoryEntity:
    id: int | None
    instrument_type: str  # 'UNDERLYING' or 'OPTION'
    instrument_id: int
    trade_date: date
    open_price: float
    high_price: float
    low_price: float
    close_price: float
    yesterday_price: float | None = None
    settlement_price: float | None = None
    volume: int = 0
    value: float = 0.0
    trade_count: int = 0
    open_interest: int | None = None
    created_at: datetime | None = None


@dataclass
class TickRawEntity:
    id: int | None
    instrument_type: str
    instrument_id: int
    trade_date: date
    trade_time: time
    price: float
    volume: int
    sequence_number: int | None = None
    raw_payload: dict | None = None
    created_at: datetime | None = None


@dataclass
class TickCleanEntity:
    id: int | None
    instrument_type: str
    instrument_id: int
    trade_date: date
    trade_timestamp: datetime
    sequence_number: int
    price: float
    volume: int
    is_canceled: bool = False
    is_outlier: bool = False
    created_at: datetime | None = None


@dataclass
class DataQualityLogEntity:
    id: int | None
    instrument_type: str
    instrument_id: int
    trade_date: date
    check_category: str
    status: str
    gap_count: int = 0
    details: dict | None = None
    notes: str | None = None
    created_at: datetime | None = None
