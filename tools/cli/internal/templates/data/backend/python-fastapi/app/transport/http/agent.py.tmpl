from fastapi import APIRouter, Request

import logging
import uuid

from app.contracts.response import (
    ERR_CODE_INTERNAL_ERROR,
    ERR_CODE_INVALID_REQUEST,
    ERR_CODE_FORBIDDEN,
    ERR_CODE_UNAUTHORIZED,
    fail,
    ok,
)
from app.config.settings import get_settings
from app.middleware.tenant_context import resolve_tenant_uuid
from app.services.plugin_service import PluginService
from app.services.privacy_service import PrivacyService
from app.services.tool_grant_service import ToolGrantService
from app.shared.sts_client import STSExchangeError, get_manager

router = APIRouter(prefix="/agent")
plugin_service = PluginService()
privacy_service = PrivacyService()
toolgrant_service = ToolGrantService()
logger = logging.getLogger(__name__)


def _request_id(request: Request) -> str | None:
    return request.headers.get("X-Request-ID") or request.headers.get("Request-ID")


def _bearer_token(request: Request) -> str:
    raw = (request.headers.get("Authorization") or "").strip()
    if not raw:
        return ""
    if raw.lower().startswith("bearer "):
        return raw[7:].strip()
    return raw


def _require_auth(request: Request):
    request_id = _request_id(request)
    if not _bearer_token(request):
        return fail(
            ERR_CODE_UNAUTHORIZED,
            "缺少 Authorization Bearer token",
            request_id=request_id,
            status_code=401,
        )
    return None


@router.post("/tenants/{tenant_id}/credentials")
async def upsert_credentials(request: Request, tenant_id: str, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = (tenant_id or "").strip()
    if not tenant_uuid:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "invalid tenant_uuid",
            request_id=request_id,
            status_code=400,
        )
    try:
        uuid.UUID(tenant_uuid)
    except ValueError:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "invalid tenant_uuid",
            request_id=request_id,
            status_code=400,
        )
    required = ("plugin_id", "client_id", "client_secret")
    missing = [key for key in required if not (payload or {}).get(key)]
    if missing:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "missing: " + ",".join(missing),
            request_id=request_id,
            status_code=400,
        )
    try:
        plugin_service.upsert_credentials(
            tenant_uuid,
            payload.get("plugin_id") or "",
            payload.get("client_id") or "",
            payload.get("client_secret") or "",
        )
    except ValueError as exc:
        message = str(exc)
        if "secret_key" in message:
            logger.warning("save credentials failed: %s", exc)
            return fail(
                ERR_CODE_INTERNAL_ERROR,
                message,
                request_id=request_id,
                status_code=500,
            )
        return fail(
            ERR_CODE_INVALID_REQUEST,
            message,
            request_id=request_id,
            status_code=400,
        )
    except Exception as exc:
        logger.warning("save credentials failed: %s", exc)
        return fail(
            ERR_CODE_INTERNAL_ERROR,
            "save credentials failed",
            request_id=request_id,
            status_code=500,
        )
    return ok({"plugin_id": payload.get("plugin_id")}, request_id=request_id)


@router.post("/sts/exchange")
async def sts_exchange(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    settings = get_settings()
    if not settings.grpc_upstream_sts_client_id or not settings.grpc_upstream_sts_client_secret:
        return fail(
            ERR_CODE_INTERNAL_ERROR,
            "powerx client not initialized",
            request_id=request_id,
            status_code=500,
        )
    if not settings.sts_endpoint:
        return fail(
            ERR_CODE_INTERNAL_ERROR,
            "sts endpoint not configured",
            request_id=request_id,
            status_code=500,
        )
    try:
        manager = get_manager(
            settings.sts_endpoint,
            settings.grpc_upstream_sts_client_id,
            settings.grpc_upstream_sts_client_secret,
            settings.grpc_upstream_sts_audience,
            settings.grpc_upstream_sts_scope,
        )
        token, expires_in = manager.exchange_now()
    except STSExchangeError as exc:
        return fail(
            ERR_CODE_INTERNAL_ERROR,
            str(exc),
            request_id=request_id,
            status_code=500,
        )
    return ok({"access_token": token, "expires_in": expires_in}, request_id=request_id)


@router.get("/security/privacy/consent")
async def get_active_consent(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = resolve_tenant_uuid(request)
    if not tenant_uuid:
        return fail(
            ERR_CODE_FORBIDDEN,
            "tenant context missing",
            request_id=request_id,
            status_code=403,
        )
    result = privacy_service.active_consent_assets(tenant_uuid)
    return ok(result, request_id=request_id)


@router.post("/security/privacy/lifecycle")
async def acknowledge_lifecycle_event(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    tenant_uuid = resolve_tenant_uuid(request)
    if not tenant_uuid:
        return fail(
            ERR_CODE_FORBIDDEN,
            "tenant context missing",
            request_id=request_id,
            status_code=403,
        )
    event_type = payload.get("event_type") or ""
    asset_key = payload.get("asset_key") or ""
    if not event_type or not asset_key:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "event_type/asset_key 必填",
            request_id=request_id,
            status_code=400,
        )
    privacy_service.record_lifecycle_event(
        tenant_uuid=tenant_uuid,
        event_type=event_type,
        asset_key=asset_key,
        metadata=payload.get("metadata") or {},
        recorded_by="agent",
    )
    return ok({"ok": True}, request_id=request_id, status_code=202)


@router.post("/security/toolgrants/verify")
async def verify_toolgrant(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    tenant_uuid = resolve_tenant_uuid(request)
    if not tenant_uuid:
        return fail(
            ERR_CODE_FORBIDDEN,
            "tenant context missing",
            request_id=request_id,
            status_code=403,
        )
    token = payload.get("token") or ""
    if not token:
        return fail(ERR_CODE_INVALID_REQUEST, "token 必填", request_id=request_id, status_code=400)
    try:
        claims = toolgrant_service.validate(tenant_uuid, token)
    except ValueError as exc:
        return fail(
            ERR_CODE_FORBIDDEN,
            str(exc),
            request_id=request_id,
            status_code=403,
        )
    return ok({"ok": True, "claims": claims}, request_id=request_id)
