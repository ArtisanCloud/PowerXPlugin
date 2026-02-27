from fastapi import APIRouter, Request

from app.contracts.response import (
    ERR_CODE_INVALID_REQUEST,
    ERR_CODE_UNAUTHORIZED,
    fail,
    ok,
)
from app.services.runtime_session_service import RuntimeSessionService
import os
from urllib import request as urlrequest
from urllib.error import URLError

from app.middleware.tenant_context import resolve_tenant_uuid

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


@router.post("/runtime/internal/ws-bus/publish")
async def ws_bus_publish(request: Request, payload: dict):
    request_id = _request_id(request)
    settings = getattr(request.app.state, "settings", None)
    if settings is not None and not (settings.dev_mode or settings.server_dev_mode):
        return fail("FORBIDDEN", "ws bus publish only available in dev mode", request_id=request_id, status_code=403)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    topic = str(payload.get("topic") or "").strip()
    if not topic:
        return fail(ERR_CODE_INVALID_REQUEST, "topic 必填", request_id=request_id, status_code=400)
    allowed = {
        "_topic.template.update",
        "_topic.audit.template.updated",
        "_topic.template.validate.completed",
        "_topic.template.batch_clone.completed",
        "_topic.template.update.completed",
    }
    if topic not in allowed:
        return fail(ERR_CODE_INVALID_REQUEST, "topic not allowed", request_id=request_id, status_code=400)
    tenant_uuid = str(payload.get("tenant_uuid") or "").strip()
    if not tenant_uuid:
        tenant_uuid = resolve_tenant_uuid(request) or ""
    trace_id = str(payload.get("trace_id") or "").strip()
    if os.getenv("POWERX_PROXY") == "1" and settings and settings.gateway_base_url:
        base = settings.gateway_base_url.rstrip("/")
        if base.endswith("/api/v1"):
            base = base[: -len("/api/v1")]
        endpoint = f"{base}/api/v1/admin/runtime/internal/ws-bus/publish"
        body = json.dumps(
            {"topic": topic, "payload": payload.get("payload"), "tenant_uuid": tenant_uuid, "trace_id": trace_id}
        ).encode("utf-8")
        req = urlrequest.Request(endpoint, data=body, method="POST")
        req.add_header("Content-Type", "application/json")
        raw_auth = (request.headers.get("authorization") or "").strip()
        if raw_auth:
            req.add_header("Authorization", raw_auth)
        if tenant_uuid:
            req.add_header("X-PowerX-Tenant", tenant_uuid)
        try:
            with urlrequest.urlopen(req, timeout=5) as resp:
                if resp.status >= 400:
                    return fail("UPSTREAM_ERROR", f"upstream rejected: {resp.status}", request_id=request_id, status_code=502)
        except URLError as exc:
            return fail("UPSTREAM_ERROR", str(exc), request_id=request_id, status_code=502)
        return ok({"ok": True}, request_id=request_id)

    hub = getattr(request.app.state, "ws_bus_hub", None)
    if hub is None:
        return fail("SERVICE_UNAVAILABLE", "ws bus not configured", request_id=request_id, status_code=503)
    await hub.publish(topic, payload.get("payload"), tenant_uuid=tenant_uuid, trace_id=trace_id)
    return ok({"ok": True}, request_id=request_id)


@router.post("/runtime/internal/ws-bus/grant")
async def ws_bus_grant(request: Request, payload: dict):
    request_id = _request_id(request)
    settings = getattr(request.app.state, "settings", None)
    if settings is not None and not (settings.dev_mode or settings.server_dev_mode):
        return fail("FORBIDDEN", "ws bus register only available in dev mode", request_id=request_id, status_code=403)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload or not payload.get("topics"):
        return fail(ERR_CODE_INVALID_REQUEST, "topics 必填", request_id=request_id, status_code=400)
    topics = [str(t or "").strip() for t in payload.get("topics") or []]
    topics = [t for t in topics if t]
    allowed = {
        "_topic.template.update",
        "_topic.audit.template.updated",
        "_topic.template.validate.completed",
        "_topic.template.batch_clone.completed",
        "_topic.template.update.completed",
    }
    for t in topics:
        if t not in allowed:
            return fail(ERR_CODE_INVALID_REQUEST, "topic not allowed", request_id=request_id, status_code=400)
    if os.getenv("POWERX_PROXY") == "1" and settings and settings.gateway_base_url:
        base = settings.gateway_base_url.rstrip("/")
        if base.endswith("/api/v1"):
            base = base[: -len("/api/v1")]
        endpoint = f"{base}/api/v1/admin/runtime/internal/ws-bus/grant"
        body = json.dumps({"topics": topics, "tenant_uuid": payload.get("tenant_uuid"), "trace_id": payload.get("trace_id")}).encode(
            "utf-8"
        )
        req = urlrequest.Request(endpoint, data=body, method="POST")
        req.add_header("Content-Type", "application/json")
        raw_auth = (request.headers.get("authorization") or "").strip()
        if raw_auth:
            req.add_header("Authorization", raw_auth)
        tenant_uuid = str(payload.get("tenant_uuid") or "").strip()
        if tenant_uuid:
            req.add_header("X-PowerX-Tenant", tenant_uuid)
        try:
            with urlrequest.urlopen(req, timeout=5) as resp:
                if resp.status >= 400:
                    return fail("UPSTREAM_ERROR", f"upstream rejected: {resp.status}", request_id=request_id, status_code=502)
        except URLError as exc:
            return fail("UPSTREAM_ERROR", str(exc), request_id=request_id, status_code=502)
    return ok({"ok": True, "topics": topics}, request_id=request_id)
