from fastapi import APIRouter, Request

from app.contracts.response import (
    ERR_CODE_INVALID_REQUEST,
    ERR_CODE_NOT_FOUND,
    ERR_CODE_UNAUTHORIZED,
    fail,
    ok,
)
from app.services.template_service import TemplateService

router = APIRouter(prefix="/admin")
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


@router.get("/templates")
async def list_templates(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(
        service.list_templates(dict(request.query_params)),
        request_id=request_id,
    )


@router.get("/templates/{template_id}")
async def get_template(request: Request, template_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    template = service.get_template(template_id)
    if not template:
        return fail(
            ERR_CODE_NOT_FOUND,
            "not found",
            request_id=request_id,
            status_code=404,
        )
    return ok(template, request_id=request_id)


@router.post("/templates")
async def create_template(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not (payload or {}).get("name") or not (payload or {}).get("description") or not (payload or {}).get("content"):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "name/description/content 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok(service.create_template(payload), request_id=request_id)


@router.put("/templates/{template_id}")
async def update_template(request: Request, template_id: str, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not (payload or {}).get("name") or not (payload or {}).get("description") or not (payload or {}).get("content"):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "name/description/content 必填",
            request_id=request_id,
            status_code=400,
        )
    template = service.update_template(template_id, payload)
    if not template:
        return fail(
            ERR_CODE_NOT_FOUND,
            "not found",
            request_id=request_id,
            status_code=404,
        )
    return ok(template, request_id=request_id)


@router.delete("/templates/{template_id}")
async def delete_template(request: Request, template_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    result = service.delete_template(template_id)
    if not result.get("ok"):
        return fail(
            ERR_CODE_NOT_FOUND,
            "not found",
            request_id=request_id,
            status_code=404,
        )
    return ok(result, request_id=request_id)
