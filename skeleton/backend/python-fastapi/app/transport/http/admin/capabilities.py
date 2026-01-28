from fastapi import APIRouter, Request

from app.contracts.response import (
    ERR_CODE_INVALID_REQUEST,
    ERR_CODE_UNAUTHORIZED,
    fail,
    ok,
)
from app.services.capability_service import CapabilityService

router = APIRouter(prefix="/admin")
service = CapabilityService()


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


@router.get("/capabilities")
async def list_capabilities(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.list_capabilities(), request_id=request_id)


@router.get("/capabilities/register/template")
async def register_template(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.register_template(), request_id=request_id)


@router.post("/capabilities/register")
async def register(request: Request, payload: dict):
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
    return ok(service.register(payload), request_id=request_id)


@router.post("/capabilities/register/validate")
async def validate(request: Request, payload: dict):
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
    return ok(service.validate(payload), request_id=request_id)


@router.get("/capabilities/lifecycle/template")
async def lifecycle_template(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.lifecycle_template(), request_id=request_id)


@router.get("/capabilities/lifecycle")
async def list_lifecycle(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.list_lifecycle(), request_id=request_id)


@router.post("/capabilities/lifecycle")
async def create_lifecycle(request: Request, payload: dict):
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
    return ok(service.create_lifecycle(payload), request_id=request_id)


@router.post("/capabilities/lifecycle/{plan_id}/status")
async def update_lifecycle_status(request: Request, plan_id: str, payload: dict):
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
    return ok(service.update_lifecycle_status(plan_id, payload), request_id=request_id)


@router.get("/capabilities/exposure/template")
async def exposure_template(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.exposure_template(), request_id=request_id)


@router.get("/capabilities/exposure/{capability_id}")
async def exposure_detail(request: Request, capability_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.exposure_detail(capability_id), request_id=request_id)


@router.put("/capabilities/exposure/{capability_id}")
async def update_exposure(request: Request, capability_id: str, payload: dict):
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
    return ok(service.update_exposure(capability_id, payload), request_id=request_id)


@router.get("/capabilities/quotas/{capability_id}")
async def list_quotas(request: Request, capability_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.list_quotas(capability_id), request_id=request_id)


@router.post("/capabilities/quotas/{capability_id}")
async def update_quotas(request: Request, capability_id: str, payload: dict):
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
    return ok(service.update_quotas(capability_id, payload), request_id=request_id)
