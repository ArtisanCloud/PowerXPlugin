from fastapi import APIRouter, Request

from app.contracts.response import (
    ERR_CODE_INVALID_REQUEST,
    ERR_CODE_UNAUTHORIZED,
    ok,
    fail,
)
from app.services.auth_service import AuthService

router = APIRouter(prefix="/admin")
service = AuthService()


def _request_id(request: Request) -> str | None:
    return request.headers.get("X-Request-ID") or request.headers.get("Request-ID")


def _bearer_token(request: Request) -> str:
    raw = (request.headers.get("Authorization") or "").strip()
    if not raw:
        return ""
    if raw.lower().startswith("bearer "):
        return raw[7:].strip()
    return raw


@router.post("/user/auth/login")
async def login(request: Request, payload: dict):
    request_id = _request_id(request)
    identifier = (payload or {}).get("identifier")
    password = (payload or {}).get("password")
    if not identifier or not password:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "identifier/password 必填",
            request_id=request_id,
            status_code=400,
        )
    result = service.login(payload or {})
    if not result or not result.get("access_token"):
        return fail(
            ERR_CODE_UNAUTHORIZED,
            "认证失败，请重新登录",
            request_id=request_id,
            status_code=401,
        )
    return ok(result, request_id=request_id)


@router.post("/user/auth/register")
async def register(request: Request, payload: dict):
    request_id = _request_id(request)
    identifier = (payload or {}).get("username") or (payload or {}).get("email")
    if not identifier or not (payload or {}).get("password"):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "username/email/password 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok(service.register(payload or {}), request_id=request_id)


@router.post("/user/auth/logout")
async def logout(request: Request, payload: dict):
    request_id = _request_id(request)
    refresh_token = (payload or {}).get("refresh_token") or (payload or {}).get("refreshToken")
    if not refresh_token:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "refresh_token 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok(service.logout(payload or {}), request_id=request_id)


@router.post("/user/auth/refresh")
async def refresh(request: Request, payload: dict):
    request_id = _request_id(request)
    refresh_token = (payload or {}).get("refresh_token") or (payload or {}).get("refreshToken")
    if not refresh_token:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "refresh_token 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok(service.refresh(payload or {}), request_id=request_id)


@router.get("/user/auth/me")
async def me(request: Request):
    request_id = _request_id(request)
    if not _bearer_token(request):
        return fail(
            ERR_CODE_UNAUTHORIZED,
            "缺少 Authorization Bearer token",
            request_id=request_id,
            status_code=401,
        )
    return ok(service.me(), request_id=request_id)


@router.put("/user/auth/profile")
async def profile(request: Request, payload: dict):
    request_id = _request_id(request)
    if not _bearer_token(request):
        return fail(
            ERR_CODE_UNAUTHORIZED,
            "缺少 Authorization Bearer token",
            request_id=request_id,
            status_code=401,
        )
    if not payload:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "请求体不能为空",
            request_id=request_id,
            status_code=400,
        )
    return ok(service.profile(payload), request_id=request_id)


@router.post("/user/auth/change-password")
async def change_password(request: Request, payload: dict):
    request_id = _request_id(request)
    if not _bearer_token(request):
        return fail(
            ERR_CODE_UNAUTHORIZED,
            "缺少 Authorization Bearer token",
            request_id=request_id,
            status_code=401,
        )
    old_password = (payload or {}).get("oldPassword") or (payload or {}).get("old_password")
    new_password = (payload or {}).get("newPassword") or (payload or {}).get("new_password")
    confirm_password = (payload or {}).get("confirmPassword") or (payload or {}).get("confirm_password")
    if not old_password or not new_password or not confirm_password:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "oldPassword/newPassword/confirmPassword 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok(service.change_password(payload or {}), request_id=request_id)


@router.post("/user/auth/reset-password")
async def reset_password(request: Request, payload: dict):
    request_id = _request_id(request)
    if not (payload or {}).get("email"):
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "email 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok(service.reset_password(payload or {}), request_id=request_id)


@router.post("/user/auth/reset-password/confirm")
async def reset_password_confirm(request: Request, payload: dict):
    request_id = _request_id(request)
    token = (payload or {}).get("token")
    new_password = (payload or {}).get("newPassword") or (payload or {}).get("new_password")
    confirm_password = (payload or {}).get("confirmPassword") or (payload or {}).get("confirm_password")
    if not token or not new_password or not confirm_password:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "token/newPassword/confirmPassword 必填",
            request_id=request_id,
            status_code=400,
        )
    return ok(service.reset_password_confirm(payload or {}), request_id=request_id)


@router.get("/user/auth/validate")
async def validate(request: Request):
    request_id = _request_id(request)
    if not _bearer_token(request):
        return fail(
            ERR_CODE_UNAUTHORIZED,
            "缺少 Authorization Bearer token",
            request_id=request_id,
            status_code=401,
        )
    return ok(service.validate(), request_id=request_id)


