from __future__ import annotations

import logging
import os
import time
from typing import Callable

from fastapi import Request

from app.middleware.tenant_context import get_tenant_context

_LOGGER = logging.getLogger(__name__)


def _trace_enabled() -> bool:
    value = os.getenv("POWERX_DEBUG_TRAFFIC", "").strip().lower()
    if value in {"1", "true", "yes", "on"}:
        return True
    if value in {"0", "false", "no", "off"}:
        return False
    return os.getenv("POWERX_PROXY") != "1"


def _request_mode() -> str:
    if os.getenv("POWERX_PROXY") == "1":
        return "powerx-proxy"
    return "standalone"


def _provider_mode() -> str:
    mode = os.getenv("POWERX_PROVIDER_MODE", "").strip().lower()
    if mode in {"local", "delegated"}:
        return mode
    return "local"


def _detect_auth(request: Request) -> tuple[str, str]:
    auth = request.headers.get("Authorization", "")
    if auth:
        return "bearer", _shorten(auth, 40)
    ctx = request.headers.get("X-PowerX-CTX", "")
    if ctx:
        return "signed_ctx", _shorten(ctx, 40)
    return "none", ""


def _trace_identifier(request: Request) -> str:
    for key in ("X-Request-ID", "Request-ID"):
        value = (request.headers.get(key) or "").strip()
        if value:
            return value
    value = getattr(request.state, "request_id", "")
    return value or ""


def _client_ip(request: Request) -> str:
    forwarded = (request.headers.get("X-Forwarded-For") or "").split(",")[0].strip()
    if forwarded:
        return forwarded
    if request.client:
        return request.client.host
    return ""


def _shorten(raw: str, keep: int) -> str:
    raw = (raw or "").strip()
    if not raw:
        return ""
    if len(raw) <= keep:
        return raw
    return raw[:keep] + "..."


async def request_id_middleware(request: Request, call_next: Callable):
    request_id = request.headers.get("X-Request-ID") or request.headers.get("Request-ID")
    if not request_id:
        request_id = str(int(time.time_ns()))
        headers = list(request.scope.get("headers") or [])
        headers.append((b"x-request-id", request_id.encode("utf-8")))
        request.scope["headers"] = headers
    request.state.request_id = request_id
    response = await call_next(request)
    response.headers.setdefault("X-Request-ID", request_id)
    return response


async def request_trace_middleware(request: Request, call_next: Callable):
    if not _trace_enabled():
        return await call_next(request)

    start = time.time()
    auth_mode, auth_preview = _detect_auth(request)
    tenant_ctx = get_tenant_context(request)
    trace_id = _trace_identifier(request)
    _LOGGER.info(
        "[PLUGIN-REQ-TRACE] stage=begin mode=%s provider_mode=%s method=%s path=%s auth=%s auth.head=%s "
        "tenant_uuid=%s user_id=%s trace=%s ip=%s ua=%s",
        _request_mode(),
        _provider_mode(),
        request.method,
        request.url.path,
        auth_mode,
        auth_preview,
        tenant_ctx.tenant_uuid if tenant_ctx else "",
        tenant_ctx.user_id if tenant_ctx else 0,
        trace_id,
        _client_ip(request),
        _shorten(request.headers.get("User-Agent", ""), 80),
    )

    response = await call_next(request)

    latency = time.time() - start
    raw = getattr(request.state, "raw_bearer_token", "")
    if raw:
        auth_preview = _shorten(raw, 40)
        auth_mode = "bearer(validated)"
    tenant_ctx = get_tenant_context(request)
    _LOGGER.info(
        "[PLUGIN-REQ-TRACE] stage=end mode=%s provider_mode=%s status=%s latency=%s auth=%s auth.head=%s "
        "tenant_uuid=%s user_id=%s trace=%s",
        _request_mode(),
        _provider_mode(),
        response.status_code,
        f"{latency:.3f}s",
        auth_mode,
        auth_preview,
        tenant_ctx.tenant_uuid if tenant_ctx else "",
        tenant_ctx.user_id if tenant_ctx else 0,
        trace_id,
    )
    return response
