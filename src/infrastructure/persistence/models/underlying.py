from sqlalchemy import Boolean, Integer, String
from sqlalchemy.orm import Mapped, mapped_column

from src.infrastructure.database.base import Base, TimestampMixin


class UnderlyingAssetModel(Base, TimestampMixin):
    __tablename__ = "underlying_assets"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    symbol_fa: Mapped[str] = mapped_column(String(50), unique=True, index=True, nullable=False)
    name_fa: Mapped[str] = mapped_column(String(150), nullable=False)
    tsetmc_id: Mapped[str | None] = mapped_column(String(50), unique=True, index=True, nullable=True)
    market_type: Mapped[str] = mapped_column(String(50), default="Bourse", nullable=False)
    is_active: Mapped[bool] = mapped_column(Boolean, default=True, nullable=False)

    def __repr__(self) -> str:
        return f"<UnderlyingAsset(id={self.id}, symbol_fa='{self.symbol_fa}', name_fa='{self.name_fa}')>"
