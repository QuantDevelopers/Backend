from datetime import UTC, date, datetime, time
from unittest.mock import AsyncMock, MagicMock

import pytest

from src.domain.entities import (
    DailyHistoryEntity,
    DataQualityLogEntity,
    OptionContractEntity,
    TickCleanEntity,
    TickRawEntity,
    UnderlyingAssetEntity,
)
from src.infrastructure.persistence.models import (
    DailyHistoryModel,
    DataQualityLogModel,
    OptionContractModel,
    TickCleanModel,
    TickRawModel,
    UnderlyingAssetModel,
)
from src.infrastructure.persistence.repositories.option_repository import OptionContractRepository


def test_domain_entities_creation():
    ua = UnderlyingAssetEntity(
        id=1,
        symbol_fa="اهرم",
        name_fa="صندوق س. ا. کاریسما-شتاب",
        tsetmc_id="24000305886470390",
        isin="IRO1AHRM0001",
    )
    assert ua.symbol_fa == "اهرم"
    assert ua.is_active is True

    opt = OptionContractEntity(
        id=10,
        underlying_asset_id=1,
        symbol_fa="ضهرم1204",
        name_fa="اختیار خ اهرم-1200-1403/12/04",
        contract_type="CALL",
        strike_price=1200.0,
        contract_size=1000,
        expiration_date=date(2025, 2, 22),
    )
    assert opt.contract_type == "CALL"
    assert opt.strike_price == 1200.0
    assert opt.status == "ACTIVE"

    daily = DailyHistoryEntity(
        id=100,
        instrument_type="OPTION",
        instrument_id=10,
        trade_date=date(2024, 11, 1),
        open_price=150.0,
        high_price=180.0,
        low_price=140.0,
        close_price=175.0,
        settlement_price=170.0,
        volume=50000,
        open_interest=1200,
    )
    assert daily.settlement_price == 170.0
    assert daily.open_interest == 1200

    tick_raw = TickRawEntity(
        id=1000,
        instrument_type="OPTION",
        instrument_id=10,
        trade_date=date(2024, 11, 1),
        trade_time=time(9, 30, 15),
        price=175.0,
        volume=500,
        sequence_number=1,
    )
    assert tick_raw.trade_time == time(9, 30, 15)

    tick_clean = TickCleanEntity(
        id=1000,
        instrument_type="OPTION",
        instrument_id=10,
        trade_date=date(2024, 11, 1),
        trade_timestamp=datetime(2024, 11, 1, 9, 30, 15, tzinfo=UTC),
        sequence_number=1,
        price=175.0,
        volume=500,
    )
    assert tick_clean.is_canceled is False

    quality = DataQualityLogEntity(
        id=50,
        instrument_type="OPTION",
        instrument_id=10,
        trade_date=date(2024, 11, 1),
        check_category="DUPLICATE_CHECK",
        status="PASSED",
        gap_count=0,
    )
    assert quality.status == "PASSED"


def test_sqlalchemy_model_representation():
    ua_model = UnderlyingAssetModel(
        id=1,
        symbol_fa="خودرو",
        name_fa="ایران خودرو",
        tsetmc_id="2400322364771558",
        isin="IRO1IKCO0001",
    )
    assert "خودرو" in repr(ua_model)

    opt_model = OptionContractModel(
        id=2,
        underlying_asset_id=1,
        symbol_fa="ضخود1204",
        name_fa="اختیار خ خودرو-1200-1403/12/04",
        contract_type="CALL",
        strike_price=1200.0,
        contract_size=1000,
        expiration_date=date(2025, 2, 22),
    )
    assert "ضخود1204" in repr(opt_model)

    daily_model = DailyHistoryModel(
        id=3,
        instrument_type="UNDERLYING",
        instrument_id=1,
        trade_date=date(2024, 11, 1),
        open_price=1200.0,
        high_price=1250.0,
        low_price=1190.0,
        close_price=1240.0,
        volume=1000000,
        value=1240000000.0,
        trade_count=5000,
    )
    assert "DailyHistory" in repr(daily_model)

    tick_raw_model = TickRawModel(
        id=4,
        instrument_type="OPTION",
        instrument_id=2,
        trade_date=date(2024, 11, 1),
        trade_time=time(9, 15, 0),
        price=150.0,
        volume=100,
    )
    assert "TickRaw" in repr(tick_raw_model)

    tick_clean_model = TickCleanModel(
        id=5,
        instrument_type="OPTION",
        instrument_id=2,
        trade_date=date(2024, 11, 1),
        trade_timestamp=datetime(2024, 11, 1, 9, 15, 0, tzinfo=UTC),
        sequence_number=1,
        price=150.0,
        volume=100,
    )
    assert "TickClean" in repr(tick_clean_model)

    quality_model = DataQualityLogModel(
        id=6,
        instrument_type="OPTION",
        instrument_id=2,
        trade_date=date(2024, 11, 1),
        check_category="CALENDAR_COVERAGE",
        status="PASSED",
    )
    assert "DataQualityLog" in repr(quality_model)


@pytest.mark.asyncio
async def test_option_repository_get_all():
    mock_session = AsyncMock()
    mock_model = OptionContractModel(
        id=10,
        underlying_asset_id=1,
        symbol_fa="ضهرم1204",
        name_fa="اختیار خ اهرم-1200-1403/12/04",
        contract_type="CALL",
        strike_price=1200.0,
        contract_size=1000,
        expiration_date=date(2025, 2, 22),
        is_active=True,
        status="ACTIVE",
    )

    mock_result = MagicMock()
    mock_result.scalars.return_value.all.return_value = [mock_model]
    mock_session.execute.return_value = mock_result

    repo = OptionContractRepository(mock_session)
    contracts = await repo.get_all(underlying_asset_id=1, contract_type="CALL", is_active=True)

    assert len(contracts) == 1
    assert contracts[0].symbol_fa == "ضهرم1204"
    assert contracts[0].strike_price == 1200.0
    assert contracts[0].contract_type == "CALL"
