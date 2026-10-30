from unittest.mock import AsyncMock

import pytest
from httpx import AsyncClient

from src.container import Container
from src.domain.entities import UnderlyingAssetEntity


@pytest.mark.asyncio
async def test_list_underlyings_endpoint(client: AsyncClient, monkeypatch):
    mock_assets = [
        UnderlyingAssetEntity(
            id=1,
            symbol_fa="اهرم",
            name_fa="صندوق س. ا. کاریسما-شتاب",
            tsetmc_id="24000305886470390",
            market_type="Bourse",
            is_active=True,
        ),
        UnderlyingAssetEntity(
            id=2,
            symbol_fa="خودرو",
            name_fa="ایران خودرو",
            tsetmc_id="2400322364771558",
            market_type="Bourse",
            is_active=True,
        ),
    ]

    mock_service = AsyncMock()
    mock_service.list_underlyings.return_value = mock_assets

    monkeypatch.setattr(Container, "underlying_service", lambda session: mock_service)

    response = await client.get("/api/v1/underlyings")
    assert response.status_code == 200
    data = response.json()
    assert data["total"] == 2
    assert len(data["items"]) == 2
    assert data["items"][0]["symbol_fa"] == "اهرم"
    assert data["items"][1]["symbol_fa"] == "خودرو"


@pytest.mark.asyncio
async def test_get_underlying_by_symbol_found(client: AsyncClient, monkeypatch):
    mock_asset = UnderlyingAssetEntity(
        id=1,
        symbol_fa="اهرم",
        name_fa="صندوق س. ا. کاریسما-شتاب",
        tsetmc_id="24000305886470390",
        market_type="Bourse",
        is_active=True,
    )

    mock_service = AsyncMock()
    mock_service.get_underlying_by_symbol.return_value = mock_asset

    monkeypatch.setattr(Container, "underlying_service", lambda session: mock_service)

    response = await client.get("/api/v1/underlyings/اهرم")
    assert response.status_code == 200
    data = response.json()
    assert data["symbol_fa"] == "اهرم"
    assert data["name_fa"] == "صندوق س. ا. کاریسما-شتاب"


@pytest.mark.asyncio
async def test_get_underlying_by_symbol_not_found(client: AsyncClient, monkeypatch):
    mock_service = AsyncMock()
    mock_service.get_underlying_by_symbol.return_value = None

    monkeypatch.setattr(Container, "underlying_service", lambda session: mock_service)

    response = await client.get("/api/v1/underlyings/ناموجود")
    assert response.status_code == 404
