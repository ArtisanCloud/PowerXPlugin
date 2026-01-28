from __future__ import annotations

from datetime import datetime
from typing import Any
from uuid import uuid4

from sqlalchemy import or_, select

from app.entity.models import Department, Member, Permission, Role, Tenant, User
from app.entity.repository.db import get_db


def _now() -> datetime:
    return datetime.utcnow()


def _to_dict(obj: Any) -> dict:
    data = {}
    for key in obj.__table__.columns.keys():
        data[key] = getattr(obj, key)
    return data


def _default_page(value: Any) -> int:
    try:
        page = int(value)
    except (TypeError, ValueError):
        return 1
    return page if page > 0 else 1


def _default_page_size(value: Any) -> int:
    try:
        size = int(value)
    except (TypeError, ValueError):
        return 20
    if size <= 0:
        return 20
    return min(size, 100)


def _parse_int(value: Any) -> int | None:
    try:
        return int(value)
    except (TypeError, ValueError):
        return None


class IAMService:
    def list_tenants(self, params: dict):
        db = get_db().session()
        try:
            query = select(Tenant)
            status = params.get("status")
            keyword = params.get("q") or params.get("query")
            if status:
                query = query.where(Tenant.status == status)
            if keyword:
                query = query.where(
                    or_(Tenant.key.ilike(f"%{keyword}%"), Tenant.name.ilike(f"%{keyword}%"))
                )
            page = _default_page(params.get("page"))
            page_size = _default_page_size(params.get("page_size") or params.get("pageSize"))
            total = db.execute(query).scalars().all()
            items = (
                db.execute(query.offset((page - 1) * page_size).limit(page_size))
                .scalars()
                .all()
            )
            return {
                "items": [_to_dict(item) for item in items],
                "total": len(total),
                "page": page,
                "page_size": page_size,
            }
        finally:
            db.close()

    def get_tenant(self, tenant_id: str):
        db = get_db().session()
        try:
            tenant = db.execute(
                select(Tenant).where(
                    or_(
                        Tenant.id == _parse_int(tenant_id),
                        Tenant.uuid == tenant_id,
                        Tenant.key == tenant_id,
                    )
                )
            ).scalar_one_or_none()
            return _to_dict(tenant) if tenant else {}
        finally:
            db.close()

    def create_tenant(self, payload: dict):
        db = get_db().session()
        try:
            uuid_value = payload.get("uuid") or payload.get("tenant_uuid") or uuid4().hex
            tenant = Tenant(
                uuid=uuid_value,
                key=payload.get("key") or payload.get("name") or uuid_value,
                name=payload.get("name") or payload.get("key") or "tenant",
                status=payload.get("status") or "active",
                plan=payload.get("plan") or "free",
                created_at=_now(),
                updated_at=_now(),
            )
            db.add(tenant)
            db.commit()
            return _to_dict(tenant)
        finally:
            db.close()

    def update_tenant(self, tenant_id: str, payload: dict):
        db = get_db().session()
        try:
            tenant = db.execute(
                select(Tenant).where(Tenant.id == _parse_int(tenant_id))
            ).scalar_one_or_none()
            if not tenant:
                return {}
            if payload.get("name") is not None:
                tenant.name = payload["name"]
            if payload.get("status") is not None:
                tenant.status = payload["status"]
            if payload.get("plan") is not None:
                tenant.plan = payload["plan"]
            tenant.updated_at = _now()
            db.commit()
            return _to_dict(tenant)
        finally:
            db.close()

    def list_roles(self, params: dict):
        db = get_db().session()
        try:
            query = select(Role)
            tenant_uuid = params.get("tenant_uuid") or params.get("tenantUuid")
            keyword = params.get("q") or params.get("query")
            scope_type = params.get("scope_type") or params.get("scopeType")
            if tenant_uuid:
                query = query.where(Role.tenant_uuid == tenant_uuid)
            if keyword:
                query = query.where(or_(Role.name.ilike(f"%{keyword}%"), Role.code.ilike(f"%{keyword}%")))
            if scope_type:
                query = query.where(Role.scope_type == scope_type)
            items = db.execute(query).scalars().all()
            return {"items": [_to_dict(item) for item in items], "total": len(items)}
        finally:
            db.close()

    def get_role(self, role_id: str):
        db = get_db().session()
        try:
            role = db.execute(select(Role).where(Role.id == _parse_int(role_id))).scalar_one_or_none()
            return _to_dict(role) if role else {}
        finally:
            db.close()

    def create_role(self, payload: dict):
        db = get_db().session()
        try:
            role = Role(
                tenant_uuid=payload.get("tenant_uuid") or payload.get("tenantUuid") or "",
                code=payload.get("code") or payload.get("name") or uuid4().hex,
                name=payload.get("name") or payload.get("code") or "role",
                description=payload.get("description"),
                scope_type=payload.get("scope_type") or payload.get("scopeType") or "tenant",
                policy_version=payload.get("policy_version") or "v1",
                created_at=_now(),
                updated_at=_now(),
            )
            db.add(role)
            db.commit()
            return _to_dict(role)
        finally:
            db.close()

    def update_role(self, role_id: str, payload: dict):
        db = get_db().session()
        try:
            role = db.execute(select(Role).where(Role.id == _parse_int(role_id))).scalar_one_or_none()
            if not role:
                return {}
            if payload.get("name") is not None:
                role.name = payload["name"]
            if payload.get("description") is not None:
                role.description = payload["description"]
            if payload.get("scope_type") is not None:
                role.scope_type = payload["scope_type"]
            if payload.get("scopeType") is not None:
                role.scope_type = payload["scopeType"]
            if payload.get("policy_version") is not None:
                role.policy_version = payload["policy_version"]
            role.updated_at = _now()
            db.commit()
            return _to_dict(role)
        finally:
            db.close()

    def delete_role(self, role_id: str):
        db = get_db().session()
        try:
            role = db.execute(select(Role).where(Role.id == _parse_int(role_id))).scalar_one_or_none()
            if not role:
                return {"ok": False}
            db.delete(role)
            db.commit()
            return {"ok": True}
        finally:
            db.close()

    def list_permissions(self):
        db = get_db().session()
        try:
            items = db.execute(select(Permission)).scalars().all()
            return {"items": [_to_dict(item) for item in items]}
        finally:
            db.close()

    def create_permission(self, payload: dict):
        db = get_db().session()
        try:
            permission = Permission(
                resource=payload.get("resource") or "",
                action=payload.get("action") or "",
                description=payload.get("description"),
                created_at=_now(),
                updated_at=_now(),
            )
            db.add(permission)
            db.commit()
            return _to_dict(permission)
        finally:
            db.close()

    def update_permission(self, permission_id: str, payload: dict):
        db = get_db().session()
        try:
            permission = db.execute(
                select(Permission).where(Permission.id == _parse_int(permission_id))
            ).scalar_one_or_none()
            if not permission:
                return {}
            if payload.get("resource") is not None:
                permission.resource = payload["resource"]
            if payload.get("action") is not None:
                permission.action = payload["action"]
            if payload.get("description") is not None:
                permission.description = payload["description"]
            permission.updated_at = _now()
            db.commit()
            return _to_dict(permission)
        finally:
            db.close()

    def delete_permission(self, permission_id: str):
        db = get_db().session()
        try:
            permission = db.execute(
                select(Permission).where(Permission.id == _parse_int(permission_id))
            ).scalar_one_or_none()
            if not permission:
                return {"ok": False}
            db.delete(permission)
            db.commit()
            return {"ok": True}
        finally:
            db.close()

    def list_departments(self, params: dict):
        db = get_db().session()
        try:
            query = select(Department)
            tenant_uuid = params.get("tenant_uuid") or params.get("tenantUuid")
            if tenant_uuid:
                query = query.where(Department.tenant_uuid == tenant_uuid)
            items = db.execute(query).scalars().all()
            return {"items": [_to_dict(item) for item in items]}
        finally:
            db.close()

    def create_department(self, payload: dict):
        db = get_db().session()
        try:
            department = Department(
                tenant_uuid=payload.get("tenant_uuid") or payload.get("tenantUuid") or "",
                name=payload.get("name") or "",
                code=payload.get("code") or "",
                parent_id=payload.get("parent_id") or payload.get("parentId"),
                description=payload.get("description"),
                sort_order=payload.get("sort_order") or payload.get("sortOrder") or 0,
                path=payload.get("path") or "",
                created_at=_now(),
                updated_at=_now(),
            )
            db.add(department)
            db.commit()
            return _to_dict(department)
        finally:
            db.close()

    def update_department(self, department_id: str, payload: dict):
        db = get_db().session()
        try:
            department = db.execute(
                select(Department).where(Department.id == _parse_int(department_id))
            ).scalar_one_or_none()
            if not department:
                return {}
            if payload.get("name") is not None:
                department.name = payload["name"]
            if payload.get("description") is not None:
                department.description = payload["description"]
            if payload.get("parent_id") is not None:
                department.parent_id = payload["parent_id"]
            if payload.get("parentId") is not None:
                department.parent_id = payload["parentId"]
            if payload.get("sort_order") is not None:
                department.sort_order = payload["sort_order"]
            if payload.get("sortOrder") is not None:
                department.sort_order = payload["sortOrder"]
            if payload.get("path") is not None:
                department.path = payload["path"]
            department.updated_at = _now()
            db.commit()
            return _to_dict(department)
        finally:
            db.close()

    def delete_department(self, department_id: str):
        db = get_db().session()
        try:
            department = db.execute(
                select(Department).where(Department.id == _parse_int(department_id))
            ).scalar_one_or_none()
            if not department:
                return {"ok": False}
            db.delete(department)
            db.commit()
            return {"ok": True}
        finally:
            db.close()

    def list_members(self, params: dict):
        db = get_db().session()
        try:
            query = select(Member, User).join(User, User.id == Member.user_id)
            tenant_uuid = params.get("tenant_uuid") or params.get("tenantUuid")
            status = params.get("status")
            keyword = params.get("q") or params.get("query")
            if tenant_uuid:
                query = query.where(Member.tenant_uuid == tenant_uuid)
            if status:
                query = query.where(Member.status == status)
            if keyword:
                query = query.where(
                    or_(
                        Member.username.ilike(f"%{keyword}%"),
                        User.email.ilike(f"%{keyword}%"),
                        User.phone.ilike(f"%{keyword}%"),
                    )
                )
            items = db.execute(query).all()
            result = []
            for member, user in items:
                result.append(
                    {
                        "member_id": member.id,
                        "tenant_uuid": member.tenant_uuid,
                        "user_id": member.user_id,
                        "email": user.email,
                        "phone": user.phone,
                        "display_name": member.display_name or user.display_name,
                        "username": member.username,
                        "status": member.status,
                        "department_id": member.department_id,
                        "created_at": member.created_at,
                        "last_login_at": member.last_login_at,
                        "roles": [],
                    }
                )
            return {"items": result}
        finally:
            db.close()

    def create_member(self, payload: dict):
        db = get_db().session()
        try:
            email = payload.get("email")
            phone = payload.get("phone")
            user = None
            if email or phone:
                user = (
                    db.execute(
                        select(User).where(or_(User.email == email, User.phone == phone))
                    )
                    .scalars()
                    .first()
                )
            if not user:
                user = User(
                    email=email,
                    phone=phone,
                    display_name=payload.get("display_name") or payload.get("displayName"),
                    avatar_url=payload.get("avatar_url") or payload.get("avatarUrl"),
                    status=payload.get("status") or "active",
                    password_hash=payload.get("password") or "",
                    created_at=_now(),
                    updated_at=_now(),
                )
                db.add(user)
                db.flush()
            member = Member(
                tenant_uuid=payload.get("tenant_uuid") or payload.get("tenantUuid") or "",
                user_id=user.id,
                username=payload.get("username") or (email or phone or ""),
                display_name=payload.get("display_name") or payload.get("displayName"),
                avatar_url=payload.get("avatar_url") or payload.get("avatarUrl"),
                status=payload.get("status") or "active",
                department_id=payload.get("department_id") or payload.get("departmentId"),
                meta=payload.get("meta"),
                created_at=_now(),
                updated_at=_now(),
            )
            db.add(member)
            db.commit()
            return {
                "member_id": member.id,
                "tenant_uuid": member.tenant_uuid,
                "user_id": member.user_id,
                "email": user.email,
                "phone": user.phone,
                "display_name": member.display_name or user.display_name,
                "username": member.username,
                "status": member.status,
                "department_id": member.department_id,
                "created_at": member.created_at,
                "last_login_at": member.last_login_at,
                "roles": [],
            }
        finally:
            db.close()

    def update_member(self, member_id: str, payload: dict):
        db = get_db().session()
        try:
            member = db.execute(select(Member).where(Member.id == _parse_int(member_id))).scalar_one_or_none()
            if not member:
                return {}
            if payload.get("display_name") is not None:
                member.display_name = payload["display_name"]
            if payload.get("displayName") is not None:
                member.display_name = payload["displayName"]
            if payload.get("status") is not None:
                member.status = payload["status"]
            if payload.get("department_id") is not None:
                member.department_id = payload["department_id"]
            if payload.get("departmentId") is not None:
                member.department_id = payload["departmentId"]
            member.updated_at = _now()
            db.commit()
            user = db.execute(select(User).where(User.id == member.user_id)).scalar_one_or_none()
            return {
                "member_id": member.id,
                "tenant_uuid": member.tenant_uuid,
                "user_id": member.user_id,
                "email": user.email if user else None,
                "phone": user.phone if user else None,
                "display_name": member.display_name or (user.display_name if user else None),
                "username": member.username,
                "status": member.status,
                "department_id": member.department_id,
                "created_at": member.created_at,
                "last_login_at": member.last_login_at,
                "roles": [],
            }
        finally:
            db.close()
