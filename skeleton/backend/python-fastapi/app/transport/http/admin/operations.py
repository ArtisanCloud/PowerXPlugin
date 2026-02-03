from __future__ import annotations

from datetime import datetime

from fastapi import APIRouter, Request, Response

from app.contracts.response import (
    ERR_CODE_INVALID_REQUEST,
    ERR_CODE_UNAUTHORIZED,
    fail,
    ok,
)
from app.services.operations_service import OperationsService, _as_datetime, _normalize_list

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


def _resolve_plugin_id(request: Request, payload: dict | None = None) -> str | None:
    plugin_id = getattr(request.state, "plugin_id", None)
    if plugin_id:
        return plugin_id
    if payload and payload.get("plugin_id"):
        return payload.get("plugin_id")
    return request.query_params.get("plugin_id")


@router.get("/operations/support/playbook")
async def get_playbook(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    plugin_id = _resolve_plugin_id(request)
    tenant_uuid = request.query_params.get("tenant_uuid")
    if not plugin_id:
        return fail(ERR_CODE_INVALID_REQUEST, "plugin_id 必填", request_id=request_id, status_code=400)
    payload = service.get_support_playbook(plugin_id, tenant_uuid)
    return ok(payload, request_id=request_id)


@router.put("/operations/support/playbook")
async def update_playbook(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    plugin_id = _resolve_plugin_id(request, payload)
    if not plugin_id:
        return fail(ERR_CODE_INVALID_REQUEST, "plugin_id 必填", request_id=request_id, status_code=400)
    updated = service.configure_support_playbook(plugin_id, payload)
    return ok(updated, request_id=request_id)


@router.post("/operations/support/channels/test")
async def test_channels(request: Request, payload: dict | None = None):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"status": "ok"}, message="channel validation dispatched", request_id=request_id)


@router.get("/operations/support/metrics")
async def get_metrics(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    plugin_id = _resolve_plugin_id(request)
    if not plugin_id:
        return fail(ERR_CODE_INVALID_REQUEST, "plugin_id 必填", request_id=request_id, status_code=400)
    metrics = service.compute_support_metrics(plugin_id)
    return ok(metrics, request_id=request_id)


@router.get("/operations/sla/profiles")
async def list_sla_profiles(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    plugin_id = _resolve_plugin_id(request)
    if not plugin_id:
        return fail(ERR_CODE_INVALID_REQUEST, "plugin_id 必填", request_id=request_id, status_code=400)
    items = service.list_sla_profiles(plugin_id)
    return ok(items, request_id=request_id)


@router.post("/operations/sla/profiles")
async def upsert_profile(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    plugin_id = _resolve_plugin_id(request, payload)
    if not plugin_id:
        return fail(ERR_CODE_INVALID_REQUEST, "plugin_id 必填", request_id=request_id, status_code=400)
    profile = service.upsert_sla_profile(plugin_id, payload)
    return ok(profile, request_id=request_id)


@router.post("/operations/sla/profiles/recompute")
async def recompute_profiles(request: Request, payload: dict | None = None):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    plugin_id = _resolve_plugin_id(request, payload or {})
    if not plugin_id:
        return fail(ERR_CODE_INVALID_REQUEST, "plugin_id 必填", request_id=request_id, status_code=400)
    service.recompute_sla(plugin_id)
    return Response(status_code=202)


@router.patch("/operations/sla/profiles/actuals")
async def update_actuals(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    plugin_id = _resolve_plugin_id(request, payload)
    if not plugin_id:
        return fail(ERR_CODE_INVALID_REQUEST, "plugin_id 必填", request_id=request_id, status_code=400)
    plan_type = payload.get("planType") or payload.get("plan_type")
    if not plan_type:
        return fail(ERR_CODE_INVALID_REQUEST, "planType 必填", request_id=request_id, status_code=400)
    profile = service.update_sla_actuals(plugin_id, plan_type, payload)
    return ok(profile, request_id=request_id)


@router.post("/operations/incidents")
async def create_incident(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    plugin_id = _resolve_plugin_id(request, payload)
    if not plugin_id:
        return fail(ERR_CODE_INVALID_REQUEST, "plugin_id 必填", request_id=request_id, status_code=400)
    response = service.create_incident(plugin_id, payload)
    return ok(response, request_id=request_id, status_code=201)


@router.get("/operations/incidents")
async def list_incidents(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    plugin_id = _resolve_plugin_id(request)
    if not plugin_id:
        return fail(ERR_CODE_INVALID_REQUEST, "plugin_id 必填", request_id=request_id, status_code=400)
    severities = _normalize_list(request.query_params.getlist("severity"))
    statuses = _normalize_list(request.query_params.getlist("status"))
    labels = _normalize_list(request.query_params.getlist("label"))
    from_dt: datetime | None = _as_datetime(request.query_params.get("from"))
    to_dt: datetime | None = _as_datetime(request.query_params.get("to"))
    items = service.list_incidents(plugin_id, severities, statuses, labels, from_dt, to_dt)
    return ok(items, request_id=request_id)


@router.get("/operations/incidents/{incident_id}")
async def get_incident(request: Request, incident_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    plugin_id = _resolve_plugin_id(request)
    if not plugin_id:
        return fail(ERR_CODE_INVALID_REQUEST, "plugin_id 必填", request_id=request_id, status_code=400)
    response = service.get_incident_response(plugin_id, incident_id)
    return ok(response, request_id=request_id)


@router.patch("/operations/incidents/{incident_id}")
async def update_incident(request: Request, incident_id: str, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    plugin_id = _resolve_plugin_id(request, payload)
    if not plugin_id:
        return fail(ERR_CODE_INVALID_REQUEST, "plugin_id 必填", request_id=request_id, status_code=400)
    response = service.update_incident(plugin_id, incident_id, payload)
    return ok(response, request_id=request_id)


@router.post("/operations/incidents/{incident_id}/timeline")
async def append_timeline(request: Request, incident_id: str, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    plugin_id = _resolve_plugin_id(request, payload)
    if not plugin_id:
        return fail(ERR_CODE_INVALID_REQUEST, "plugin_id 必填", request_id=request_id, status_code=400)
    entry = service.add_incident_timeline(plugin_id, incident_id, payload)
    return ok(entry, message="timeline entry recorded", request_id=request_id)
