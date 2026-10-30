from datetime import date

from sqlalchemy import (
    BigInteger,
    CheckConstraint,
    Date,
    Index,
    Integer,
    Numeric,
    String,
    UniqueConstraint,
)
from sqlalchemy.orm import Mapped, mapped_column

from src.infrastructure.database.base import Base, TimestampMixin


class DailyHistoryModel(Base, TimestampMixin):
    __tablename__ = "daily_history"

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True, autoincrement=True)
    instrument_type: Mapped[str] = mapped_column(String(20), nullable=False, index=True)  # 'UNDERLYING' or 'OPTION'
    instrument_id: Mapped[int] = mapped_column(Integer, nullable=False, index=True)
    trade_date: Mapped[date] = mapped_column(Date, nullable=False, index=True)

    open_price: Mapped[float] = mapped_column(Numeric(15, 2), nullable=False)
    high_price: Mapped[float] = mapped_column(Numeric(15, 2), nullable=False)
    low_price: Mapped[float] = mapped_column(Numeric(15, 2), nullable=False)
    close_price: Mapped[float] = mapped_column(Numeric(15, 2), nullable=False)
    yesterday_price: Mapped[float | None] = mapped_column(Numeric(15, 2), nullable=True)
    settlement_price: Mapped[float | None] = mapped_column(Numeric(15, 2), nullable=True)  # قیمت تسویه (ویژه اختیار)

    volume: Mapped[int] = mapped_column(BigInteger, nullable=False, default=0)
    value: Mapped[float] = mapped_column(Numeric(20, 2), nullable=False, default=0.0)
    trade_count: Mapped[int] = mapped_column(Integer, nullable=False, default=0)
    open_interest: Mapped[int | None] = mapped_column(Integer, nullable=True)  # موقعیت‌های باز (Open Interest)

    __table_args__ = (
        CheckConstraint("instrument_type IN ('UNDERLYING', 'OPTION')", name="check_daily_instrument_type"),
        UniqueConstraint("instrument_type", "instrument_id", "trade_date", name="uq_daily_instrument_date"),
        Index("ix_daily_inst_date", "instrument_type", "instrument_id", "trade_date"),
    )

    def __repr__(self) -> str:
        return (
            f"<DailyHistory(id={self.id}, type='{self.instrument_type}', "
            f"inst_id={self.instrument_id}, date='{self.trade_date}', close={self.close_price})>"
        )
