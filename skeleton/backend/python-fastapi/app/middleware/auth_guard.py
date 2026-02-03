from __future__ import annotations

import base64
import hashlib
import hmac
import json
import os
import time
from dataclasses import dataclass
from typing import Any, Callable, Iterable

from fastapi import Request
from fastapi.responses import JSONResponse

from app.config.settings import Settings
from app.middleware.tenant_context import TenantContext, set_tenant_context

_CTX_HEADER = "x-powerx-ctx"
_CTX_SIG_HEADER = "x-powerx-ctx-sig"


@dataclass(frozen=True)
class JWTAuthConfig:
    issuer: str
    accept_audiences: list[str]
    hmac_secret: str
    clock_skew_seconds: int
    optional: bool
    allow_signed_context: bool
    context_hmac_secret: str
    max_ctx_age_seconds: int


def build_jwt_config(settings: Settings) -> JWTAuthConfig:
    in_proxy = os.getenv("POWERX_PROXY") == "1"
    if in_proxy:
        plugin_id = os.getenv("POWERX_PLUGIN_ID", "").strip()
        audience = os.getenv("POWERX_SECURITY_JWT_AUDIENCE", "").strip()
        if not audience and plugin_id:
            audience = f"plugin:{plugin_id}"
        return JWTAuthConfig(
            issuer=os.getenv("POWERX_SECURITY_JWT_ISSUER", "").strip(),
            accept_audiences=_split_audiences(audience),
            hmac_secret=os.getenv("POWERX_SECURITY_JWT_SECRET", "").strip(),
            clock_skew_seconds=60,
            optional=False,
            allow_signed_context=True,
            context_hmac_secret=os.getenv("POWERX_SECURITY_CTX_HMAC_SECRET", "").strip(),
            max_ctx_age_seconds=300,
        )

    optional = False
    env_optional = os.getenv("POWERX_AUTH_OPTIONAL", "").strip()
    if env_optional:
        optional = env_optional in {"1", "true", "TRUE", "yes", "on"}
    else:
        is_production = not settings.dev_mode and not settings.server_dev_mode
        optional = not is_production

    issuer = settings.context_issuer.strip() if settings.context_issuer else "powerx-local"
    audiences = _split_audiences(settings.context_audience.strip()) if settings.context_audience else ["powerx:plugin"]
    hmac_secret = settings.context_hmac_secret.strip() if settings.context_hmac_secret else "powerx-plugin-dev"
    return JWTAuthConfig(
        issuer=issuer or "powerx-local",
        accept_audiences=audiences,
        hmac_secret=hmac_secret,
        clock_skew_seconds=60,
        optional=optional,
        allow_signed_context=False,
        context_hmac_secret=hmac_secret,
        max_ctx_age_seconds=300,
    )


async def auth_guard_middleware(request: Request, call_next: Callable):
    cfg: JWTAuthConfig = getattr(request.app.state, "jwt_cfg", None)
    if cfg is None:
        return await call_next(request)
    path = _strip_plugin_prefix(request.url.path)
    api_prefix = request.app.state.settings.api_prefix
    if _is_public_auth(path, api_prefix) or _is_health_endpoint(path, api_prefix):
        return await call_next(request)

    tc, raw_bearer, ok = _parse_from_headers(request, cfg)
    if ok:
        set_tenant_context(request, tc)
        request.state.raw_bearer_token = raw_bearer
        return await call_next(request)

    raw_auth = request.headers.get("authorization", "")
    if raw_auth.lower().startswith("bearer "):
        token = raw_auth[7:].strip()
        if token and cfg.hmac_secret:
            tc = _parse_hs256(token, cfg)
            if tc is not None:
                set_tenant_context(request, tc)
                request.state.raw_bearer_token = token
                return await call_next(request)
        return JSONResponse(status_code=401, content={"error": "jwt Unauthorized"})

    if cfg.optional:
        return await call_next(request)

    return JSONResponse(status_code=401, content={"error": "jwt Unauthorized"})


def _parse_from_headers(request: Request, cfg: JWTAuthConfig) -> tuple[TenantContext, str, bool]:
    authz = request.headers.get("authorization", "")
    if authz.lower().startswith("bearer "):
        raw = authz[7:].strip()
        if raw and cfg.hmac_secret:
            tc = _parse_hs256(raw, cfg)
            if tc is not None:
                return tc, raw, True
    if cfg.allow_signed_context and cfg.context_hmac_secret:
        tc = _try_load_signed_context(request, cfg.context_hmac_secret, cfg.max_ctx_age_seconds)
        if tc is not None:
            return tc, "", True
    return TenantContext(), "", False


def _try_load_signed_context(
    request: Request, secret: str, max_age_seconds: int
) -> TenantContext | None:
    ctx_b64 = request.headers.get(_CTX_HEADER, "")
    sig_hex = request.headers.get(_CTX_SIG_HEADER, "")
    if not ctx_b64 or not sig_hex:
        return None
    try:
        raw = base64.b64decode(ctx_b64)
    except (ValueError, TypeError):
        return None
    mac = hmac.new(secret.encode("utf-8"), ctx_b64.encode("utf-8"), hashlib.sha256)
    if not hmac.compare_digest(mac.hexdigest(), sig_hex):
        return None
    try:
        payload = json.loads(raw.decode("utf-8"))
    except (json.JSONDecodeError, UnicodeDecodeError):
        return None
    ts = _to_int(payload.get("ts"))
    if max_age_seconds > 0 and ts > 0 and (time.time() - ts) > max_age_seconds:
        return None
    return _claims_to_context(payload)


