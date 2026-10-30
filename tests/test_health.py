from unittest.mock import AsyncMock, patch

import pytest
from httpx import AsyncClient


@pytest.mark.asyncio
async def test_health_endpoint_healthy(client: AsyncClient):
    with patch("src.interfaces.http.api.v1.health.check_db_connection", new=AsyncMock(return_value=True)):
        response = await client.get("/health")
        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "healthy"
        assert data["app"] == "QuantBackend"
        assert data["database"] == "connected"
        assert "timestamp" in data


@pytest.mark.asyncio
async def test_health_endpoint_degraded(client: AsyncClient):
    with patch("src.interfaces.http.api.v1.health.check_db_connection", new=AsyncMock(return_value=False)):
        response = await client.get("/health")
        assert response.status_code == 503
        data = response.json()
        assert data["status"] == "degraded"
        assert data["database"] == "disconnected"
