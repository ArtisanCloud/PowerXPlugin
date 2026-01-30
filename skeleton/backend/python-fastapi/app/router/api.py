from fastapi import FastAPI

from app.config.settings import Settings
from app.transport.http.health import router as health_router
from app.transport.http.admin.auth import router as admin_auth_router
from app.transport.http.admin.iam import router as admin_iam_router
from app.transport.http.admin.templates import router as admin_templates_router, public_router as templates_router
from app.transport.http.admin.capabilities import router as admin_capabilities_router
from app.transport.http.admin.runtime_sessions import router as admin_runtime_router
from app.transport.http.admin.manifest import router as admin_manifest_router
from app.transport.http.admin.integration import router as admin_integration_router
from app.transport.http.integration import router as integration_router
from app.transport.http.admin.operations import router as admin_operations_router
from app.transport.http.admin.marketplace import router as admin_marketplace_router, public_router as marketplace_router
from app.transport.http.admin.security import router as admin_security_router
from app.transport.http.admin.privacy import router as admin_privacy_router
from app.transport.http.admin.tool_grant import router as admin_tool_grant_router


def _include_router(app: FastAPI, router, prefix: str) -> None:
    app.include_router(router, prefix=prefix)


def register_routes(app: FastAPI, settings: Settings) -> None:
    host_prefix = f"/_p/{{plugin_id}}{settings.api_prefix}"
    for router in (
        health_router,
        admin_auth_router,
        admin_iam_router,
        templates_router,
        admin_templates_router,
        admin_capabilities_router,
        admin_runtime_router,
        admin_manifest_router,
        admin_integration_router,
        integration_router,
        admin_operations_router,
        admin_marketplace_router,
        marketplace_router,
        admin_security_router,
        admin_privacy_router,
        admin_tool_grant_router,
    ):
        _include_router(app, router, settings.api_prefix)
        _include_router(app, router, host_prefix)