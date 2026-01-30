from fastapi import APIRouter, Request

from app.contracts.response import (
    ERR_CODE_INVALID_REQUEST,
    ERR_CODE_UNAUTHORIZED,
    fail,
    ok,
)

router = APIRouter()


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


@router.post("/integration/dispatch")
async def dispatch(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    return ok({"status": "ok"}, request_id=request_id)


@router.post("/integration/capabilities/invoke")
async def invoke_capability(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    return ok({"status": "ok", "payload": {}, "metadata": {}}, request_id=request_id)


@router.get("/integration/grant-matrix")
async def list_grant_matrix(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"items": []}, request_id=request_id)


@router.post("/integration/grant-matrix")
async def submit_grant_matrix(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    return ok({"ok": True}, request_id=request_id)


@router.post("/integration/webhooks/subscriptions")
async def create_subscription(request: Request, payload: dict):
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
    return ok(payload, request_id=request_id)


@router.get("/integration/webhooks/subscriptions")
async def list_subscriptions(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"items": []}, request_id=request_id)


@router.post("/integration/webhooks/dlq/{attempt_id}/replay")
async def replay_dlq(request: Request, attempt_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"ok": True, "attempt_id": attempt_id}, request_id=request_id)


@router.post("/integration/secrets")
async def create_secret(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not (payload or {}).get("integration_type"):
        return fail(ERR_CODE_INVALID_REQUEST, "integration_type 必填", request_id=request_id, status_code=400)
    return ok(payload, request_id=request_id)


@router.post("/integration/secrets/{secret_id}/rotate")
async def rotate_secret(request: Request, secret_id: str, payload: dict | None = None):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"id": secret_id, "status": "rotating"}, request_id=request_id)
