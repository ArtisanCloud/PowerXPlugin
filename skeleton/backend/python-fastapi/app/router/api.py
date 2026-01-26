from fastapi import FastAPI

from app.config.settings import Settings
from app.transport.http.health import router as health_router
from app.transport.http.admin.auth import router as admin_auth_router
from app.transport.http.admin.iam import router as admin_iam_router
from app.transport.http.admin.templates import router as admin_templates_router
from app.transport.http.admin.capabilities import router as admin_capabilities_router
from app.transport.http.admin.runtime_sessions import router as admin_runtime_router


def register_routes(app: FastAPI, settings: Settings) -> None:
    app.include_router(health_router, prefix=settings.api_prefix)
    app.include_router(admin_auth_router, prefix=settings.api_prefix)
    app.include_router(admin_iam_router, prefix=settings.api_prefix)
    app.include_router(admin_templates_router, prefix=settings.api_prefix)
    app.include_router(admin_capabilities_router, prefix=settings.api_prefix)
    app.include_router(admin_runtime_router, prefix=settings.api_prefix)
