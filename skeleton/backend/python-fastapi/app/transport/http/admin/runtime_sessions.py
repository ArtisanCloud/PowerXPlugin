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


@router.post("/runtime/sessions/register")
async def register(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not (payload or {}).get("runtime_assignment_id"):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "runtime_assignment_id 必填",
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
    if not payload:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "payload 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok(service.ack(session_id, payload), request_id=request_id)


@router.post("/runtime/sessions/{session_id}/heartbeat")
async def heartbeat(request: Request, session_id: str, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.heartbeat(session_id, payload or {}), request_id=request_id)


@router.post("/runtime/sessions/{session_id}/close")
async def close(request: Request, session_id: str, payload: dict):
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
    if not payload:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "payload 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok(service.invoke(session_id, payload), request_id=request_id)
