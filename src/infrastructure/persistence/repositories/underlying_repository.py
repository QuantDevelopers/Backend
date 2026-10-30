from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from src.domain.entities import UnderlyingAssetEntity
from src.infrastructure.persistence.models.underlying import UnderlyingAssetModel

DEFAULT_25_UNDERLYINGS = [
    {"symbol_fa": "اهرم", "name_fa": "صندوق س. ا. کاریسما-شتاب", "market_type": "Bourse", "is_active": True},
    {"symbol_fa": "خودرو", "name_fa": "ایران خودرو", "market_type": "Bourse", "is_active": True},
    {"symbol_fa": "خساپا", "name_fa": "سایپا", "market_type": "Bourse", "is_active": True},
    {"symbol_fa": "شستا", "name_fa": "سرمایه گذاری تامین اجتماعی", "market_type": "Bourse", "is_active": True},
    {"symbol_fa": "فملی", "name_fa": "ملی صنایع مس ایران", "market_type": "Bourse", "is_active": True},
    {"symbol_fa": "فولاد", "name_fa": "فولاد مبارکه اصفهان", "market_type": "Bourse", "is_active": True},
    {"symbol_fa": "طلا", "name_fa": "صندوق پشتوانه طلای لوتوس", "market_type": "Bourse", "is_active": True},
    {"symbol_fa": "عیار", "name_fa": "صندوق طلای عیار مفید", "market_type": "Bourse", "is_active": True},
    {"symbol_fa": "کهربا", "name_fa": "صندوق طلای کهربا", "market_type": "Bourse", "is_active": True},
    {"symbol_fa": "زر", "name_fa": "صندوق طلای زرین ترنج", "market_type": "Bourse", "is_active": True},
    {"symbol_fa": "گوهر", "name_fa": "صندوق طلای گوهر نفیس", "market_type": "Bourse", "is_active": True},
    {"symbol_fa": "زرفام", "name_fa": "صندوق طلای زرفام ترنج", "market_type": "Bourse", "is_active": True},
    {"symbol_fa": "ناب", "name_fa": "صندوق طلای ناب مفید", "market_type": "Bourse", "is_active": True},
    {"symbol_fa": "وبملت", "name_fa": "بانک ملت", "market_type": "Bourse", "is_active": True},
    {"symbol_fa": "وتجارت", "name_fa": "بانک تجارت", "market_type": "Bourse", "is_active": True},
    {"symbol_fa": "دی", "name_fa": "بانک دی", "market_type": "Farabourse", "is_active": True},
    {"symbol_fa": "خگستر", "name_fa": "گسترش سرمایه گذاری ایران خودرو", "market_type": "Bourse", "is_active": True},
    {"symbol_fa": "شتران", "name_fa": "پالایش نفت تهران", "market_type": "Bourse", "is_active": True},
    {"symbol_fa": "شپنا", "name_fa": "پالایش نفت اصفهان", "market_type": "Bourse", "is_active": True},
    {"symbol_fa": "شبندر", "name_fa": "پالایش نفت بندرعباس", "market_type": "Bourse", "is_active": True},
    {"symbol_fa": "تاپیکو", "name_fa": "س. نفت و گاز و پتروشیمی تامین", "market_type": "Bourse", "is_active": True},
    {"symbol_fa": "فارس", "name_fa": "صنایع پتروشیمی خلیج فارس", "market_type": "Bourse", "is_active": True},
    {"symbol_fa": "وغدیر", "name_fa": "سرمایه گذاری غدیر", "market_type": "Bourse", "is_active": True},
    {"symbol_fa": "کگل", "name_fa": "معدنی و صنعتی گل گهر", "market_type": "Bourse", "is_active": True},
    {"symbol_fa": "کچاد", "name_fa": "معدنی و صنعتی چادرملو", "market_type": "Bourse", "is_active": True},
]


class UnderlyingRepository:
    def __init__(self, session: AsyncSession) -> None:
        self.session = session

    async def get_all(self) -> list[UnderlyingAssetEntity]:
        stmt = select(UnderlyingAssetModel).order_by(UnderlyingAssetModel.id)
        result = await self.session.execute(stmt)
        models = result.scalars().all()
        return [
            UnderlyingAssetEntity(
                id=m.id,
                symbol_fa=m.symbol_fa,
                name_fa=m.name_fa,
                tsetmc_id=m.tsetmc_id,
                market_type=m.market_type,
                is_active=m.is_active,
                created_at=m.created_at,
                updated_at=m.updated_at,
            )
            for m in models
        ]

    async def get_by_symbol(self, symbol_fa: str) -> UnderlyingAssetEntity | None:
        stmt = select(UnderlyingAssetModel).where(UnderlyingAssetModel.symbol_fa == symbol_fa)
        result = await self.session.execute(stmt)
        m = result.scalars().first()
        if not m:
            return None
        return UnderlyingAssetEntity(
            id=m.id,
            symbol_fa=m.symbol_fa,
            name_fa=m.name_fa,
            tsetmc_id=m.tsetmc_id,
            market_type=m.market_type,
            is_active=m.is_active,
            created_at=m.created_at,
            updated_at=m.updated_at,
        )

    async def seed_defaults(self) -> int:
        """Seed the 25 key underlying assets if table is empty."""
        existing = await self.get_all()
        if len(existing) >= len(DEFAULT_25_UNDERLYINGS):
            return 0

        inserted_count = 0
        existing_symbols = {item.symbol_fa for item in existing}
        for item in DEFAULT_25_UNDERLYINGS:
            if item["symbol_fa"] not in existing_symbols:
                model = UnderlyingAssetModel(
                    symbol_fa=item["symbol_fa"],
                    name_fa=item["name_fa"],
                    market_type=item["market_type"],
                    is_active=item["is_active"],
                )
                self.session.add(model)
                inserted_count += 1

        if inserted_count > 0:
            await self.session.commit()
        return inserted_count
