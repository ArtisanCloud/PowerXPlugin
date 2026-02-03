from fastapi import APIRouter, Request

from app.contracts.response import (
    ERR_CODE_INVALID_REQUEST,
    ERR_CODE_UNAUTHORIZED,
    fail,
    ok,
)
from app.services.integration_service import IntegrationService

router = APIRouter(prefix="/admin")
service = IntegrationService()


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


@router.get("/integration/approvals")
async def list_approvals(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"items": service.list_approvals()}, request_id=request_id)


@router.post("/integration/approvals/{approval_id}/approve")
async def approve(request: Request, approval_id: str, payload: dict | None = None):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.approve(approval_id, payload or {}), request_id=request_id)


@router.post("/integration/approvals/{approval_id}/reject")
async def reject(request: Request, approval_id: str, payload: dict | None = None):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.reject(approval_id, payload or {}), request_id=request_id)


@router.get("/integration/grant-matrix")
async def list_grant_matrix(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"items": service.list_grant_matrix_overrides()}, request_id=request_id)


@router.get("/integration/webhooks")
async def list_webhooks(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = request.query_params.get("tenant_uuid") or request.query_params.get("tenantUuid")
    items = service.list_subscriptions(tenant_uuid)
    return ok({"items": items}, request_id=request_id)


@router.post("/integration/webhooks")
async def create_webhook(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not (payload or {}).get("event_type") or not (payload or {}).get("target_url"):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "event_type/target_url 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok(service.create_subscription(payload), request_id=request_id)


@router.put("/integration/webhooks/{webhook_id}")
async def update_webhook(request: Request, webhook_id: str, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    return ok(service.update_subscription(webhook_id, payload), request_id=request_id)


@router.delete("/integration/webhooks/{webhook_id}")
async def delete_webhook(request: Request, webhook_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.delete_subscription(webhook_id), request_id=request_id)


@router.get("/integration/webhooks/{webhook_id}/attempts")
async def list_webhook_attempts(request: Request, webhook_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    items = service.list_attempts(webhook_id)
    return ok({"items": items}, request_id=request_id)


@router.post("/integration/webhooks/attempts/{attempt_id}/replay")
async def replay_attempt(request: Request, attempt_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.replay_attempt(attempt_id), request_id=request_id)


@router.get("/integration/secrets")
async def list_secrets(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = request.query_params.get("tenant_uuid") or request.query_params.get("tenantUuid")
    items = service.list_secrets(tenant_uuid)
    return ok({"items": items}, request_id=request_id)


@router.post("/integration/secrets")
async def create_secret(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not (payload or {}).get("integration_type"):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "integration_type 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok(service.create_secret(payload), request_id=request_id)


@router.post("/integration/secrets/{secret_id}/rotate")
async def rotate_secret(request: Request, secret_id: str, payload: dict | None = None):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.rotate_secret(secret_id, payload), request_id=request_id)


@router.post("/integration/secrets/{secret_id}/rotate/complete")
async def complete_rotation(request: Request, secret_id: str, payload: dict | None = None):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.complete_rotation(secret_id, payload), request_id=request_id)


@router.post("/integration/secrets/{secret_id}/revoke")
async def revoke_secret(request: Request, secret_id: str, payload: dict | None = None):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.revoke_secret(secret_id, payload), request_id=request_id)


@router.get("/integration/secrets/{secret_id}/audit")
async def get_secret_audit(request: Request, secret_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.get_secret_audit(secret_id), request_id=request_id)