@router.get("/user/auth/permissions")
async def permissions(request: Request):
    request_id = _request_id(request)
    if not _bearer_token(request):
        return fail(
            ERR_CODE_UNAUTHORIZED,
            "缺少 Authorization Bearer token",
            request_id=request_id,
            status_code=401,
        )
    return ok(service.permissions(), request_id=request_id)

@router.get("/user/auth/me/context")
async def me_context(request: Request):
    request_id = _request_id(request)
    if not _bearer_token(request):
        return fail(
            ERR_CODE_UNAUTHORIZED,
            "缺少 Authorization Bearer token",
            request_id=request_id,
            status_code=401,
        )
    return ok({}, request_id=request_id)


@router.get("/user/auth/me/tenants")
async def me_tenants(request: Request):
    request_id = _request_id(request)
    if not _bearer_token(request):
        return fail(
            ERR_CODE_UNAUTHORIZED,
            "缺少 Authorization Bearer token",
            request_id=request_id,
            status_code=401,
        )
    return ok([], request_id=request_id)


@router.post("/user/auth/me/switch-tenant")
async def switch_tenant(request: Request, payload: dict):
    request_id = _request_id(request)
    if not _bearer_token(request):
        return fail(
            ERR_CODE_UNAUTHORIZED,
            "缺少 Authorization Bearer token",
            request_id=request_id,
            status_code=401,
        )
    if not payload:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "请求体不能为空",
            request_id=request_id,
            status_code=400,
        )
    return ok(payload, request_id=request_id)


@router.get("/user/auth/me/roles")
async def me_roles(request: Request):
    request_id = _request_id(request)
    if not _bearer_token(request):
        return fail(
            ERR_CODE_UNAUTHORIZED,
            "缺少 Authorization Bearer token",
            request_id=request_id,
            status_code=401,
        )
    return ok([], request_id=request_id)


@router.get("/user/auth/me/departments")
async def me_departments(request: Request):
    request_id = _request_id(request)
    if not _bearer_token(request):
        return fail(
            ERR_CODE_UNAUTHORIZED,
            "缺少 Authorization Bearer token",
            request_id=request_id,
            status_code=401,
        )
    return ok([], request_id=request_id)


@router.post("/user/auth/me/avatar")
async def me_avatar(request: Request, payload: dict):
    request_id = _request_id(request)
    if not _bearer_token(request):
        return fail(
            ERR_CODE_UNAUTHORIZED,
            "缺少 Authorization Bearer token",
            request_id=request_id,
            status_code=401,
        )
    if not payload:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "请求体不能为空",
            request_id=request_id,
            status_code=400,
        )
    return ok(payload, request_id=request_id)


@router.post("/user/auth/me/check-permission")
async def check_permission(request: Request, payload: dict):
    request_id = _request_id(request)
    if not _bearer_token(request):
        return fail(
            ERR_CODE_UNAUTHORIZED,
            "缺少 Authorization Bearer token",
            request_id=request_id,
            status_code=401,
        )
    if not payload:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "请求体不能为空",
            request_id=request_id,
            status_code=400,
        )
    return ok({"has_permission": True}, request_id=request_id)


@router.get("/users")
async def list_users():
    return ok([])


@router.get("/users/{user_id}")
async def get_user(user_id: str):
    return ok({})


@router.post("/users")
async def create_user(payload: dict):
    return ok(payload)


@router.put("/users/{user_id}")
async def update_user(user_id: str, payload: dict):
    return ok(payload)


@router.delete("/users/{user_id}")
async def delete_user(user_id: str):
    return ok({"deleted": True})


@router.post("/users/batch-delete")
async def delete_users(payload: dict):
    return ok({"deleted": True})


@router.patch("/users/{user_id}/status")
async def update_user_status(user_id: str, payload: dict):
    return ok({"updated": True})


@router.get("/roles")
async def list_roles():
    return ok([])


@router.get("/roles/{role_id}")
async def get_role(role_id: str):
    return ok({})


@router.post("/roles")
async def create_role(payload: dict):
    return ok(payload)


@router.put("/roles/{role_id}")
async def update_role(role_id: str, payload: dict):
    return ok(payload)


@router.delete("/roles/{role_id}")
async def delete_role(role_id: str):
    return ok({"deleted": True})


@router.post("/roles/{role_id}/permissions")
async def update_role_permissions(role_id: str, payload: dict):
    return ok(payload)


@router.get("/departments")
async def list_departments():
    return ok([])


@router.get("/departments/{department_id}")
async def get_department(department_id: str):
    return ok({})


@router.post("/departments")
async def create_department(payload: dict):
    return ok(payload)


@router.put("/departments/{department_id}")
async def update_department(department_id: str, payload: dict):
    return ok(payload)


@router.delete("/departments/{department_id}")
async def delete_department(department_id: str):
    return ok({"deleted": True})


@router.get("/departments/tree")
async def get_department_tree():
    return ok([])


@router.get("/permissions")
async def list_permissions():
    return ok([])


@router.get("/permissions/groups")
async def list_permission_groups():
    return ok({})
