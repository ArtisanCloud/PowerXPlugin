from fastapi import APIRouter, Request

from app.contracts.response import (
    ERR_CODE_INVALID_REQUEST,
    ERR_CODE_UNAUTHORIZED,
    fail,
    ok,
)
from app.middleware.tenant_context import resolve_tenant_uuid
from app.services.marketplace_service import MarketplaceService
from app.services.operations_service import OperationsService

router = APIRouter(prefix="/admin")
public_router = APIRouter()
service = MarketplaceService()
operations_service = OperationsService()


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


def _resolve_tenant_uuid(request: Request, payload: dict | None = None) -> str:
    if payload:
        candidate = payload.get("tenant_uuid") or payload.get("tenantUuid")
        if candidate:
            return str(candidate).strip()
    candidate = request.query_params.get("tenant_uuid") or request.query_params.get("tenantUuid")
    if candidate:
        return str(candidate).strip()
    return str(resolve_tenant_uuid(request) or "").strip()


@router.get("/marketplace/listings")
async def list_listings(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = _resolve_tenant_uuid(request)
    if not tenant_uuid:
        return fail(ERR_CODE_INVALID_REQUEST, "tenant_uuid 必填", request_id=request_id, status_code=400)
    status = request.query_params.get("status")
    items = service.list_listings(tenant_uuid, status)
    return ok({"items": items}, request_id=request_id)


@router.post("/marketplace/listings")
async def create_listing(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    return ok(service.create_listing(payload), request_id=request_id)


@router.get("/marketplace/listings/{listing_id}")
async def get_listing(request: Request, listing_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.get_listing(listing_id), request_id=request_id)


@router.patch("/marketplace/listings/{listing_id}")
async def update_listing(request: Request, listing_id: str, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    return ok(service.update_listing(listing_id, payload), request_id=request_id)


@router.post("/marketplace/listings/{listing_id}/review")
async def review_listing(request: Request, listing_id: str, payload: dict | None = None):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.update_listing_status(listing_id, "in_review", payload or {}), request_id=request_id)


@router.post("/marketplace/listings/{listing_id}/publish")
async def publish_listing(request: Request, listing_id: str, payload: dict | None = None):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.update_listing_status(listing_id, "published", payload or {}), request_id=request_id)


@router.post("/marketplace/listings/{listing_id}/suspend")
async def suspend_listing(request: Request, listing_id: str, payload: dict | None = None):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.update_listing_status(listing_id, "suspended", payload or {}), request_id=request_id)


@router.post("/marketplace/checklist/graphql")
async def checklist_graphql(request: Request, payload: dict | None = None):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"data": {}}, request_id=request_id)


@router.get("/marketplace/recommendation/config")
async def get_recommendation_config(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"config": {}}, request_id=request_id)


@router.post("/marketplace/recommendation/sync")
async def trigger_recommendation_sync(request: Request, payload: dict | None = None):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"ok": True}, request_id=request_id)


@router.patch("/marketplace/recommendation/experiment")
async def update_recommendation_experiment(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    return ok(payload, request_id=request_id)


@router.post("/marketplace/usage")
async def ingest_usage(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(ERR_CODE_INVALID_REQUEST, "payload 必填", request_id=request_id, status_code=400)
    tenant_uuid = _resolve_tenant_uuid(request, payload)
    payload = dict(payload or {})
    if tenant_uuid:
        payload["tenant_uuid"] = tenant_uuid
    return ok(service.ingest_usage(payload), request_id=request_id)


@router.get("/marketplace/usage/tenants/{tenant_id}/licenses/{license_id}/metrics")
async def get_usage_metrics(request: Request, tenant_id: str, license_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = _resolve_tenant_uuid(request)
    if tenant_uuid and tenant_id and tenant_uuid != tenant_id:
        return fail(ERR_CODE_INVALID_REQUEST, "tenant_uuid mismatch", request_id=request_id, status_code=400)
    return ok(service.list_usage_metrics(tenant_id, license_id), request_id=request_id)


@router.get("/marketplace/revenue-share/reports")
async def list_revenue_reports(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = _resolve_tenant_uuid(request)
    if not tenant_uuid:
        return fail(ERR_CODE_INVALID_REQUEST, "tenant_uuid 必填", request_id=request_id, status_code=400)
    return ok({"items": service.list_revenue_reports(tenant_uuid)}, request_id=request_id)


# Public marketplace routes (minimal)
@public_router.get("/marketplace/listings")
async def public_listings(request: Request):
    request_id = _request_id(request)
    return ok({"items": service.list_listings()}, request_id=request_id)


@public_router.get("/marketplace/sla/{plugin_id}")
async def public_sla(request: Request, plugin_id: str):
    request_id = _request_id(request)
    items = operations_service.list_sla_profiles(plugin_id)
    return ok({"items": items}, request_id=request_id)
