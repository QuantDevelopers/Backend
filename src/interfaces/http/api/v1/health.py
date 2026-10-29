from datetime import datetime, timezone
from fastapi import APIRouter, status
from fastapi.responses import JSONResponse
from src.config import get_settings
from src.infrastructure.database.session import check_db_connection
from src.interfaces.http.schemas import HealthResponse

router = APIRouter(tags=["Health"])


@router.get(
    "/health",
    response_model=HealthResponse,
    status_code=status.HTTP_200_OK,
    summary="Service Health Check",
    description="Returns the operational status of the backend API and database connection.",
)
async def get_health():
    settings = get_settings()
    is_db_connected = await check_db_connection()
    status_str = "healthy" if is_db_connected else "degraded"
    db_status = "connected" if is_db_connected else "disconnected"

    response = HealthResponse(
        status=status_str,
        app=settings.app_name,
        database=db_status,
        timestamp=datetime.now(timezone.utc),
    )

    if not is_db_connected:
        return JSONResponse(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            content=response.model_dump(mode="json"),
        )
    return response
