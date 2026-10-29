from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.ext.asyncio import AsyncSession
from src.container import Container
from src.infrastructure.database.session import get_db_session
from src.interfaces.http.schemas import UnderlyingAssetResponse, UnderlyingListResponse

router = APIRouter(prefix="/underlyings", tags=["Underlying Assets"])


@router.get(
    "",
    response_model=UnderlyingListResponse,
    status_code=status.HTTP_200_OK,
    summary="List All Underlying Assets",
    description="Returns the 25 key underlying assets for the Tehran Stock Exchange options market.",
)
async def list_underlyings(
    session: AsyncSession = Depends(get_db_session),
):
    service = Container.underlying_service(session)
    assets = await service.list_underlyings()
    return UnderlyingListResponse(
        total=len(assets),
        items=[UnderlyingAssetResponse.model_validate(asset) for asset in assets],
    )


@router.get(
    "/{symbol_fa}",
    response_model=UnderlyingAssetResponse,
    status_code=status.HTTP_200_OK,
    summary="Get Underlying Asset by Symbol",
    description="Returns details of a specific underlying asset by its Persian symbol (e.g., خودرو, اهرم).",
)
async def get_underlying_by_symbol(
    symbol_fa: str,
    session: AsyncSession = Depends(get_db_session),
):
    service = Container.underlying_service(session)
    asset = await service.get_underlying_by_symbol(symbol_fa)
    if not asset:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Underlying asset with symbol '{symbol_fa}' not found",
        )
    return UnderlyingAssetResponse.model_validate(asset)
