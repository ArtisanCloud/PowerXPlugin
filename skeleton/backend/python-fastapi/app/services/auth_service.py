from __future__ import annotations

from datetime import datetime, timedelta
from typing import Any
import hashlib
import os
import secrets
from uuid import UUID, uuid4

import bcrypt
import jwt
from sqlalchemy import String, func, or_, select

from app.config.settings import get_settings
from app.entity.models import (
    IAMMemberRole,
    IAMRefreshToken,
    IAMRolePermission,
    Member,
    Permission,
    Role,
    Tenant,
    User,
)
from app.entity.repository.db import get_db


def _now() -> datetime:
    return datetime.utcnow()


def _parse_duration_seconds(value: Any, default_seconds: int) -> int:
    if value is None:
        return default_seconds
    if isinstance(value, (int, float)):
        return int(value)
    if not isinstance(value, str):
        return default_seconds
    raw = value.strip()
    if not raw:
        return default_seconds
    unit = raw[-1].lower()
    num_part = raw[:-1] if unit.isalpha() else raw
    try:
        num = float(num_part)
    except ValueError:
        return default_seconds
    if not unit.isalpha():
        return int(num)
    if unit == "s":
        return int(num)
    if unit == "m":
        return int(num * 60)
    if unit == "h":
        return int(num * 3600)
    if unit == "d":
        return int(num * 86400)
    return default_seconds


def _default_tenant_key() -> str:
    return (
        os.getenv("PLUGIN_IAM_TENANT_KEY", "").strip()
        or "00000000-0000-0000-0000-000000000001"
    )


def _resolve_plugin_id() -> str:
    return (
        os.getenv("POWERX_PLUGIN_ID", "").strip()
        or os.getenv("PLUGIN_ID", "").strip()
        or "com.powerx.plugins.base"
    )


def _resolve_policy_version() -> str:
    return os.getenv("PLUGIN_IAM_POLICY_VERSION", "").strip() or "local.v1"


def _resolve_refresh_ttl_seconds() -> int:
    raw = os.getenv("PLUGIN_IAM_REFRESH_TTL", "").strip()
    return _parse_duration_seconds(raw, 30 * 24 * 3600)


def _token_hash(token: str) -> str:
    return hashlib.sha256(token.encode("utf-8")).hexdigest()


def _is_bcrypt_hash(value: str) -> bool:
    if not value:
        return False
    return value.startswith("$2a$") or value.startswith("$2b$") or value.startswith("$2y$") or value.startswith("$2$")


def _verify_password(raw: str, stored: str) -> bool:
    if not raw or not stored:
        return False
    if _is_bcrypt_hash(stored):
        return bcrypt.checkpw(raw.encode("utf-8"), stored.encode("utf-8"))
    return secrets.compare_digest(raw, stored)


def _hash_password(raw: str) -> str:
    if not raw:
        return ""
    return bcrypt.hashpw(raw.encode("utf-8"), bcrypt.gensalt(rounds=12)).decode("utf-8")


def _to_dict(obj: Any) -> dict:
    data = {}
    for key in obj.__table__.columns.keys():
        data[key] = getattr(obj, key)
    return data


_RESET_TOKENS: dict[str, dict[str, Any]] = {}
_RESET_TTL_SECONDS = 30 * 60


