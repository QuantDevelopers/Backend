from collections.abc import Sequence

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from src.domain.entities import OptionContractEntity
from src.infrastructure.persistence.models.option_contract import OptionContractModel


class OptionContractRepository:
    def __init__(self, session: AsyncSession) -> None:
        self.session = session

    def _to_entity(self, m: OptionContractModel) -> OptionContractEntity:
        return OptionContractEntity(
            id=m.id,
            underlying_asset_id=m.underlying_asset_id,
            symbol_fa=m.symbol_fa,
            name_fa=m.name_fa,
            contract_type=m.contract_type,
            strike_price=float(m.strike_price),
            contract_size=m.contract_size,
            expiration_date=m.expiration_date,
            begin_date=m.begin_date,
            tsetmc_id=m.tsetmc_id,
            isin=m.isin,
            is_active=m.is_active,
            status=m.status,
            created_at=m.created_at,
            updated_at=m.updated_at,
        )

    async def get_all(
        self,
        underlying_asset_id: int | None = None,
        contract_type: str | None = None,
        is_active: bool | None = None,
    ) -> list[OptionContractEntity]:
        stmt = select(OptionContractModel)
        if underlying_asset_id is not None:
            stmt = stmt.where(OptionContractModel.underlying_asset_id == underlying_asset_id)
        if contract_type is not None:
            stmt = stmt.where(OptionContractModel.contract_type == contract_type.upper())
        if is_active is not None:
            stmt = stmt.where(OptionContractModel.is_active == is_active)

        stmt = stmt.order_by(OptionContractModel.expiration_date, OptionContractModel.strike_price)
        result = await self.session.execute(stmt)
        return [self._to_entity(m) for m in result.scalars().all()]

    async def get_by_symbol(self, symbol_fa: str) -> OptionContractEntity | None:
        stmt = select(OptionContractModel).where(OptionContractModel.symbol_fa == symbol_fa)
        result = await self.session.execute(stmt)
        m = result.scalars().first()
        return self._to_entity(m) if m else None

    async def save(self, entity: OptionContractEntity) -> OptionContractEntity:
        model = OptionContractModel(
            underlying_asset_id=entity.underlying_asset_id,
            symbol_fa=entity.symbol_fa,
            name_fa=entity.name_fa,
            contract_type=entity.contract_type,
            strike_price=entity.strike_price,
            contract_size=entity.contract_size,
            expiration_date=entity.expiration_date,
            begin_date=entity.begin_date,
            tsetmc_id=entity.tsetmc_id,
            isin=entity.isin,
            is_active=entity.is_active,
            status=entity.status,
        )
        self.session.add(model)
        await self.session.commit()
        await self.session.refresh(model)
        return self._to_entity(model)

    async def save_bulk(self, entities: Sequence[OptionContractEntity]) -> int:
        models = [
            OptionContractModel(
                underlying_asset_id=e.underlying_asset_id,
                symbol_fa=e.symbol_fa,
                name_fa=e.name_fa,
                contract_type=e.contract_type,
                strike_price=e.strike_price,
                contract_size=e.contract_size,
                expiration_date=e.expiration_date,
                begin_date=e.begin_date,
                tsetmc_id=e.tsetmc_id,
                isin=e.isin,
                is_active=e.is_active,
                status=e.status,
            )
            for e in entities
        ]
        self.session.add_all(models)
        await self.session.commit()
        return len(models)
