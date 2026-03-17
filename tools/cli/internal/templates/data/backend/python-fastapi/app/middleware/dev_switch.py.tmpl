from __future__ import annotations

import os
from typing import Callable

from fastapi import Request

from app.config.settings import Settings
from app.middleware.tenant_context import TenantContext, get_tenant_context, set_tenant_context

_DEFAULT_DEV_TENANT_UUID = "00000000-0000-0000-0000-000000000001"


def _is_production(settings: Settings) -> bool:
    return not settings.dev_mode and not settings.server_dev_mode


def _default_tenant_uuid(settings: Settings) -> str:
    return (
        settings.grpc_upstream_tenant_uuid
        or os.getenv("POWERX_TENANT_UUID", "").strip()
        or _DEFAULT_DEV_TENANT_UUID
    )


async def dev_switch_middleware(request: Request, call_next: Callable):
    settings: Settings = request.app.state.settings
    if os.getenv("POWERX_PROXY") != "1" and not _is_production(settings):
        if not get_tenant_context(request):
            tenant_uuid = _default_tenant_uuid(settings)
            set_tenant_context(
                request,
                TenantContext(
                    tenant_uuid=tenant_uuid or "",
                    user_id=0,
                    roles=["superadmin"],
                    permissions=["*"],
                ),
            )
    response = await call_next(request)
    return response
