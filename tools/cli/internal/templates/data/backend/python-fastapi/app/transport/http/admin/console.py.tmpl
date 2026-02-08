from fastapi import APIRouter, Request

from app.contracts.response import (
    ERR_CODE_INVALID_REQUEST,
    ERR_CODE_UNAUTHORIZED,
    fail,
    ok,
)
from app.services.admin_console_service import AdminConsoleService
from app.services.integration_service import IntegrationService

router = APIRouter(prefix="/admin")
console_service = AdminConsoleService()
integration_service = IntegrationService()


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


@router.get("/dev-console/config/sections")
async def list_config_sections(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"items": console_service.list_config_sections()}, request_id=request_id)


@router.put("/dev-console/config/sections/{section_key}")
async def update_config_section(request: Request, section_key: str, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    return ok(console_service.update_config_section(section_key, payload), request_id=request_id)


@router.get("/dev-console/audit/events")
async def list_audit_events(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"items": console_service.list_audit_events()}, request_id=request_id)


@router.get("/dev-console/audit/export")
async def export_audit_events(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(console_service.export_audit_events(), request_id=request_id)


@router.get("/dev-console/jobs/runs")
async def list_job_runs(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"items": console_service.list_job_runs()}, request_id=request_id)


@router.post("/dev-console/jobs/runs/{run_id}/retry")
async def retry_job_run(request: Request, run_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(console_service.retry_job_run(run_id), request_id=request_id)


@router.post("/dev-console/safe-ops/actions")
async def execute_safe_op(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    return ok(console_service.execute_safe_op(payload), request_id=request_id)


@router.get("/dev-console/troubleshooting/summary")
async def troubleshooting_summary(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(console_service.troubleshoot_summary(), request_id=request_id)


@router.get("/dev-console/webhooks/attempts")
async def list_webhook_attempts(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    items = integration_service.list_attempts(None)
    return ok({"items": items}, request_id=request_id)


@router.get("/dev-console/webhooks/attempts/{attempt_id}")
async def get_webhook_attempt(request: Request, attempt_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(integration_service.get_attempt(attempt_id), request_id=request_id)
