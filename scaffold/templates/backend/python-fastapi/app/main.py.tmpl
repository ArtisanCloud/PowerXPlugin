from fastapi import FastAPI

from app.config.settings import get_settings
from app.entity.repository.db import DatabaseConfig, init_db
from app.middleware.auth_guard import build_jwt_config, auth_guard_middleware
from app.middleware.dev_switch import dev_switch_middleware
from app.middleware.rbac import build_rbac_config, rbac_middleware
from app.middleware.tenant_context import tenant_context_middleware
from app.middleware.tenant_guard import tenant_guard_middleware
from app.middleware.request_trace import request_id_middleware, request_trace_middleware
from app.observability.host_context import host_context_middleware
from app.router.api import register_routes
from app.transport.http.rbac_registry import build_rbac_entries


def create_app() -> FastAPI:
    settings = get_settings()
    init_db(DatabaseConfig(url=settings.database_url, echo=settings.db_echo))
    app = FastAPI(title=settings.app_name)
    app.state.settings = settings
    app.state.jwt_cfg = build_jwt_config(settings)
    app.state.rbac_cfg = build_rbac_config(settings)
    app.state.rbac_cfg.route_permissions = build_rbac_entries(
        settings.api_prefix, app.state.rbac_cfg.plugin_id
    )
    # Middleware execution is reverse of registration order in Starlette/FastAPI.
    # Register in reverse so runtime order matches Gin: request_id -> host_context -> request_trace
    # -> tenant_context -> dev_switch -> auth_guard -> rbac -> tenant_guard.
    app.middleware("http")(tenant_guard_middleware)
    app.middleware("http")(rbac_middleware)
    app.middleware("http")(auth_guard_middleware)
    app.middleware("http")(dev_switch_middleware)
    app.middleware("http")(tenant_context_middleware)
    app.middleware("http")(request_trace_middleware)
    app.middleware("http")(host_context_middleware)
    app.middleware("http")(request_id_middleware)
    register_routes(app, settings)
    return app


app = create_app()
