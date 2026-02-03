from fastapi import APIRouter, Request

from app.contracts.response import (
    ERR_CODE_INVALID_REQUEST,
    ERR_CODE_UNAUTHORIZED,
    fail,
    ok,
)
from app.services.runtime_session_service import RuntimeSessionService

router = APIRouter(prefix="/admin")
service = RuntimeSessionService()


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


def _require_payload(request_id: str | None, payload: dict | None, message: str = "payload 必填"):
    if not payload:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            message,
            request_id=request_id,
            status_code=400,
        )
    return None


def _validate_invoke(payload: dict) -> str | None:
    required = [
        "message_id",
        "trace_id",
        "correlation_id",
        "tenant_uuid",
        "tool_scope",
        "issued_at",
        "payload_ref",
        "signature",
    ]
    missing = [key for key in required if not payload.get(key)]
    if missing:
        return "missing required fields: " + ",".join(missing)
    return None


@router.post("/runtime/sessions/register")
async def register(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not (payload or {}).get("runtime_assignment_id"):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "invalid register request",
            request_id=request_id,
            status_code=400,
        )
    return ok(service.register(payload), request_id=request_id)


@router.post("/runtime/sessions/{session_id}/ack")
async def ack(request: Request, session_id: str, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    missing = _require_payload(request_id, payload, message="invalid ack payload")
    if missing:
        return missing
    return ok(service.ack(session_id, payload), request_id=request_id)


@router.post("/runtime/sessions/{session_id}/heartbeat")
async def heartbeat(request: Request, session_id: str, payload: dict | None = None):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.heartbeat(session_id, payload or {}), request_id=request_id)


@router.post("/runtime/sessions/{session_id}/close")
async def close(request: Request, session_id: str, payload: dict | None = None):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.close(session_id, payload or {}), request_id=request_id)


@router.post("/runtime/sessions/{session_id}/invoke")
async def invoke(request: Request, session_id: str, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    missing = _require_payload(request_id, payload, message="invalid invoke payload")
    if missing:
        return missing
    invalid = _validate_invoke(payload)
    if invalid:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            invalid,
            request_id=request_id,
            status_code=400,
        )
    if payload.get("session_id") and payload.get("session_id") != session_id:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "session_id mismatch",
            request_id=request_id,
            status_code=400,
        )
    return ok(service.invoke(session_id, payload), request_id=request_id)


@router.post("/runtime/bootstrap")
async def bootstrap(request: Request, payload: dict | None = None):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"ok": True}, request_id=request_id)


@router.get("/runtime/metrics")
async def runtime_metrics(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"metrics": []}, request_id=request_id)


@router.get("/runtime/quota/status")
async def quota_status(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"status": "ok"}, request_id=request_id)


@router.post("/runtime/quota/overrides")
async def quota_override(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    return ok({"ok": True, "override": payload}, request_id=request_id)


@router.post("/runtime/event-bridge/emit")
async def emit_event_bridge(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    return ok({"ok": True, "event": payload}, request_id=request_id)
