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


def _tenant_uuid(request: Request) -> str:
    return (
        request.headers.get("tenant_uuid")
        or request.headers.get("Tenant-UUID")
        or request.headers.get("X-Tenant-UUID")
        or ""
    ).strip()


@router.get("/capabilities")
async def list_capabilities(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    source = request.query_params.get("source")
    return ok(
        service.list_capabilities(
            source=source,
            bearer_token=_bearer_token(request),
            tenant_uuid=_tenant_uuid(request),
        ),
        request_id=request_id,
    )


@router.get("/capabilities/sources")
async def list_capability_sources(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.list_sources(), request_id=request_id)


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
    capability_id = request.query_params.get("capability_id")
    plans = service.list_lifecycle()
    if capability_id:
        plans = [plan for plan in plans if plan.get("capability_id") == capability_id]
    return ok({"capability_id": capability_id, "plans": plans}, request_id=request_id)


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
    return ok({"capability_id": capability_id, "package": service.exposure_detail(capability_id)}, request_id=request_id)


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
    return ok({"capability_id": capability_id, "quotas": service.list_quotas(capability_id)}, request_id=request_id)


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

@router.get("/capabilities/reviews/{capability_id}")
async def review_list(request: Request, capability_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"capability_id": capability_id, "tasks": service.list_reviews(capability_id)}, request_id=request_id)


@router.post("/capabilities/reviews/{capability_id}/resubmit")
async def review_resubmit(request: Request, capability_id: str, payload: dict | None = None):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"capability_id": capability_id, "tasks": service.resubmit_review(capability_id, payload)}, request_id=request_id)


@router.post("/capabilities/reviews/tasks/{task_id}/comments")
async def review_add_comment(request: Request, task_id: str, payload: dict | None = None):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.add_review_comment(task_id, payload), request_id=request_id)


@router.post("/capabilities/reviews/tasks/{task_id}/decision")
async def review_decide(request: Request, task_id: str, payload: dict | None = None):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.decide_review(task_id, payload), request_id=request_id)
