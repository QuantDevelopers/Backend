from dataclasses import dataclass
from datetime import datetime


@dataclass
class UnderlyingAssetEntity:
    id: int | None
    symbol_fa: str
    name_fa: str
    tsetmc_id: str | None
    market_type: str
    is_active: bool
    created_at: datetime | None = None
    updated_at: datetime | None = None
