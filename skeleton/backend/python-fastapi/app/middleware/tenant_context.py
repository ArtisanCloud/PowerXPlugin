from typing import Callable

from fastapi import Request


async def tenant_context_middleware(request: Request, call_next: Callable):
    tenant_uuid = request.headers.get("x-tenant-uuid")
    if tenant_uuid:
        request.state.tenant_uuid = tenant_uuid
    response = await call_next(request)
    return response
