from fastapi import FastAPI

from app.config.settings import Settings
from app.transport.http.health import router as health_router


def register_routes(app: FastAPI, settings: Settings) -> None:
    app.include_router(health_router, prefix=settings.api_prefix)
