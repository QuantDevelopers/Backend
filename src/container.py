from sqlalchemy.ext.asyncio import AsyncEngine, AsyncSession, async_sessionmaker

from src.application.services.option_service import OptionService
from src.application.services.underlying_service import UnderlyingService
from src.config import Settings, get_settings
from src.infrastructure.database.session import get_engine, get_session_factory
from src.infrastructure.persistence.repositories.option_repository import OptionContractRepository
from src.infrastructure.persistence.repositories.underlying_repository import UnderlyingRepository


class Container:
    @classmethod
    def settings(cls) -> Settings:
        return get_settings()

    @classmethod
    def engine(cls) -> AsyncEngine:
        return get_engine()

    @classmethod
    def session_factory(cls) -> async_sessionmaker[AsyncSession]:
        return get_session_factory()

    @classmethod
    def underlying_service(cls, session: AsyncSession) -> UnderlyingService:
        repository = UnderlyingRepository(session)
        return UnderlyingService(repository)

    @classmethod
    def option_service(cls, session: AsyncSession) -> OptionService:
        repository = OptionContractRepository(session)
        return OptionService(repository)
