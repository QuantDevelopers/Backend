from src.domain.entities import OptionContractEntity
from src.infrastructure.persistence.repositories.option_repository import OptionContractRepository


class OptionService:
    def __init__(self, repository: OptionContractRepository) -> None:
        self.repository = repository

    async def list_options(
        self,
        underlying_asset_id: int | None = None,
        contract_type: str | None = None,
        is_active: bool | None = None,
    ) -> list[OptionContractEntity]:
        return await self.repository.get_all(
            underlying_asset_id=underlying_asset_id,
            contract_type=contract_type,
            is_active=is_active,
        )

    async def get_by_symbol(self, symbol_fa: str) -> OptionContractEntity | None:
        return await self.repository.get_by_symbol(symbol_fa)

    async def create_or_update(self, entity: OptionContractEntity) -> OptionContractEntity:
        return await self.repository.save(entity)
