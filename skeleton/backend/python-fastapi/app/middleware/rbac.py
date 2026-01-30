from __future__ import annotations

import base64
import json
import os
import fnmatch
from dataclasses import dataclass, field
from typing import Any, Callable

from fastapi import Request
from fastapi.responses import JSONResponse

from app.config.settings import Settings
from app.middleware.tenant_context import TenantContext, get_tenant_context


@dataclass
class Permission:
    resource: str
    action: str


@dataclass
class RBACConfig:
    enabled: bool = True
    default_deny: bool = True
    super_admin_roles: list[str] = field(default_factory=lambda: ["superadmin", "admin"])
    route_permissions: dict[str, Permission] = field(default_factory=dict)
    delegate_to_powerx: bool = False
    powerx_issuer: str = ""
    powerx_audience: str = ""
    plugin_id: str = ""


def build_rbac_config(settings: Settings) -> RBACConfig:
    delegate = _should_delegate_to_powerx()
    issuer = os.getenv("POWERX_SECURITY_JWT_ISSUER", "").strip()
    audience = os.getenv("POWERX_SECURITY_JWT_AUDIENCE", "").strip()
    plugin_id = os.getenv("POWERX_PLUGIN_ID", "").strip()
    if not audience and plugin_id:
        audience = f"plugin:{plugin_id}"
    return RBACConfig(
        enabled=True,
        default_deny=True,
        super_admin_roles=["superadmin", "admin"],
        route_permissions={},
        delegate_to_powerx=delegate,
        powerx_issuer=issuer,
        powerx_audience=audience,
        plugin_id=plugin_id,
    )


async def rbac_middleware(request: Request, call_next: Callable):
    cfg: RBACConfig = getattr(request.app.state, "rbac_cfg", None)
    if cfg is None or not cfg.enabled:
        return await call_next(request)
    if request.method.upper() == "OPTIONS":
        return await call_next(request)

    path = _strip_plugin_prefix(request.url.path)
    template_path = _route_template(request)
    match_path = _strip_plugin_prefix(template_path) if template_path else path
    api_prefix = getattr(request.app.state, "settings", None).api_prefix if getattr(request.app.state, "settings", None) else "/api/v1"
    prefix = api_prefix.rstrip("/")
    if path.startswith(f"{prefix}/internal/") or path.startswith(f"{prefix}/agent/") or _is_health_endpoint(path, prefix):
        return await call_next(request)

    if cfg.delegate_to_powerx:
        if _allow_powerx_delegate(request, cfg):
            return await call_next(request)
        return JSONResponse(
            status_code=401,
            content={
                "error": "PowerX delegated authentication required",
                "hint": "请在宿主 PowerX 登录后再访问插件，或设置 POWERX_PROXY=0 运行 Standalone 模式",
            },
        )

    tc = get_tenant_context(request)
    if tc is None:
        return JSONResponse(status_code=401, content={"error": "Role Authentication required"})
    if _is_super_admin(tc.roles, cfg.super_admin_roles):
        return await call_next(request)

    perm, has = _match_route(request.method, match_path, cfg.route_permissions)
    if has:
        perm = _normalize_permission(perm, cfg)
    else:
        inferred = _infer_permission(request.method, path)
        if inferred:
            perm = _normalize_permission(inferred, cfg)
            has = True

    pass_rbac = (not has and not cfg.default_deny) or (has and _has_perm(tc.permissions, perm))
    if not pass_rbac:
        resource = perm.resource if perm else "unknown"
        action = perm.action if perm else "unknown"
        return JSONResponse(
            status_code=403,
            content={
                "error": "Insufficient permissions",
                "required_resource": resource,
                "required_action": action,
            },
        )

    return await call_next(request)


def _should_delegate_to_powerx() -> bool:
    value = os.getenv("POWERX_RBAC_DELEGATE", "").strip().lower()
    if value in {"1", "true", "yes", "on"}:
        return True
    if value in {"0", "false", "no", "off"}:
        return False
    return os.getenv("POWERX_PROXY") == "1"


def _allow_powerx_delegate(request: Request, cfg: RBACConfig) -> bool:
    tc = get_tenant_context(request)
    if not tc:
        return False
    raw = getattr(request.state, "raw_bearer_token", "")
    if not raw:
        return False
    claims = _decode_jwt_claims(raw)
    if claims is None:
        return False
    if cfg.powerx_issuer and claims.get("iss") != cfg.powerx_issuer:
        return False
    if cfg.powerx_audience and not _audience_matches(claims.get("aud"), cfg.powerx_audience):
        return False
    return True


