from datetime import datetime, timedelta
from typing import Any

from fastapi import APIRouter, Request

from app.contracts.response import (
    ERR_CODE_INVALID_REQUEST,
    ERR_CODE_UNAUTHORIZED,
    fail,
    ok,
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


def _map_tokens(tokens: dict) -> dict:
    if not tokens:
        return {}
    expires_in = tokens.get("expires_in") or 0
    expires_at = tokens.get("expires_at")
    if not expires_at and expires_in:
        expires_at = int((datetime.utcnow() + timedelta(seconds=int(expires_in))).timestamp() * 1000)
    return {
        "token_type": tokens.get("token_type") or "Bearer",
        "access_token": tokens.get("access_token") or "",
        "refresh_token": tokens.get("refresh_token") or "",
        "expires_in": expires_in,
        "expires_at": expires_at or 0,
        "scope": tokens.get("scope") or "",
        "policy_version": tokens.get("policy_version"),
        "plugin_id": tokens.get("plugin_id"),
    }


def _map_me_context(payload: dict) -> dict:
    user = payload.get("user") or {}
    member = payload.get("member") or {}
    tenant = payload.get("tenant") or {}

    tenant_uuid = tenant.get("uuid") or member.get("tenant_uuid") or ""
    tenant_info: dict[str, Any] = {
        "uuid": tenant_uuid,
        "key": tenant.get("key") or "",
        "name": tenant.get("name") or "",
    }
    legacy_id = tenant.get("id")
    if isinstance(legacy_id, int) and legacy_id > 0:
        tenant_info["legacy_id"] = legacy_id

    member_id = member.get("id") or member.get("member_id")
    is_root = bool(user.get("is_root"))

    context = {
        "tenant": tenant_info,
        "is_root": is_root,
        "current_tenant_uuid": tenant_uuid,
        "current_member_id": member_id,
        "user": {
            "id": user.get("id"),
            "username": member.get("username") or user.get("email") or "",
            "email": user.get("email") or "",
            "display_name": user.get("display_name") or member.get("display_name") or "",
            "is_root": is_root,
        },
        "roles": [],
        "permissions": [],
        "policy_version": None,
    }
    members = []
    if tenant_uuid:
        members.append(
            {
                "tenant_uuid": tenant_uuid,
                "tenant_name": tenant_info.get("name"),
                "member_id": member_id,
                "is_admin": is_root,
            }
        )
    context["members"] = members
    if user.get("plugin_id"):
        context["plugin_id"] = user.get("plugin_id")
    return context


@router.post("/user/auth/login")
async def login(request: Request, payload: dict):
    request_id = _request_id(request)
    identifier = (payload or {}).get("identifier")
    password = (payload or {}).get("password")
    if not identifier or not password:
        return fail(
            ERR_CODE_INVALID_REQUEST,
            "参数错误: identifier/password 必填",
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
    return ok(_map_tokens(result), request_id=request_id)


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
    service.logout(payload or {})
    return ok({"ok": True}, request_id=request_id)


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
    result = service.refresh(payload or {})
    return ok(_map_tokens(result), request_id=request_id)


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
    return ok(_map_me_context(service.me()), request_id=request_id)


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
