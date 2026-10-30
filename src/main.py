from collections.abc import AsyncIterator
from contextlib import asynccontextmanager

import uvicorn
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from src.config import get_settings
from src.container import Container
from src.infrastructure.database.base import Base
from src.interfaces.http.api.v1.health import router as health_router
from src.interfaces.http.api.v1.underlyings import router as underlyings_router

settings = get_settings()


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncIterator[None]:
    # Startup: ensure tables exist and seed initial assets
    engine = Container.engine()
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)

    # Seed 25 underlyings
    session_factory = Container.session_factory()
    async with session_factory() as session:
        service = Container.underlying_service(session)
        await service.list_underlyings()

    yield

    # Shutdown: clean up database connections
    await engine.dispose()


def create_app() -> FastAPI:
    app = FastAPI(
        title="Quant Option Data Platform API",
        description=(
            "Tehran Stock Exchange (TSETMC) Options Market Tick & Intraday Data Platform. "
            "Provides endpoints for underlying assets, option contracts, raw and clean tick streams, "
            "as-of joined series, and data quality indicators."
        ),
        version="1.0.0",
        lifespan=lifespan,
        root_path=settings.root_path,
        docs_url="/docs",
        redoc_url="/redoc",
    )

    # CORS configuration for Frontend repo
    app.add_middleware(
        CORSMiddleware,
        allow_origins=settings.cors_origins_list,
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )

    # Mount API routers
    app.include_router(health_router)
    app.include_router(underlyings_router, prefix="/api/v1")

    return app


app = create_app()

if __name__ == "__main__":
    uvicorn.run(
        "src.main:app",
        host=settings.app_host,
        port=settings.app_port,
        reload=settings.debug,
    )
