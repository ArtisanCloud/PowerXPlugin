import logging

from fastapi import APIRouter, Request

from app.contracts.response import (
    ERR_CODE_FORBIDDEN,
    ERR_CODE_INVALID_REQUEST,
    ERR_CODE_UNAUTHORIZED,
    fail,
    ok,
)
from app.middleware.tenant_context import get_tenant_context
from app.services.integration_service import IntegrationService

router = APIRouter()
service = IntegrationService()
logger = logging.getLogger("integration_http")


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


def _require_root(request: Request):
    request_id = _request_id(request)
    context = get_tenant_context(request)
    if _is_root_context(context):
        return None
    return fail(
        ERR_CODE_FORBIDDEN,
        "仅 root 可访问",
        request_id=request_id,
        status_code=403,
    )


def _is_root_context(context) -> bool:
    if not context:
        return False
    if bool(getattr(context, "is_root", False)):
        return True
    roles = getattr(context, "roles", None) or []
    normalized = {str(role).strip().lower() for role in roles if str(role).strip()}
    return any(role in normalized for role in ("root", "superadmin", "admin"))


def _auth_scheme_from_header(value: str) -> str:
    raw = (value or "").strip().lower()
    if raw.startswith("bearer "):
        return "bearer"
    if raw.startswith("apikey ") or raw.startswith("api-key ") or raw.startswith("api_key "):
        return "apikey"
    if raw:
        return "unknown"
    return "none"


def _collect_forward_headers(request: Request) -> dict[str, str]:
    headers: dict[str, str] = {}
    mock_value = (request.headers.get("X-PX-Use-Mock") or "").strip()
    if mock_value:
        headers["X-PX-Use-Mock"] = mock_value
    return headers


@router.post("/integration/dispatch")
async def dispatch(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    return ok(service.dispatch(payload), request_id=request_id)


@router.post("/integration/capabilities/invoke")
async def invoke_capability(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    root_err = _require_root(request)
    if root_err:
        return root_err
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    forward_headers = _collect_forward_headers(request)
    logger.info(
        "capability invoke accepted request_id=%s capability_id=%s action=%s inbound_auth=%s forwarded_auth=%s",
        request_id,
        (payload or {}).get("capabilityId"),
        (payload or {}).get("action"),
        _auth_scheme_from_header((request.headers.get("Authorization") or "").strip()),
        "Authorization" in forward_headers,
    )
    try:
        result = service.invoke_capability(payload, forward_headers=forward_headers, request_id=request_id)
    except Exception as exc:
        status_code = getattr(exc, "status_code", 500)
        trace_id = str(getattr(exc, "trace_id", "") or "").strip()
        details = getattr(exc, "details", None)
        warnings = getattr(exc, "warnings", None)
        logger.error(
            "capability invoke failed request_id=%s status=%s trace_id=%s error=%s details=%s",
            request_id,
            status_code,
            trace_id,
            str(exc),
            details,
        )
        response_details = details if isinstance(details, dict) else {}
        if warnings:
            response_details["warnings"] = warnings
        if trace_id:
            response_details["trace_id"] = trace_id
        return fail("UPSTREAM_ERROR", str(exc), details=response_details or None, request_id=request_id, status_code=status_code)
    return ok(result, request_id=request_id)


@router.get("/integration/grant-matrix")
async def list_grant_matrix(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"items": service.list_grant_matrix_overrides()}, request_id=request_id)


@router.post("/integration/grant-matrix")
async def submit_grant_matrix(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    return ok(service.submit_grant_matrix(payload), request_id=request_id)


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
    return ok(service.create_subscription(payload), request_id=request_id)


@router.get("/integration/webhooks/subscriptions")
async def list_subscriptions(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"items": service.list_subscriptions(None)}, request_id=request_id)


@router.post("/integration/webhooks/dlq/{attempt_id}/replay")
async def replay_dlq(request: Request, attempt_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.replay_attempt(attempt_id), request_id=request_id)


@router.post("/integration/secrets")
async def create_secret(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not (payload or {}).get("integration_type"):
        return fail(ERR_CODE_INVALID_REQUEST, "integration_type 必填", request_id=request_id, status_code=400)
    return ok(service.create_secret(payload), request_id=request_id)


@router.post("/integration/secrets/{secret_id}/rotate")
async def rotate_secret(request: Request, secret_id: str, payload: dict | None = None):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.rotate_secret(secret_id, payload), request_id=request_id)
