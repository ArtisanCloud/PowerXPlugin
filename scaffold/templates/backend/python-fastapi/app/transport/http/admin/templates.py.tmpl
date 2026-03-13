from fastapi import APIRouter, Request
from sqlalchemy.exc import SQLAlchemyError

from app.contracts.response import (
    ERR_CODE_FORBIDDEN,
    ERR_CODE_INTERNAL_ERROR,
    ERR_CODE_INVALID_REQUEST,
    ERR_CODE_NOT_FOUND,
    ERR_CODE_UNAUTHORIZED,
    fail,
    ok,
)
from app.middleware.tenant_context import get_tenant_context, resolve_tenant_uuid
from app.services.template_service import TemplateService

router = APIRouter(prefix="/admin")
public_router = APIRouter()
service = TemplateService()


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
    if context and context.is_root:
        return None
    return fail(
        ERR_CODE_FORBIDDEN,
        "仅 root 可访问",
        request_id=request_id,
        status_code=403,
    )


def _validate_template_payload(payload: dict):
    return (payload or {}).get("name") and (payload or {}).get("description") and (payload or {}).get("content")


def _batch_clone_response(payload: dict):
    return {
        "source_ids": payload.get("source_ids") or payload.get("sourceIds") or [],
        "copies": payload.get("copies") or 1,
        "name_prefix": payload.get("name_prefix") or payload.get("namePrefix"),
        "description_prefix": payload.get("description_prefix") or payload.get("descriptionPrefix"),
        "items": [],
    }

def _inject_tenant(payload: dict | None, request: Request) -> dict:
    body = dict(payload or {})
    tenant_uuid = (body.get("tenant_uuid") or body.get("tenantUuid") or "").strip()
    if tenant_uuid:
        return body
    resolved = resolve_tenant_uuid(request)
    if resolved:
        body["tenant_uuid"] = resolved
    return body


@public_router.get("/templates")
async def list_templates(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    root_err = _require_root(request)
    if root_err:
        return root_err
    params = dict(request.query_params)
    if not (params.get("tenant_uuid") or params.get("tenantUuid")):
        resolved = resolve_tenant_uuid(request)
        if resolved:
            params["tenant_uuid"] = resolved
    try:
        return ok(service.list_templates(params), request_id=request_id)
    except SQLAlchemyError as exc:
        return fail(
            ERR_CODE_INTERNAL_ERROR,
            "templates 查询失败",
            details={"reason": str(exc)},
            request_id=request_id,
            status_code=500,
        )


@public_router.get("/templates/{template_id}")
async def get_template(request: Request, template_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    root_err = _require_root(request)
    if root_err:
        return root_err
    template = service.get_template(template_id)
    if not template:
        return fail(ERR_CODE_NOT_FOUND, "not found", request_id=request_id, status_code=404)
    return ok(template, request_id=request_id)


@public_router.post("/templates")
async def create_template(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    root_err = _require_root(request)
    if root_err:
        return root_err
    if not _validate_template_payload(payload or {}):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "name/description/content 必填",
            request_id=request_id,
            status_code=400,
        )
    payload = _inject_tenant(payload, request)
    try:
        return ok(service.create_template(payload), request_id=request_id)
    except SQLAlchemyError as exc:
        return fail(
            ERR_CODE_INTERNAL_ERROR,
            "templates 保存失败",
            details={"reason": str(exc)},
            request_id=request_id,
            status_code=500,
        )


@public_router.put("/templates/{template_id}")
async def update_template(request: Request, template_id: str, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    root_err = _require_root(request)
    if root_err:
        return root_err
    if not _validate_template_payload(payload or {}):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "name/description/content 必填",
            request_id=request_id,
            status_code=400,
        )
    payload = _inject_tenant(payload, request)
    try:
        template = service.update_template(template_id, payload)
    except SQLAlchemyError as exc:
        return fail(
            ERR_CODE_INTERNAL_ERROR,
            "templates 更新失败",
            details={"reason": str(exc)},
            request_id=request_id,
            status_code=500,
        )
    if not template:
        return fail(ERR_CODE_NOT_FOUND, "not found", request_id=request_id, status_code=404)
    return ok(template, request_id=request_id)


@public_router.delete("/templates/{template_id}")
async def delete_template(request: Request, template_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    root_err = _require_root(request)
    if root_err:
        return root_err
    result = service.delete_template(template_id)
    if not result.get("ok"):
        return fail(ERR_CODE_NOT_FOUND, "not found", request_id=request_id, status_code=404)
    return ok(result, request_id=request_id)


@public_router.post("/templates/batch-clone")
async def batch_clone_templates(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    root_err = _require_root(request)
    if root_err:
        return root_err
    source_ids = (payload or {}).get("source_ids") or (payload or {}).get("sourceIds")
    if not source_ids:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "source_ids 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok(_batch_clone_response(payload or {}), request_id=request_id)


@public_router.post("/templates/{template_id}/validate")
async def validate_template(request: Request, template_id: str, payload: dict | None = None):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    root_err = _require_root(request)
    if root_err:
        return root_err
    return ok({"template_id": template_id, "valid": True, "errors": []}, request_id=request_id)


@router.post("/templates/batch-clone")
async def admin_batch_clone(request: Request, payload: dict):
    return await batch_clone_templates(request, payload)


@router.post("/templates/{template_id}/validate")
async def admin_validate_template(request: Request, template_id: str, payload: dict | None = None):
    return await validate_template(request, template_id, payload)