def _parse_hs256(raw_token: str, cfg: JWTAuthConfig) -> TenantContext | None:
    claims = _decode_and_verify_jwt(raw_token, cfg)
    if claims is None:
        return None
    return _claims_to_context(claims)


def _verify_hs256(raw_token: str, cfg: JWTAuthConfig, strict: bool = True) -> bool:
    return _decode_and_verify_jwt(raw_token, cfg, strict=strict) is not None


def _decode_and_verify_jwt(
    raw_token: str, cfg: JWTAuthConfig, strict: bool = True
) -> dict[str, Any] | None:
    try:
        header_b64, payload_b64, sig_b64 = raw_token.split(".")
    except ValueError:
        return None
    header = _b64url_json(header_b64)
    if not isinstance(header, dict):
        return None
    if header.get("alg") != "HS256":
        return None
    signing_input = f"{header_b64}.{payload_b64}".encode("utf-8")
    mac = hmac.new(cfg.hmac_secret.encode("utf-8"), signing_input, hashlib.sha256)
    expected_sig = base64.urlsafe_b64encode(mac.digest()).rstrip(b"=").decode("utf-8")
    if not hmac.compare_digest(expected_sig, sig_b64):
        return None
    claims = _b64url_json(payload_b64)
    if not isinstance(claims, dict):
        return None
    if strict and not _validate_claims(claims, cfg):
        return None
    return claims


def _validate_claims(claims: dict[str, Any], cfg: JWTAuthConfig) -> bool:
    if cfg.issuer and claims.get("iss") and claims.get("iss") != cfg.issuer:
        return False
    if cfg.accept_audiences and claims.get("aud"):
        if not _audience_matches(claims.get("aud"), cfg.accept_audiences):
            return False
    now = time.time()
    leeway = cfg.clock_skew_seconds or 60
    exp = _to_int(claims.get("exp"))
    if exp and now > (exp + leeway):
        return False
    nbf = _to_int(claims.get("nbf"))
    if nbf and (now + leeway) < nbf:
        return False
    return True


def _claims_to_context(payload: dict[str, Any]) -> TenantContext:
    tenant = payload.get("tid") or payload.get("tenant_uuid") or ""
    if isinstance(tenant, (int, float)):
        tenant = str(int(tenant))
    user_id = _to_int(payload.get("uid")) or 0
    return TenantContext(
        tenant_uuid=str(tenant or "").strip(),
        user_id=user_id,
        roles=_to_list(payload.get("roles")),
        permissions=_to_list(payload.get("perms")),
        policy_version=str(payload.get("policy_version") or "").strip(),
        plugin_id=str(payload.get("plugin_id") or "").strip(),
    )


def _split_audiences(raw: str) -> list[str]:
    if not raw:
        return []
    parts = [item.strip() for item in raw.replace(";", ",").split(",")]
    return [item for item in parts if item]


def _audience_matches(aud: Any, allowed: Iterable[str]) -> bool:
    if isinstance(aud, str):
        return aud in allowed
    if isinstance(aud, list):
        return any(isinstance(item, str) and item in allowed for item in aud)
    return False


def _to_int(value: Any) -> int:
    try:
        return int(value)
    except (TypeError, ValueError):
        return 0


def _to_list(value: Any) -> list[str]:
    if isinstance(value, list):
        return [str(item).strip() for item in value if str(item).strip()]
    if isinstance(value, str) and value.strip():
        return [value.strip()]
    return []


def _b64url_json(segment: str) -> Any:
    try:
        raw = base64.urlsafe_b64decode(_pad_b64(segment))
        return json.loads(raw.decode("utf-8"))
    except (ValueError, json.JSONDecodeError, UnicodeDecodeError):
        return None


def _pad_b64(segment: str) -> bytes:
    padding = "=" * (-len(segment) % 4)
    return (segment + padding).encode("utf-8")


def _strip_plugin_prefix(path: str) -> str:
    if not path.startswith("/_p/"):
        return path
    parts = path.split("/", 3)
    if len(parts) >= 4:
        return "/" + parts[3]
    return path


def _is_public_auth(path: str, api_prefix: str) -> bool:
    prefix = api_prefix.rstrip("/")
    return path.startswith(f"{prefix}/admin/user/auth")


def _is_health_endpoint(path: str, api_prefix: str) -> bool:
    lowered = path.lower()
    prefix = api_prefix.rstrip("/")
    if lowered.startswith("/healthz") or lowered.startswith(f"{prefix}/healthz"):
        return True
    if lowered.startswith(f"{prefix}/admin/runtime/metrics"):
        return True
    if lowered.startswith("/assets/builds/meta") or lowered.startswith(f"{prefix}/assets/builds/meta"):
        return True
    return False
