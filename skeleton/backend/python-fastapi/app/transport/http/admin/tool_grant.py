from fastapi import APIRouter, Request, Response

from app.contracts.response import (
    ERR_CODE_INVALID_REQUEST,
    ERR_CODE_UNAUTHORIZED,
    fail,
    ok,
)
from app.services.tool_grant_service import ToolGrantService

router = APIRouter(prefix="/admin")
service = ToolGrantService()


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


@router.post("/tool-grant/revoke")
async def revoke_toolgrant(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not (payload or {}).get("tenant_uuid") or not (payload or {}).get("toolgrant_id"):
        return fail(ERR_CODE_INVALID_REQUEST, "tenant_uuid/toolgrant_id 必填", request_id=request_id, status_code=400)
    service.revoke(payload)
    return Response(status_code=204)


@router.get("/tool-grant/revocations")
async def list_revocations(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = request.query_params.get("tenant_uuid")
    if not tenant_uuid:
        return fail(ERR_CODE_INVALID_REQUEST, "tenant_uuid 必填", request_id=request_id, status_code=400)
    limit = request.query_params.get("limit")
    limit_val = int(limit) if limit and limit.isdigit() else 0
    items = service.list_revocations(tenant_uuid, limit_val)
    return ok({"data": items}, request_id=request_id)


@router.get("/tool-grant/usage")
async def list_usage(request: Request):
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
    items = service.list_usage_events(tenant_uuid, toolgrant_id, limit_val)
    return ok({"data": items}, request_id=request_id)
