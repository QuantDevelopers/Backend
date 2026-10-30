from datetime import date

from sqlalchemy import (
    BigInteger,
    CheckConstraint,
    Date,
    Index,
    Integer,
    String,
    Text,
)
from sqlalchemy.dialects.postgresql import JSONB
from sqlalchemy.orm import Mapped, mapped_column

from src.infrastructure.database.base import Base, TimestampMixin


class DataQualityLogModel(Base, TimestampMixin):
    """Quality assurance, gap analysis, and consistency checks on ingested data."""

    __tablename__ = "data_quality_logs"

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True, autoincrement=True)
    instrument_type: Mapped[str] = mapped_column(String(20), nullable=False, index=True)
    instrument_id: Mapped[int] = mapped_column(Integer, nullable=False, index=True)
    trade_date: Mapped[date] = mapped_column(Date, nullable=False, index=True)

    check_category: Mapped[str] = mapped_column(
        String(50), nullable=False, index=True
    )  # 'DUPLICATE_CHECK', 'CHRONOLOGICAL_ORDER', 'PRICE_BOUNDS', 'CALENDAR_COVERAGE'
    status: Mapped[str] = mapped_column(String(20), nullable=False, index=True)  # 'PASSED', 'WARNING', 'FAILED'
    gap_count: Mapped[int] = mapped_column(Integer, default=0, nullable=False)
    details: Mapped[dict | None] = mapped_column(JSONB, nullable=True)
    notes: Mapped[str | None] = mapped_column(Text, nullable=True)

    __table_args__ = (
        CheckConstraint("status IN ('PASSED', 'WARNING', 'FAILED')", name="check_quality_status"),
        Index("ix_quality_inst_date", "instrument_type", "instrument_id", "trade_date"),
    )

    def __repr__(self) -> str:
        return (
            f"<DataQualityLog(id={self.id}, type='{self.instrument_type}', "
            f"check='{self.check_category}', status='{self.status}')>"
        )
