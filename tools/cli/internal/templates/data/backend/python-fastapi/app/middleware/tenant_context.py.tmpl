from __future__ import annotations

from contextvars import ContextVar
from dataclasses import dataclass, field
from typing import Callable, Optional
from uuid import UUID

from fastapi import Request

_tenant_ctx: ContextVar[Optional["TenantContext"]] = ContextVar("tenant_ctx", default=None)
_tenant_uuid_ctx: ContextVar[Optional[str]] = ContextVar("tenant_uuid", default=None)


@dataclass
class TenantContext:
    tenant_uuid: str = ""
    user_id: int = 0
    is_root: bool = False
    roles: list[str] = field(default_factory=list)
    permissions: list[str] = field(default_factory=list)
    policy_version: str = ""
    plugin_id: str = ""


def set_tenant_context(request: Request, tc: TenantContext) -> None:
    request.state.tenant_ctx = tc
    _tenant_ctx.set(tc)
    if tc.tenant_uuid:
        request.state.tenant_uuid = tc.tenant_uuid
        _tenant_uuid_ctx.set(tc.tenant_uuid)


def get_tenant_context(request: Request) -> Optional[TenantContext]:
    tc = getattr(request.state, "tenant_ctx", None)
    if isinstance(tc, TenantContext):
        return tc
    return _tenant_ctx.get()


def set_tenant_uuid(request: Request, tenant_uuid: str) -> None:
    if tenant_uuid:
        request.state.tenant_uuid = tenant_uuid
        _tenant_uuid_ctx.set(tenant_uuid)


def resolve_tenant_uuid(request: Request) -> Optional[str]:
    ctx_uuid = _tenant_uuid_ctx.get()
    if ctx_uuid:
        return ctx_uuid.strip()
    state_uuid = getattr(request.state, "tenant_uuid", None)
    if isinstance(state_uuid, str) and state_uuid.strip():
        return state_uuid.strip()
    tc = get_tenant_context(request)
    if tc and tc.tenant_uuid.strip():
        return tc.tenant_uuid.strip()
    header_uuid = _normalize_uuid(request.headers.get("tenant_uuid", ""))
    if header_uuid:
        return header_uuid
    query_uuid = _normalize_uuid(request.query_params.get("tenant_uuid", ""))
    if query_uuid:
        return query_uuid
    return None


def _normalize_uuid(raw: str) -> Optional[str]:
    raw = (raw or "").strip()
    if not raw:
        return None
    try:
        return str(UUID(raw)).lower()
    except ValueError:
        return None


async def tenant_context_middleware(request: Request, call_next: Callable):
    _tenant_ctx.set(None)
    _tenant_uuid_ctx.set(None)
    tenant_uuid = resolve_tenant_uuid(request)
    if tenant_uuid:
        set_tenant_uuid(request, tenant_uuid)
        tc = get_tenant_context(request)
        if tc and not tc.tenant_uuid:
            tc.tenant_uuid = tenant_uuid
            set_tenant_context(request, tc)
    response = await call_next(request)
    return response
