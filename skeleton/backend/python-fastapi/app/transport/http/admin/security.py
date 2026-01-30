from fastapi import APIRouter, Request

from app.contracts.response import (
    ERR_CODE_INVALID_REQUEST,
    ERR_CODE_UNAUTHORIZED,
    fail,
    ok,
)
from app.services.privacy_service import PrivacyService
from app.services.security_service import SecurityService
from app.services.tool_grant_service import ToolGrantService

router = APIRouter(prefix="/admin")
privacy_service = PrivacyService()
security_service = SecurityService()
tool_grant_service = ToolGrantService()


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


@router.get("/security/consent-tokens")
async def list_consent_tokens(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = request.query_params.get("tenant_uuid")
    if not tenant_uuid:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "tenant_uuid 必填",
            request_id=request_id,
            status_code=400,
        )
    items = privacy_service.list_consent_tokens(tenant_uuid)
    return ok({"items": items}, request_id=request_id)


@router.post("/security/consent-tokens/{token_id}/revoke")
async def revoke_consent_token(request: Request, token_id: str, payload: dict | None = None):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = request.query_params.get("tenant_uuid")
    if not tenant_uuid:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "tenant_uuid 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok({"ok": True, "token_id": token_id}, request_id=request_id)


@router.get("/security/lifecycle-events")
async def list_lifecycle_events(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = request.query_params.get("tenant_uuid")
    if not tenant_uuid:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "tenant_uuid 必填",
            request_id=request_id,
            status_code=400,
        )
    items = privacy_service.list_lifecycle_events(tenant_uuid)
    return ok({"items": items}, request_id=request_id)


@router.get("/security/audit-reports")
async def list_audit_reports(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    reports = security_service.list_audit_reports()
    return ok({"data": reports}, request_id=request_id)


@router.post("/security/advisories")
async def create_advisory(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not (payload or {}).get("reference") or not (payload or {}).get("severity") or not (payload or {}).get("summary"):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "reference/severity/summary 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok(payload, request_id=request_id)


@router.get("/security/advisories")
async def list_advisories(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    items = security_service.list_advisories()
    return ok({"items": items}, request_id=request_id)


@router.post("/security/advisories/{advisory_id}/publish")
async def publish_advisory(request: Request, advisory_id: str, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not (payload or {}).get("patched_in_version"):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "patched_in_version 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok({"id": advisory_id, "status": "published"}, request_id=request_id)


@router.post("/security/toolgrants/revoke")
async def revoke_toolgrant(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not (payload or {}).get("tenant_uuid") or not (payload or {}).get("toolgrant_id"):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "tenant_uuid/toolgrant_id 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok({"ok": True}, request_id=request_id)


@router.get("/security/toolgrants/revocations")
async def list_toolgrant_revocations(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = request.query_params.get("tenant_uuid")
    if not tenant_uuid:
        return fail(ERR_CODE_INVALID_REQUEST, "tenant_uuid 必填", request_id=request_id, status_code=400)
    items = tool_grant_service.list_revocations(tenant_uuid)
    return ok({"data": items}, request_id=request_id)


@router.get("/security/toolgrants/usage")
async def list_toolgrant_usage(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = request.query_params.get("tenant_uuid")
    if not tenant_uuid:
        return fail(ERR_CODE_INVALID_REQUEST, "tenant_uuid 必填", request_id=request_id, status_code=400)
    items = tool_grant_service.list_usage_events(tenant_uuid)
    return ok({"data": items}, request_id=request_id)
