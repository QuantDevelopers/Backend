from functools import lru_cache
from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )

    # Application
    app_name: str = Field("QuantBackend", alias="APP_NAME")
    app_host: str = Field("0.0.0.0", alias="APP_HOST")
    app_port: int = Field(8000, alias="APP_PORT")
    debug: bool = Field(True, alias="DEBUG")
    root_path: str = Field("", alias="ROOT_PATH")

    # PostgreSQL Database
    database_url: str = Field(
        "postgresql+asyncpg://postgres:postgres@postgres:5432/quant_db",
        alias="DATABASE_URL",
    )
    db_echo: bool = Field(False, alias="DB_ECHO")
    db_pool_size: int = Field(10, alias="DB_POOL_SIZE")
    db_max_overflow: int = Field(20, alias="DB_MAX_OVERFLOW")

    # CORS
    cors_origins: str = Field(
        "http://localhost:3000,http://localhost:5173,http://127.0.0.1:3000,http://127.0.0.1:5173",
        alias="CORS_ORIGINS",
    )

    # Crawler Settings
    crawler_rate_limit_rps: float = Field(3.0, alias="CRAWLER_RATE_LIMIT_RPS")
    crawler_max_retries: int = Field(5, alias="CRAWLER_MAX_RETRIES")
    crawler_backoff_factor: float = Field(1.5, alias="CRAWLER_BACKOFF_FACTOR")
    crawler_raw_data_path: str = Field("./data/raw", alias="CRAWLER_RAW_DATA_PATH")

    @property
    def cors_origins_list(self) -> list[str]:
        return [origin.strip() for origin in self.cors_origins.split(",") if origin.strip()]


@lru_cache
def get_settings() -> Settings:
    return Settings()
