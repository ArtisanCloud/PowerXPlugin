from fastapi import FastAPI

from app.config.settings import get_settings
from app.entity.repository.db import DatabaseConfig, init_db
from app.router.api import register_routes


def create_app() -> FastAPI:
    settings = get_settings()
    init_db(DatabaseConfig(url=settings.database_url, echo=settings.db_echo))
    app = FastAPI(title=settings.app_name)
    register_routes(app, settings)
    return app


app = create_app()
