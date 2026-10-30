from datetime import date, datetime, time

from sqlalchemy import (
    BigInteger,
    Boolean,
    CheckConstraint,
    Date,
    DateTime,
    Index,
    Integer,
    Numeric,
    String,
    Time,
    UniqueConstraint,
)
from sqlalchemy.dialects.postgresql import JSONB
from sqlalchemy.orm import Mapped, mapped_column

from src.infrastructure.database.base import Base, TimestampMixin


class TickRawModel(Base, TimestampMixin):
    """Raw tick data received directly from TSETMC without any transformations."""

    __tablename__ = "ticks_raw"

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True, autoincrement=True)
    instrument_type: Mapped[str] = mapped_column(String(20), nullable=False, index=True)  # 'UNDERLYING' or 'OPTION'
    instrument_id: Mapped[int] = mapped_column(Integer, nullable=False, index=True)
    trade_date: Mapped[date] = mapped_column(Date, nullable=False, index=True)
    trade_time: Mapped[time] = mapped_column(Time, nullable=False, index=True)  # HH:MM:SS
    sequence_number: Mapped[int | None] = mapped_column(Integer, nullable=True)  # Trade row index

    price: Mapped[float] = mapped_column(Numeric(15, 2), nullable=False)
    volume: Mapped[int] = mapped_column(BigInteger, nullable=False)
    raw_payload: Mapped[dict | None] = mapped_column(JSONB, nullable=True)

    __table_args__ = (
        CheckConstraint("instrument_type IN ('UNDERLYING', 'OPTION')", name="check_raw_instrument_type"),
        Index("ix_raw_ticks_lookup", "instrument_type", "instrument_id", "trade_date", "trade_time"),
    )

    def __repr__(self) -> str:
        return (
            f"<TickRaw(id={self.id}, type='{self.instrument_type}', "
            f"inst_id={self.instrument_id}, time='{self.trade_date} {self.trade_time}', price={self.price})>"
        )


class TickCleanModel(Base, TimestampMixin):
    """Cleaned, deduplicated, sorted and validated tick sequence."""

    __tablename__ = "ticks_clean"

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True, autoincrement=True)
    instrument_type: Mapped[str] = mapped_column(String(20), nullable=False, index=True)  # 'UNDERLYING' or 'OPTION'
    instrument_id: Mapped[int] = mapped_column(Integer, nullable=False, index=True)
    trade_date: Mapped[date] = mapped_column(Date, nullable=False, index=True)
    trade_timestamp: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), nullable=False, index=True
    )  # UTC / Tehran normalized
    sequence_number: Mapped[int] = mapped_column(Integer, nullable=False, index=True)

    price: Mapped[float] = mapped_column(Numeric(15, 2), nullable=False)
    volume: Mapped[int] = mapped_column(BigInteger, nullable=False)
    is_canceled: Mapped[bool] = mapped_column(Boolean, default=False, nullable=False)
    is_outlier: Mapped[bool] = mapped_column(Boolean, default=False, nullable=False)

    __table_args__ = (
        CheckConstraint("instrument_type IN ('UNDERLYING', 'OPTION')", name="check_clean_instrument_type"),
        UniqueConstraint(
            "instrument_type",
            "instrument_id",
            "trade_date",
            "sequence_number",
            name="uq_clean_tick_seq",
        ),
        Index("ix_clean_ticks_ts", "instrument_type", "instrument_id", "trade_timestamp"),
    )

    def __repr__(self) -> str:
        return (
            f"<TickClean(id={self.id}, type='{self.instrument_type}', "
            f"inst_id={self.instrument_id}, ts='{self.trade_timestamp}', price={self.price}, vol={self.volume})>"
        )
