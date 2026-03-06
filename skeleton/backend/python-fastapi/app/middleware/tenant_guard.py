from __future__ import annotations

import os
from typing import Callable

from fastapi import Request
from fastapi.responses import JSONResponse

from app.middleware.tenant_context import TenantContext, get_tenant_context, resolve_tenant_uuid, set_tenant_context


def _is_delegated_mode() -> bool:
    return os.getenv("POWERX_PROXY") == "1"


def _need_tenant(path: str, api_prefix: str) -> bool:
    prefix = api_prefix.rstrip("/")
    if path.startswith(f"{prefix}/admin/runtime"):
        return True
    if not _is_delegated_mode():
        return False
    for p in (
        f"{prefix}/admin/templates",
        f"{prefix}/admin/marketplace",
        f"{prefix}/admin/capabilities",
        f"{prefix}/admin/security",
    ):
        if path.startswith(p):
            return True
    return False


async def tenant_guard_middleware(request: Request, call_next: Callable):
    path = request.url.path
    api_prefix = request.app.state.settings.api_prefix
    if not _need_tenant(path, api_prefix):
        return await call_next(request)

    tenant_uuid = resolve_tenant_uuid(request)
    if not tenant_uuid:
        return JSONResponse(status_code=401, content={"error": "tenant context missing"})

    tc = get_tenant_context(request)
    if tc is None:
        set_tenant_context(request, TenantContext(tenant_uuid=tenant_uuid))
    elif not tc.tenant_uuid:
        tc.tenant_uuid = tenant_uuid
        set_tenant_context(request, tc)

    return await call_next(request)
