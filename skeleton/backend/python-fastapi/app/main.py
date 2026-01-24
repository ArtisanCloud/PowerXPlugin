from fastapi import FastAPI

from app.config.settings import get_settings
from app.router.api import register_routes


def create_app() -> FastAPI:
    settings = get_settings()
    app = FastAPI(title=settings.app_name)
    register_routes(app, settings)
    return app


app = create_app()
