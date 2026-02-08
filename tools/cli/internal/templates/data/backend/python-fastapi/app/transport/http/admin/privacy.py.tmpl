from fastapi import APIRouter, Request

from app.contracts.response import (
    ERR_CODE_INVALID_REQUEST,
    ERR_CODE_UNAUTHORIZED,
    fail,
    ok,
)
from app.middleware.tenant_context import resolve_tenant_uuid
from app.services.privacy_service import PrivacyService

router = APIRouter(prefix="/admin")
service = PrivacyService()


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


def _resolve_tenant_uuid(request: Request) -> str:
    candidate = request.query_params.get("tenant_uuid") or request.query_params.get("tenantUuid")
    if candidate:
        return str(candidate).strip()
    return str(resolve_tenant_uuid(request) or "").strip()


@router.get("/privacy/consent-tokens")
async def list_consent_tokens(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = _resolve_tenant_uuid(request)
    if not tenant_uuid:
        return fail(ERR_CODE_INVALID_REQUEST, "tenant_uuid 必填", request_id=request_id, status_code=400)
    items = service.list_consent_tokens(tenant_uuid)
    return ok({"items": items}, request_id=request_id)


@router.get("/privacy/lifecycle-events")
async def list_lifecycle_events(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = _resolve_tenant_uuid(request)
    if not tenant_uuid:
        return fail(ERR_CODE_INVALID_REQUEST, "tenant_uuid 必填", request_id=request_id, status_code=400)
    items = service.list_lifecycle_events(tenant_uuid)
    return ok({"items": items}, request_id=request_id)
