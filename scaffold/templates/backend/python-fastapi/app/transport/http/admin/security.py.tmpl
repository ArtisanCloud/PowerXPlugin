from fastapi import APIRouter, Request, Response

from datetime import datetime

from app.contracts.response import (
    ERR_CODE_INVALID_REQUEST,
    ERR_CODE_NOT_FOUND,
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
    statuses = request.query_params.getlist("status")
    items = privacy_service.list_consent_tokens(tenant_uuid, statuses)
    return ok({"data": items}, request_id=request_id)


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
    privacy_service.revoke_consent_token(tenant_uuid, token_id, payload or {})
    return Response(status_code=204)


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
    event_types = request.query_params.getlist("event_type")
    limit = request.query_params.get("limit")
    limit_val = int(limit) if limit and limit.isdigit() else 0
    items = privacy_service.list_lifecycle_events(tenant_uuid, event_types, limit_val)
    return ok({"data": items}, request_id=request_id)


@router.get("/security/audit-reports")
async def list_audit_reports(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    limit = request.query_params.get("limit")
    limit_val = int(limit) if limit and limit.isdigit() else 0
    reports = security_service.list_audit_reports(limit_val)
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
    sla_deadline = payload.get("sla_deadline")
    if sla_deadline:
        try:
            payload["sla_deadline"] = datetime.fromisoformat(sla_deadline.replace("Z", "+00:00"))
        except ValueError:
            return fail(
                ERR_CODE_INVALID_REQUEST,
                "sla_deadline must be RFC3339",
                request_id=request_id,
                status_code=400,
            )
    return ok(security_service.create_advisory(payload), request_id=request_id, status_code=201)


@router.get("/security/advisories")
async def list_advisories(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    severity = request.query_params.getlist("severity")
    if not severity:
        raw = request.query_params.get("severity")
        if raw:
            severity = [raw]
    status = request.query_params.getlist("status")
    if not status:
        raw = request.query_params.get("status")
        if raw:
            status = [raw]
    limit = request.query_params.get("limit")
    limit_val = int(limit) if limit and limit.isdigit() else 0
    items = security_service.list_advisories(severity, status, limit_val)
    return ok({"data": items}, request_id=request_id)


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
    advisory = security_service.publish_advisory(advisory_id, payload)
    if not advisory:
        return fail(
            ERR_CODE_NOT_FOUND,
            "advisory not found",
            request_id=request_id,
            status_code=404,
        )
    return ok(advisory, request_id=request_id)


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
    tool_grant_service.revoke(payload)
    return Response(status_code=204)


@router.get("/security/toolgrants/revocations")
async def list_toolgrant_revocations(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = request.query_params.get("tenant_uuid")
    if not tenant_uuid:
        return fail(ERR_CODE_INVALID_REQUEST, "tenant_uuid 必填", request_id=request_id, status_code=400)
    limit = request.query_params.get("limit")
    limit_val = int(limit) if limit and limit.isdigit() else 0
    items = tool_grant_service.list_revocations(tenant_uuid, limit_val)
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
    toolgrant_id = request.query_params.get("toolgrant_id")
    limit = request.query_params.get("limit")
    limit_val = int(limit) if limit and limit.isdigit() else 0
    items = tool_grant_service.list_usage_events(tenant_uuid, toolgrant_id, limit_val)
    return ok({"data": items}, request_id=request_id)
