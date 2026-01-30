from fastapi import APIRouter, Request

from app.contracts.response import (
    ERR_CODE_INVALID_REQUEST,
    ERR_CODE_UNAUTHORIZED,
    fail,
    ok,
)
from app.services.operations_service import OperationsService

router = APIRouter(prefix="/admin")
service = OperationsService()


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


@router.get("/operations/support/playbook")
async def get_playbook(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"items": []}, request_id=request_id)


@router.put("/operations/support/playbook")
async def update_playbook(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    return ok(payload, request_id=request_id)


@router.post("/operations/support/channels/test")
async def test_channels(request: Request, payload: dict | None = None):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"ok": True}, request_id=request_id)


@router.get("/operations/support/metrics")
async def get_metrics(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"metrics": []}, request_id=request_id)


@router.get("/operations/sla/profiles")
async def list_sla_profiles(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    plugin_id = request.query_params.get("plugin_id")
    items = service.list_sla_profiles(plugin_id)
    return ok({"items": items}, request_id=request_id)


@router.post("/operations/sla/profiles")
async def upsert_profile(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    return ok(payload, request_id=request_id)


@router.post("/operations/sla/profiles/recompute")
async def recompute_profiles(request: Request, payload: dict | None = None):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"ok": True}, request_id=request_id)


@router.patch("/operations/sla/profiles/actuals")
async def update_actuals(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    return ok(payload, request_id=request_id)


@router.post("/operations/incidents")
async def create_incident(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    return ok(payload, request_id=request_id)


@router.get("/operations/incidents")
async def list_incidents(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"items": service.list_incidents(request.query_params.get("plugin_id"))}, request_id=request_id)


@router.get("/operations/incidents/{incident_id}")
async def get_incident(request: Request, incident_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"id": incident_id}, request_id=request_id)


@router.patch("/operations/incidents/{incident_id}")
async def update_incident(request: Request, incident_id: str, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    return ok({"id": incident_id, **payload}, request_id=request_id)


@router.post("/operations/incidents/{incident_id}/timeline")
async def append_timeline(request: Request, incident_id: str, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    return ok({"id": incident_id, "ok": True}, request_id=request_id)
