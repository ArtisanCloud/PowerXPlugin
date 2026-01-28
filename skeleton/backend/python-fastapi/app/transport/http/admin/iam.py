from fastapi import APIRouter, Request

from app.contracts.response import (
    ERR_CODE_INVALID_REQUEST,
    ERR_CODE_UNAUTHORIZED,
    fail,
    ok,
)
from app.services.iam_service import IAMService

router = APIRouter(prefix="/admin")
service = IAMService()


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


@router.get("/iam/tenants")
async def list_tenants(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(
        service.list_tenants(dict(request.query_params)),
        request_id=request_id,
    )


@router.get("/iam/tenants/{tenant_id}")
async def get_tenant(request: Request, tenant_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.get_tenant(tenant_id), request_id=request_id)


@router.post("/iam/tenants")
async def create_tenant(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not (payload or {}).get("key") or not (payload or {}).get("name"):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "key/name 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok(service.create_tenant(payload), request_id=request_id, status_code=201)


@router.patch("/iam/tenants/{tenant_id}")
async def update_tenant(request: Request, tenant_id: str, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "请求体不能为空",
            request_id=request_id,
            status_code=400,
        )
    return ok(service.update_tenant(tenant_id, payload), request_id=request_id)


@router.get("/iam/roles")
async def list_roles(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = request.query_params.get("tenant_uuid") or request.query_params.get(
        "tenantUuid"
    )
    if not tenant_uuid:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "tenant_uuid 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok(
        service.list_roles(dict(request.query_params)),
        request_id=request_id,
    )


@router.get("/iam/roles/{role_id}")
async def get_role(request: Request, role_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.get_role(role_id), request_id=request_id)


@router.post("/iam/roles")
async def create_role(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = (payload or {}).get("tenant_uuid") or (payload or {}).get("tenantUuid")
    if not tenant_uuid or not (payload or {}).get("code") or not (payload or {}).get("name"):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "tenant_uuid/code/name 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok(service.create_role(payload), request_id=request_id, status_code=201)


@router.patch("/iam/roles/{role_id}")
async def update_role(request: Request, role_id: str, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "请求体不能为空",
            request_id=request_id,
            status_code=400,
        )
    return ok(service.update_role(role_id, payload), request_id=request_id)


@router.delete("/iam/roles/{role_id}")
async def delete_role(request: Request, role_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.delete_role(role_id), request_id=request_id)


@router.get("/iam/roles/{role_id}/permissions")
async def list_role_permissions(request: Request, role_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok([], request_id=request_id)


@router.put("/iam/roles/{role_id}/permissions")
async def update_role_permissions(request: Request, role_id: str, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = (payload or {}).get("tenant_uuid") or (payload or {}).get("tenantUuid")
    if not tenant_uuid or not (payload or {}).get("permission_ids"):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "tenant_uuid/permission_ids 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok(payload, request_id=request_id)


@router.get("/iam/roles/{role_id}/members")
async def list_role_members(request: Request, role_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok([], request_id=request_id)


@router.post("/iam/roles/{role_id}/members")
async def add_role_members(request: Request, role_id: str, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = (payload or {}).get("tenant_uuid") or (payload or {}).get("tenantUuid")
    if not tenant_uuid or not (payload or {}).get("member_ids"):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "tenant_uuid/member_ids 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok(payload, request_id=request_id)


@router.delete("/iam/roles/{role_id}/members")
async def remove_role_members(request: Request, role_id: str, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = (payload or {}).get("tenant_uuid") or (payload or {}).get("tenantUuid")
    if not tenant_uuid or not (payload or {}).get("member_ids"):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "tenant_uuid/member_ids 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok({"deleted": True}, request_id=request_id)


@router.get("/iam/permissions")
async def list_permissions(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.list_permissions(), request_id=request_id)


@router.get("/iam/permissions/catalog")
async def list_permission_catalog(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({}, request_id=request_id)


@router.post("/iam/permissions")
async def create_permission(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not (payload or {}).get("resource") or not (payload or {}).get("action"):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "resource/action 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok(service.create_permission(payload), request_id=request_id, status_code=201)


@router.put("/iam/permissions/{permission_id}")
async def update_permission(request: Request, permission_id: str, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "请求体不能为空",
            request_id=request_id,
            status_code=400,
        )
    return ok(service.update_permission(permission_id, payload), request_id=request_id)


@router.delete("/iam/permissions/{permission_id}")
async def delete_permission(request: Request, permission_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.delete_permission(permission_id), request_id=request_id)


@router.post("/iam/permissions/sync")
async def sync_permissions(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok({"ok": True}, request_id=request_id)


@router.get("/iam/departments")
async def list_departments(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = request.query_params.get("tenant_uuid") or request.query_params.get(
        "tenantUuid"
    )
    if not tenant_uuid:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "tenant_uuid 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok(
        service.list_departments(dict(request.query_params)),
        request_id=request_id,
    )


@router.get("/iam/departments/tree")
async def list_department_tree(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok([], request_id=request_id)


@router.post("/iam/departments")
async def create_department(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not (payload or {}).get("tenant_uuid") or not (payload or {}).get("name"):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "tenant_uuid/name 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok(service.create_department(payload), request_id=request_id, status_code=201)


@router.patch("/iam/departments/{department_id}")
async def update_department(request: Request, department_id: str, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "请求体不能为空",
            request_id=request_id,
            status_code=400,
        )
    return ok(service.update_department(department_id, payload), request_id=request_id)


@router.delete("/iam/departments/{department_id}")
async def delete_department(request: Request, department_id: str):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.delete_department(department_id), request_id=request_id)


@router.get("/iam/members")
async def list_members(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = request.query_params.get("tenant_uuid") or request.query_params.get(
        "tenantUuid"
    )
    if not tenant_uuid:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "tenant_uuid 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok(
        service.list_members(dict(request.query_params)),
        request_id=request_id,
    )


@router.post("/iam/members")
async def create_member(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not (payload or {}).get("tenant_uuid") or not (payload or {}).get("email"):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "tenant_uuid/email 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok(service.create_member(payload), request_id=request_id, status_code=201)


@router.patch("/iam/members/{member_id}")
async def update_member(request: Request, member_id: str, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    if not payload:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "请求体不能为空",
            request_id=request_id,
            status_code=400,
        )
    return ok(service.update_member(member_id, payload), request_id=request_id)


@router.post("/iam/members/import")
async def import_members(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = (payload or {}).get("tenant_uuid") or (payload or {}).get("tenantUuid")
    users = (payload or {}).get("users") or (payload or {}).get("members")
    if not tenant_uuid or not users:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "tenant_uuid/users 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok(payload, request_id=request_id)
