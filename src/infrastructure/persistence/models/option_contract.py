from sqlalchemy import Boolean, CheckConstraint, Date, ForeignKey, Index, Integer, Numeric, String
from sqlalchemy.orm import Mapped, mapped_column, relationship

from src.infrastructure.database.base import Base, TimestampMixin


class OptionContractModel(Base, TimestampMixin):
    __tablename__ = "option_contracts"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    underlying_asset_id: Mapped[int] = mapped_column(
        Integer,
        ForeignKey("underlying_assets.id", ondelete="CASCADE"),
        nullable=False,
        index=True,
    )
    symbol_fa: Mapped[str] = mapped_column(String(50), unique=True, index=True, nullable=False)
    name_fa: Mapped[str] = mapped_column(String(150), nullable=False)
    contract_type: Mapped[str] = mapped_column(String(10), nullable=False, index=True)  # 'CALL' or 'PUT'
    strike_price: Mapped[float] = mapped_column(Numeric(15, 2), nullable=False, index=True)
    contract_size: Mapped[int] = mapped_column(Integer, default=1000, nullable=False)
    expiration_date: Mapped[Date] = mapped_column(Date, nullable=False, index=True)
    begin_date: Mapped[Date | None] = mapped_column(Date, nullable=True)
    tsetmc_id: Mapped[str | None] = mapped_column(String(50), unique=True, index=True, nullable=True)
    isin: Mapped[str | None] = mapped_column(String(50), unique=True, index=True, nullable=True)
    is_active: Mapped[bool] = mapped_column(Boolean, default=True, nullable=False, index=True)
    status: Mapped[str] = mapped_column(String(20), default="ACTIVE", nullable=False)  # ACTIVE, EXPIRED, DELISTED

    # Relationships
    underlying_asset = relationship("UnderlyingAssetModel", back_populates="option_contracts")

    __table_args__ = (
        CheckConstraint("contract_type IN ('CALL', 'PUT')", name="check_contract_type"),
        Index("ix_option_ua_expiry", "underlying_asset_id", "expiration_date"),
        Index("ix_option_ua_type_strike", "underlying_asset_id", "contract_type", "strike_price"),
    )

    def __repr__(self) -> str:
        return (
            f"<OptionContract(id={self.id}, symbol_fa='{self.symbol_fa}', "
            f"type='{self.contract_type}', strike={self.strike_price}, expiry='{self.expiration_date}')>"
        )