def _decode_jwt_claims(raw: str) -> dict[str, Any] | None:
    parts = raw.split(".")
    if len(parts) < 2:
        return None
    payload_b64 = parts[1]
    try:
        padding = "=" * (-len(payload_b64) % 4)
        payload = base64.urlsafe_b64decode(payload_b64 + padding)
        return json.loads(payload.decode("utf-8"))
    except (ValueError, json.JSONDecodeError, UnicodeDecodeError):
        return None


def _audience_matches(aud: Any, expected: str) -> bool:
    if isinstance(aud, str):
        return aud == expected
    if isinstance(aud, list):
        return any(isinstance(item, str) and item == expected for item in aud)
    return False


def _is_super_admin(user_roles: list[str], super_roles: list[str]) -> bool:
    if not user_roles or not super_roles:
        return False
    roles = {role.strip().lower() for role in user_roles if role.strip()}
    return any(role.strip().lower() in roles for role in super_roles if role.strip())


def _has_perm(user_perms: list[str], need: Permission) -> bool:
    if not user_perms or need is None:
        return False
    res = (need.resource or "").strip().lower()
    act = (need.action or "").strip().lower()
    want = f"{res}:{act}"
    for perm in user_perms:
        p = (perm or "").strip().lower()
        if not p:
            continue
        if p in {"*", "*:*"}:
            return True
        if p == want:
            return True
        if ":" not in p:
            continue
        pr, pa = p.split(":", 1)
        if pr in {"*", res} and pa in {"*", act}:
            return True
    return False


def _match_route(method: str, path: str, table: dict[str, Permission]) -> tuple[Permission | None, bool]:
    if not table:
        return None, False
    key = f"{method}:{path}"
    if key in table:
        return table[key], True
    for route, perm in table.items():
        if ":" not in route:
            continue
        m, pat = route.split(":", 1)
        if (m == method or m == "*") and fnmatch.fnmatch(path, pat):
            return perm, True
    return None, False


def _infer_permission(method: str, path: str) -> Permission | None:
    action = _method_to_action(method)
    if not action:
        return None
    raw = path.split("?", 1)[0]
    parts: list[str] = []
    for seg in raw.strip("/").split("/"):
        seg = seg.strip().lower()
        if not seg or seg in {"api", "v1", "v2", "admin", "tenant"}:
            continue
        if seg.startswith(":"):
            continue
        parts.append(_sanitize_segment(seg))
        if len(parts) >= 2:
            break
    if not parts:
        return None
    resource = parts[0]
    if len(parts) > 1:
        resource = f"{resource}.{parts[1]}"
    return Permission(resource=resource, action=action)


def _sanitize_segment(seg: str) -> str:
    return seg.replace("-", "_").replace(".", "_")


def _method_to_action(method: str) -> str:
    method = method.upper()
    if method in {"GET", "HEAD"}:
        return "read"
    if method in {"POST", "PUT", "PATCH", "DELETE"}:
        return "manage"
    return ""


def _normalize_permission(perm: Permission, cfg: RBACConfig) -> Permission:
    if not perm:
        return perm
    resource = (perm.resource or "").strip()
    action = (perm.action or "").strip()
    plugin_id = (cfg.plugin_id or "").strip()
    if plugin_id and resource and ":" not in resource:
        resource = f"{plugin_id}:{resource}"
    return Permission(resource=resource, action=action)


def _route_template(request: Request) -> str:
    route = request.scope.get("route")
    path = getattr(route, "path", None)
    if not path:
        return ""
    return _to_gin_path_template(str(path))


def _to_gin_path_template(raw: str) -> str:
    if "{" not in raw:
        return raw
    out = ""
    buf = ""
    in_brace = False
    for ch in raw:
        if ch == "{":
            in_brace = True
            buf = ""
            continue
        if ch == "}" and in_brace:
            in_brace = False
            out += f":{buf}"
            continue
        if in_brace:
            buf += ch
        else:
            out += ch
    return out


def _strip_plugin_prefix(path: str) -> str:
    if not path.startswith("/_p/"):
        return path
    parts = path.split("/", 3)
    if len(parts) >= 4:
        return "/" + parts[3]
    return path


def _is_health_endpoint(path: str, api_prefix: str) -> bool:
    lowered = path.lower().strip()
    if lowered.startswith("/healthz") or lowered.startswith(f"{api_prefix}/healthz"):
        return True
    if lowered.startswith(f"{api_prefix}/admin/runtime/metrics"):
        return True
    return False
