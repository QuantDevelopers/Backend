from datetime import datetime
from pydantic import BaseModel, ConfigDict, Field


class HealthResponse(BaseModel):
    status: str = Field(..., example="healthy")
    app: str = Field(..., example="QuantBackend")
    database: str = Field(..., example="connected")
    timestamp: datetime


class UnderlyingAssetResponse(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    symbol_fa: str
    name_fa: str
    tsetmc_id: str | None = None
    market_type: str
    is_active: bool
    created_at: datetime | None = None


class UnderlyingListResponse(BaseModel):
    total: int
    items: list[UnderlyingAssetResponse]