class AuthService:
    def _decode_access_token(self, token: str) -> dict[str, Any]:
        raw = (token or "").strip()
        if not raw:
            return {}
        settings = get_settings()
        secret = settings.context_hmac_secret.strip() if settings.context_hmac_secret else "powerx-plugin-dev"
        audience = settings.context_audience.strip() if settings.context_audience else "powerx:plugin"
        issuer = settings.context_issuer.strip() if settings.context_issuer else "powerx-local"
        try:
            claims = jwt.decode(
                raw,
                secret,
                algorithms=["HS256"],
                audience=audience,
                issuer=issuer,
            )
        except Exception:
            return {}
        return claims if isinstance(claims, dict) else {}

    def login(self, payload: dict):
        identifier = payload.get("identifier")
        tenant = payload.get("tenant")
        if not identifier or not (payload.get("password") or ""):
            return {"token_type": "Bearer", "access_token": "", "expires_in": 0, "refresh_token": "", "scope": ""}

        db = get_db().session()
        try:
            settings = get_settings()
            plugin_id = _resolve_plugin_id()
            policy_version = _resolve_policy_version()
            tenant_key = (tenant or "").strip() or _default_tenant_key()
            tenant_key = tenant_key.strip().lower()

            tenant_rec = (
                db.execute(
                    select(Tenant).where(
                        or_(
                            func.lower(Tenant.key) == tenant_key,
                            func.lower(func.cast(Tenant.uuid, String)) == tenant_key,
                        )
                    )
                ).scalar_one_or_none()
            )
            if not tenant_rec or (tenant_rec.status or "").lower() != "active":
                return {"token_type": "Bearer", "access_token": "", "expires_in": 0, "refresh_token": "", "scope": ""}

            ident = (identifier or "").strip()
            ident_lower = ident.lower()
            member = None
            user = None
            if "@" in ident_lower:
                user = db.execute(select(User).where(func.lower(User.email) == ident_lower)).scalar_one_or_none()
                if user:
                    member = db.execute(
                        select(Member).where(
                            Member.user_id == user.id,
                            func.lower(func.cast(Member.tenant_uuid, String)) == tenant_rec.uuid.lower(),
                            func.lower(Member.status) == "active",
                        )
                    ).scalar_one_or_none()
            if member is None:
                member = db.execute(
                    select(Member).where(
                        func.lower(Member.username) == ident_lower,
                        func.lower(func.cast(Member.tenant_uuid, String)) == tenant_rec.uuid.lower(),
                        func.lower(Member.status) == "active",
                    )
                ).scalar_one_or_none()
                if member:
                    user = db.execute(select(User).where(User.id == member.user_id)).scalar_one_or_none()

            if not user or not member:
                return {"token_type": "Bearer", "access_token": "", "expires_in": 0, "refresh_token": "", "scope": ""}

            if not _verify_password(payload.get("password") or "", user.password_hash or ""):
                return {"token_type": "Bearer", "access_token": "", "expires_in": 0, "refresh_token": "", "scope": ""}

            roles, perms = self._load_role_permission_codes(db, member.id, plugin_id)
            is_root = bool(getattr(user, "is_root", False) or getattr(member, "is_admin", False))

            now = _now()
            ttl_seconds = settings.context_ttl_seconds or 900
            expires_at = now + timedelta(seconds=int(ttl_seconds))
            issuer = settings.context_issuer.strip() if settings.context_issuer else "powerx-local"
            audience = settings.context_audience.strip() if settings.context_audience else "powerx:plugin"
            secret = settings.context_hmac_secret.strip() if settings.context_hmac_secret else "powerx-plugin-dev"
            claims = {
                "tid": tenant_rec.uuid,
                "uid": str(user.id),
                "is_root": is_root,
                "roles": roles,
                "perms": perms,
                "policy_version": policy_version,
                "plugin_id": plugin_id,
                "iss": issuer,
                "aud": audience,
                "iat": int(now.timestamp()),
                "exp": int(expires_at.timestamp()),
            }
            access_token = jwt.encode(claims, secret, algorithm="HS256")
            refresh_token = secrets.token_hex(32)

            refresh_rec = IAMRefreshToken(
                token_hash=_token_hash(refresh_token),
                user_id=user.id,
                tenant_uuid=tenant_rec.uuid,
                member_id=member.id,
                expires_at=now + timedelta(seconds=_resolve_refresh_ttl_seconds()),
                revoked=False,
            )
            db.add(refresh_rec)
            db.commit()

            return {
                "token_type": "Bearer",
                "access_token": access_token,
                "refresh_token": refresh_token,
                "expires_in": int(ttl_seconds),
                "expires_at": int(expires_at.timestamp() * 1000),
                "scope": "access",
                "plugin_id": plugin_id,
                "policy_version": policy_version,
            }
        finally:
            db.close()

    def register(self, payload: dict):
        tenant_uuid = payload.get("tenant_uuid") or _default_tenant_key()
        db = get_db().session()
        try:
            tenant_key = (tenant_uuid or "").strip() or _default_tenant_key()
            tenant_key_lower = tenant_key.lower()
            tenant_uuid_value = None
            try:
                tenant_uuid_value = str(UUID(tenant_key))
            except ValueError:
                tenant_uuid_value = None

            tenant = db.execute(
                select(Tenant).where(
                    or_(
                        func.lower(Tenant.key) == tenant_key_lower,
                        func.lower(func.cast(Tenant.uuid, String)) == tenant_key_lower,
                    )
                )
            ).scalar_one_or_none()
            if not tenant:
                tenant_uuid_resolved = tenant_uuid_value or str(uuid4())
                tenant_key_resolved = tenant_key
                tenant = Tenant(
                    uuid=tenant_uuid_resolved,
                    key=tenant_key_resolved,
                    name=tenant_key_resolved,
                    status="active",
                    plan="free",
                    created_at=_now(),
                    updated_at=_now(),
                )
                db.add(tenant)

            user = User(
                email=payload.get("email"),
                phone=payload.get("phone"),
                display_name=payload.get("display_name") or payload.get("username"),
                avatar_url=payload.get("avatar_url"),
                status="active",
                password_hash=payload.get("password") or "",
                created_at=_now(),
                updated_at=_now(),
            )
            db.add(user)
            db.flush()

            member = Member(
                tenant_uuid=tenant.uuid,
                user_id=user.id,
                username=payload.get("username") or (payload.get("email") or ""),
                display_name=payload.get("display_name") or payload.get("username"),
                status="active",
                created_at=_now(),
                updated_at=_now(),
            )
            db.add(member)

            db.commit()
            return {"user": _to_dict(user), "member": _to_dict(member), "tenant": _to_dict(tenant)}
        finally:
            db.close()

    def logout(self, payload: dict):
        refresh_token = (payload or {}).get("refresh_token") or (payload or {}).get("refreshToken") or ""
        if not refresh_token:
            return {"ok": True}
        db = get_db().session()
        try:
            rec = db.execute(
                select(IAMRefreshToken).where(IAMRefreshToken.token_hash == _token_hash(refresh_token))
            ).scalar_one_or_none()
            if rec:
                rec.revoked = True
                db.add(rec)
                db.commit()
            return {"ok": True}
        finally:
            db.close()

    def refresh(self, payload: dict):
        refresh_token = (payload or {}).get("refresh_token") or (payload or {}).get("refreshToken") or ""
        if not refresh_token:
            return {
                "token_type": "Bearer",
                "access_token": "",
                "refresh_token": "",
                "expires_in": 0,
                "scope": "",
            }
        db = get_db().session()
        try:
            now = _now()
            rec = db.execute(
                select(IAMRefreshToken).where(
                    IAMRefreshToken.token_hash == _token_hash(refresh_token),
                    IAMRefreshToken.revoked.is_(False),
                    IAMRefreshToken.expires_at > now,
                )
            ).scalar_one_or_none()
            if not rec:
                return {
                    "token_type": "Bearer",
                    "access_token": "",
                    "refresh_token": "",
                    "expires_in": 0,
                    "scope": "",
                }

            user = db.execute(select(User).where(User.id == rec.user_id)).scalar_one_or_none()
            member = db.execute(select(Member).where(Member.id == rec.member_id)).scalar_one_or_none()
            tenant_uuid = str(rec.tenant_uuid).lower()
            tenant_rec = db.execute(
                select(Tenant).where(func.lower(func.cast(Tenant.uuid, String)) == tenant_uuid)
            ).scalar_one_or_none()
            if not user or not member or not tenant_rec:
                return {
                    "token_type": "Bearer",
                    "access_token": "",
                    "refresh_token": "",
                    "expires_in": 0,
                    "scope": "",
                }

            settings = get_settings()
            plugin_id = _resolve_plugin_id()
            policy_version = _resolve_policy_version()
            roles, perms = self._load_role_permission_codes(db, member.id, plugin_id)
            is_root = bool(getattr(user, "is_root", False) or getattr(member, "is_admin", False))
            ttl_seconds = settings.context_ttl_seconds or 900
            expires_at = now + timedelta(seconds=int(ttl_seconds))
            issuer = settings.context_issuer.strip() if settings.context_issuer else "powerx-local"
            audience = settings.context_audience.strip() if settings.context_audience else "powerx:plugin"
            secret = settings.context_hmac_secret.strip() if settings.context_hmac_secret else "powerx-plugin-dev"
            claims = {
                "tid": tenant_rec.uuid,
                "uid": str(user.id),
                "is_root": is_root,
                "roles": roles,
                "perms": perms,
                "policy_version": policy_version,
                "plugin_id": plugin_id,
                "iss": issuer,
                "aud": audience,
                "iat": int(now.timestamp()),
                "exp": int(expires_at.timestamp()),
            }
            access_token = jwt.encode(claims, secret, algorithm="HS256")
            new_refresh = secrets.token_hex(32)

            rec.revoked = True
            db.add(rec)
            db.add(
                IAMRefreshToken(
                    token_hash=_token_hash(new_refresh),
                    user_id=user.id,
                    tenant_uuid=tenant_rec.uuid,
                    member_id=member.id,
                    expires_at=now + timedelta(seconds=_resolve_refresh_ttl_seconds()),
                    revoked=False,
                )
            )
            db.commit()

            return {
                "token_type": "Bearer",
                "access_token": access_token,
                "refresh_token": new_refresh,
                "expires_in": int(ttl_seconds),
                "expires_at": int(expires_at.timestamp() * 1000),
                "scope": "access",
                "plugin_id": plugin_id,
                "policy_version": policy_version,
            }
        finally:
            db.close()

    def me(self, access_token: str = ""):
        db = get_db().session()
        try:
            claims = self._decode_access_token(access_token)
            user = None
            member = None
            tenant = None
            roles: list[str] = []
            permissions: list[str] = []
            policy_version = str(claims.get("policy_version") or _resolve_policy_version())
            plugin_id = str(claims.get("plugin_id") or _resolve_plugin_id())
            claim_is_root = bool(claims.get("is_root"))

            claim_uid = claims.get("uid")
            claim_tid = str(claims.get("tid") or "").strip().lower()

            if isinstance(claim_uid, int):
                user = db.execute(select(User).where(User.id == claim_uid)).scalar_one_or_none()
            elif isinstance(claim_uid, str) and claim_uid.isdigit():
                user = db.execute(select(User).where(User.id == int(claim_uid))).scalar_one_or_none()

            if not user:
                user = db.execute(select(User)).scalars().first()
            if not user:
                return {"user": {}, "member": None, "tenant": None, "roles": [], "permissions": [], "policy_version": policy_version}

            member_query = select(Member).where(Member.user_id == user.id)
            if claim_tid:
                member = db.execute(
                    member_query.where(func.lower(func.cast(Member.tenant_uuid, String)) == claim_tid)
                ).scalar_one_or_none()
            if member is None:
                member = db.execute(member_query).scalar_one_or_none()
            if member:
                tenant = db.execute(select(Tenant).where(Tenant.uuid == member.tenant_uuid)).scalar_one_or_none()
                roles, permissions = self._load_role_permission_codes(db, member.id, plugin_id)
            is_root = bool(claim_is_root or getattr(user, "is_root", False) or (member and getattr(member, "is_admin", False)))

            return {
                "user": _to_dict(user),
                "member": _to_dict(member) if member else None,
                "tenant": _to_dict(tenant) if tenant else None,
                "is_root": is_root,
                "roles": roles,
                "permissions": permissions,
                "policy_version": policy_version,
                "plugin_id": plugin_id,
            }
        finally:
            db.close()

    def profile(self, payload: dict):
        db = get_db().session()
        try:
            user = db.execute(select(User)).scalars().first()
            if not user:
                return payload
            if payload.get("display_name") is not None:
                user.display_name = payload["display_name"]
            if payload.get("avatar_url") is not None:
                user.avatar_url = payload["avatar_url"]
            user.updated_at = _now()
            db.commit()
            return _to_dict(user)
        finally:
            db.close()

    def change_password(self, payload: dict):
        return {"ok": True}

    def reset_password(self, payload: dict):
        email = (payload or {}).get("email") or ""
        email = email.strip().lower()
        if not email:
            return {"ok": True, "reset_token": "", "expires_at": 0}
        db = get_db().session()
        try:
            user = db.execute(select(User).where(func.lower(User.email) == email)).scalar_one_or_none()
            if not user:
                return {"ok": True, "reset_token": "", "expires_at": 0}
            token = secrets.token_hex(24)
            expires_at = _now() + timedelta(seconds=_RESET_TTL_SECONDS)
            _RESET_TOKENS[token] = {"user_id": user.id, "expires_at": expires_at}
            return {
                "ok": True,
                "reset_token": token,
                "expires_at": int(expires_at.timestamp() * 1000),
            }
        finally:
            db.close()

    def reset_password_confirm(self, payload: dict):
        token = (payload or {}).get("token") or ""
        new_password = (payload or {}).get("newPassword") or (payload or {}).get("new_password") or ""
        confirm_password = (payload or {}).get("confirmPassword") or (payload or {}).get("confirm_password") or ""
        token = token.strip()
        if not token or not new_password or not confirm_password:
            return {"ok": False}
        if new_password != confirm_password:
            return {"ok": False}
        record = _RESET_TOKENS.get(token)
        if not record:
            return {"ok": False}
        expires_at = record.get("expires_at")
        if not expires_at or expires_at < _now():
            _RESET_TOKENS.pop(token, None)
            return {"ok": False}
        user_id = record.get("user_id")
        if not user_id:
            return {"ok": False}
        db = get_db().session()
        try:
            user = db.execute(select(User).where(User.id == user_id)).scalar_one_or_none()
            if not user:
                return {"ok": False}
            user.password_hash = _hash_password(new_password)
            user.updated_at = _now()
            db.commit()
            _RESET_TOKENS.pop(token, None)
            return {"ok": True}
        finally:
            db.close()

    def validate(self):
        return {"valid": True}

    def permissions(self):
        return []

    def _load_role_permission_codes(self, db, member_id: int, plugin_id: str) -> tuple[list[str], list[str]]:
        role_rows = db.execute(
            select(Role.code)
            .select_from(Role)
            .join(IAMMemberRole, IAMMemberRole.role_id == Role.id)
            .where(IAMMemberRole.member_id == member_id)
        ).all()
        roles = [row[0] for row in role_rows if row and row[0]]
        perm_rows = db.execute(
            select(Permission.resource, Permission.action)
            .select_from(Permission)
            .join(IAMRolePermission, IAMRolePermission.permission_id == Permission.id)
            .join(IAMMemberRole, IAMMemberRole.role_id == IAMRolePermission.role_id)
            .where(IAMMemberRole.member_id == member_id)
        ).all()
        perms: list[str] = []
        for resource, action in perm_rows:
            code = self._format_permission_code(resource, action, plugin_id)
            if code:
                perms.append(code)
        return roles, perms

    def _format_permission_code(self, resource: str | None, action: str | None, plugin_id: str) -> str:
        res = (resource or "").strip()
        act = (action or "").strip()
        if not res or not act:
            return ""
        if res != "*" and plugin_id and ":" not in res:
            res = f"{plugin_id}:{res}"
        return f"{res}:{act}"
