from datetime import datetime

from pydantic import BaseModel, ConfigDict, Field


class HealthResponse(BaseModel):
    status: str = Field(..., json_schema_extra={"example": "healthy"})
    app: str = Field(..., json_schema_extra={"example": "QuantBackend"})
    database: str = Field(..., json_schema_extra={"example": "connected"})
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
