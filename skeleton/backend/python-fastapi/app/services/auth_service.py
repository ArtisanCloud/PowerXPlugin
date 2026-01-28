from __future__ import annotations

from dataclasses import asdict
from datetime import datetime
from typing import Any, Optional
from uuid import uuid4

from sqlalchemy import or_, select

from app.entity.models import Member, Tenant, User
from app.entity.repository.db import get_db


def _now() -> datetime:
    return datetime.utcnow()


def _to_dict(obj: Any) -> dict:
    data = {}
    for key in obj.__table__.columns.keys():
        data[key] = getattr(obj, key)
    return data


class AuthService:
    def login(self, payload: dict):
        identifier = payload.get("identifier")
        tenant = payload.get("tenant")
        if not identifier:
            return {"token_type": "Bearer", "access_token": "", "expires_in": 0, "refresh_token": "", "scope": ""}

        db = get_db().session()
        try:
            user = db.execute(
                select(User).where(
                    or_(User.email == identifier, User.phone == identifier)
                )
            ).scalar_one_or_none()
            member = None
            if not user:
                member = db.execute(
                    select(Member).where(Member.username == identifier)
                ).scalar_one_or_none()
                if member:
                    user = db.execute(select(User).where(User.id == member.user_id)).scalar_one_or_none()
            if not user:
                return {"token_type": "Bearer", "access_token": "", "expires_in": 0, "refresh_token": "", "scope": ""}

            token = uuid4().hex
            return {
                "token_type": "Bearer",
                "access_token": token,
                "refresh_token": uuid4().hex,
                "expires_in": 3600,
                "scope": tenant or "default",
                "plugin_id": None,
                "policy_version": None,
            }
        finally:
            db.close()

    def register(self, payload: dict):
        tenant_uuid = payload.get("tenant_uuid") or "tenant-demo"
        db = get_db().session()
        try:
            tenant = db.execute(
                select(Tenant).where(
                    or_(Tenant.uuid == tenant_uuid, Tenant.key == tenant_uuid)
                )
            ).scalar_one_or_none()
            if not tenant:
                tenant = Tenant(
                    uuid=tenant_uuid,
                    key=tenant_uuid,
                    name=tenant_uuid,
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
                tenant_uuid=tenant_uuid,
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
        return {"ok": True}

    def refresh(self, payload: dict):
        return {
            "token_type": "Bearer",
            "access_token": uuid4().hex,
            "refresh_token": uuid4().hex,
            "expires_in": 3600,
            "scope": "default",
        }

    def me(self):
        db = get_db().session()
        try:
            user = db.execute(select(User)).scalars().first()
            if not user:
                return {"user": {}, "member": None, "tenant": None}
            member = db.execute(select(Member).where(Member.user_id == user.id)).scalar_one_or_none()
            tenant = None
            if member:
                tenant = db.execute(select(Tenant).where(Tenant.uuid == member.tenant_uuid)).scalar_one_or_none()
            return {
                "user": _to_dict(user),
                "member": _to_dict(member) if member else None,
                "tenant": _to_dict(tenant) if tenant else None,
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
        return {"ok": True}

    def reset_password_confirm(self, payload: dict):
        return {"ok": True}

    def validate(self):
        return {"valid": True}

    def permissions(self):
        return []
