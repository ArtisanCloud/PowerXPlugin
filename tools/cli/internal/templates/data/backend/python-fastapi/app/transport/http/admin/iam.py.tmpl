from fastapi import APIRouter, Request

from app.contracts.response import (
    ERR_CODE_INVALID_REQUEST,
    ERR_CODE_UNAUTHORIZED,
    fail,
    ok,
)
from app.middleware.tenant_context import resolve_tenant_uuid
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


def _resolve_tenant_uuid(request: Request, payload: dict | None = None) -> str:
    if payload:
        candidate = payload.get("tenant_uuid") or payload.get("tenantUuid")
        if candidate:
            return str(candidate).strip()
    candidate = request.query_params.get("tenant_uuid") or request.query_params.get("tenantUuid")
    if candidate:
        return str(candidate).strip()
    return str(resolve_tenant_uuid(request) or "").strip()


def _page(value: str | None) -> int:
    try:
        page = int(value or 1)
    except ValueError:
        return 1
    return page if page > 0 else 1


def _page_size(value: str | None) -> int:
    try:
        size = int(value or 20)
    except ValueError:
        return 20
    if size <= 0:
        return 20
    return min(size, 100)


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
    tenant_uuid = _resolve_tenant_uuid(request)
    if not tenant_uuid:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "tenant_uuid 必填",
            request_id=request_id,
            status_code=400,
        )
    params = dict(request.query_params)
    params["tenant_uuid"] = tenant_uuid
    result = service.list_roles(params)
    return ok({"items": result.get("items", [])}, request_id=request_id)


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
    tenant_uuid = _resolve_tenant_uuid(request, payload)
    if not tenant_uuid or not (payload or {}).get("code") or not (payload or {}).get("name"):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "tenant_uuid/code/name 必填",
            request_id=request_id,
            status_code=400,
        )
    payload = dict(payload or {})
    payload["tenant_uuid"] = tenant_uuid
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
    service.delete_role(role_id)
    return ok({"role_id": role_id}, message="deleted", request_id=request_id)


@router.put("/iam/roles/{role_id}/permissions")
async def update_role_permissions(request: Request, role_id: str, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = _resolve_tenant_uuid(request, payload)
    if not tenant_uuid or not (payload or {}).get("permission_ids"):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "tenant_uuid/permission_ids 必填",
            request_id=request_id,
            status_code=400,
        )
    payload = dict(payload or {})
    payload["tenant_uuid"] = tenant_uuid
    return ok(
        service.update_role_permissions(role_id, payload.get("permission_ids"), tenant_uuid),
        request_id=request_id,
    )


@router.post("/iam/roles/{role_id}/members")
async def add_role_members(request: Request, role_id: str, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = _resolve_tenant_uuid(request, payload)
    if not tenant_uuid or not (payload or {}).get("member_ids"):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "tenant_uuid/member_ids 必填",
            request_id=request_id,
            status_code=400,
        )
    payload = dict(payload or {})
    payload["tenant_uuid"] = tenant_uuid
    return ok(
        service.add_role_members(role_id, payload.get("member_ids"), tenant_uuid),
        message="added",
        request_id=request_id,
    )


@router.delete("/iam/roles/{role_id}/members")
async def remove_role_members(request: Request, role_id: str, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = _resolve_tenant_uuid(request, payload)
    if not tenant_uuid or not (payload or {}).get("member_ids"):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "tenant_uuid/member_ids 必填",
            request_id=request_id,
            status_code=400,
        )
    payload = dict(payload or {})
    payload["tenant_uuid"] = tenant_uuid
    return ok(
        service.remove_role_members(role_id, payload.get("member_ids"), tenant_uuid),
        message="removed",
        request_id=request_id,
    )


@router.get("/iam/permissions")
async def list_permissions(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.list_permissions(), request_id=request_id)


@router.get("/iam/audit/logs")
async def list_audit_logs(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = _resolve_tenant_uuid(request)
    if not tenant_uuid:
        return fail(ERR_CODE_INVALID_REQUEST, "tenant_uuid 必填", request_id=request_id, status_code=400)
    params = dict(request.query_params)
    params["tenant_uuid"] = tenant_uuid
    return ok(service.list_audit_logs(params), request_id=request_id)


@router.post("/iam/auth/local/sts")
async def mint_sts(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    return ok(service.mint_sts({}), request_id=request_id)


@router.get("/iam/departments")
async def list_departments(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = _resolve_tenant_uuid(request)
    if not tenant_uuid:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "tenant_uuid 必填",
            request_id=request_id,
            status_code=400,
        )
    params = dict(request.query_params)
    params["tenant_uuid"] = tenant_uuid
    return ok(service.list_departments(params), request_id=request_id)


@router.get("/iam/departments/tree")
async def list_department_tree(request: Request):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = _resolve_tenant_uuid(request)
    if not tenant_uuid:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "tenant_uuid 必填",
            request_id=request_id,
            status_code=400,
        )
    items = service.list_departments({"tenant_uuid": tenant_uuid}).get("items", [])
    nodes = {item.get("id"): {**item, "children": []} for item in items if item.get("id") is not None}
    roots = []
    for node in nodes.values():
        parent_id = node.get("parent_id")
        if parent_id and parent_id in nodes:
            nodes[parent_id]["children"].append(node)
        else:
            roots.append(node)
    return ok(roots, request_id=request_id)


@router.post("/iam/departments")
async def create_department(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = _resolve_tenant_uuid(request, payload)
    if not tenant_uuid or not (payload or {}).get("name"):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "tenant_uuid/name 必填",
            request_id=request_id,
            status_code=400,
        )
    payload = dict(payload or {})
    payload["tenant_uuid"] = tenant_uuid
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
    tenant_uuid = _resolve_tenant_uuid(request)
    if not tenant_uuid:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "tenant_uuid 必填",
            request_id=request_id,
            status_code=400,
        )
    page = _page(request.query_params.get("page"))
    page_size = _page_size(request.query_params.get("page_size") or request.query_params.get("pageSize"))
    params = dict(request.query_params)
    params["tenant_uuid"] = tenant_uuid
    result = service.list_members(params)
    return ok({"items": result.get("items", []), "page": page, "page_size": page_size}, request_id=request_id)


@router.post("/iam/members")
async def create_member(request: Request, payload: dict):
    request_id = _request_id(request)
    auth = _require_auth(request)
    if auth:
        return auth
    tenant_uuid = _resolve_tenant_uuid(request, payload)
    if not tenant_uuid or not (payload or {}).get("email"):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "tenant_uuid/email 必填",
            request_id=request_id,
            status_code=400,
        )
    payload = dict(payload or {})
    payload["tenant_uuid"] = tenant_uuid
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
    tenant_uuid = _resolve_tenant_uuid(request, payload)
    users = (payload or {}).get("users") or (payload or {}).get("members")
    if not tenant_uuid or not users:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "tenant_uuid/users 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok(service.bulk_import_members(tenant_uuid, users), request_id=request_id)


# Legacy alias routes for compatibility
@router.get("/iam/users")
async def list_users(request: Request):
    return await list_members(request)


@router.post("/iam/users")
async def create_user(request: Request, payload: dict):
    return await create_member(request, payload)


@router.patch("/iam/users/{member_id}")
async def update_user(request: Request, member_id: str, payload: dict):
    return await update_member(request, member_id, payload)


@router.post("/iam/users/import")
async def import_users(request: Request, payload: dict):
    return await import_members(request, payload)
