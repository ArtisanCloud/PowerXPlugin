from dataclasses import dataclass


@dataclass(frozen=True)
class Settings:
    app_name: str = "PowerX Plugin (FastAPI)"
    api_prefix: str = "/api/v1"


def get_settings() -> Settings:
    return Settings()
