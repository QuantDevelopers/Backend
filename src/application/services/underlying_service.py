from src.domain.entities import UnderlyingAssetEntity
from src.infrastructure.persistence.repositories.underlying_repository import UnderlyingRepository


class UnderlyingService:
    def __init__(self, repository: UnderlyingRepository) -> None:
        self.repository = repository

    async def list_underlyings(self) -> list[UnderlyingAssetEntity]:
        # Ensure default 25 assets exist
        await self.repository.seed_defaults()
        return await self.repository.get_all()

    async def get_underlying_by_symbol(self, symbol_fa: str) -> UnderlyingAssetEntity | None:
        return await self.repository.get_by_symbol(symbol_fa)
