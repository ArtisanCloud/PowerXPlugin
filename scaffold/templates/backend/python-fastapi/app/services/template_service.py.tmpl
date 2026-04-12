from __future__ import annotations

from datetime import datetime
from typing import Any
from uuid import UUID

from sqlalchemy import or_, select

from app.entity.models import Template
from app.entity.repository.db import get_db

_DEFAULT_TENANT_UUID = "00000000-0000-0000-0000-000000000001"


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


def _normalize_uuid(value: Any) -> str:
    raw = str(value or "").strip()
    if not raw:
        return ""
    try:
        return str(UUID(raw)).lower()
    except ValueError:
        return ""


def _resolve_tenant_uuid(payload: dict | None = None) -> str:
    source = payload or {}
    candidate = source.get("tenant_uuid") or source.get("tenantUuid")
    normalized = _normalize_uuid(candidate)
    if normalized:
        return normalized
    return _DEFAULT_TENANT_UUID


class TemplateService:
    def list_templates(self, params: dict):
        db = get_db().session()
        try:
            query = select(Template)
            keyword = params.get("q") or params.get("query")
            tenant_uuid = _normalize_uuid(params.get("tenant_uuid") or params.get("tenantUuid"))
            if tenant_uuid:
                query = query.where(Template.tenant_uuid == tenant_uuid)
            if keyword:
                query = query.where(
                    or_(
                        Template.name.ilike(f"%{keyword}%"),
                        Template.description.ilike(f"%{keyword}%"),
                    )
                )
            page = _default_page(params.get("page"))
            page_size = _default_page_size(params.get("page_size") or params.get("pageSize"))
            total = db.execute(query).scalars().all()
            items = (
                db.execute(query.offset((page - 1) * page_size).limit(page_size))
                .scalars()
                .all()
            )
            payload = {
                "list": [_to_dict(item) for item in items],
                "page_index": page,
                "page_size": page_size,
                "total": len(total),
            }
            payload["items"] = payload["list"]
            payload["page"] = page
            payload["limit"] = page_size
            return payload
        finally:
            db.close()

    def get_template(self, template_id: str):
        db = get_db().session()
        try:
            template = db.execute(
                select(Template).where(Template.id == _parse_int(template_id))
            ).scalar_one_or_none()
            return _to_dict(template) if template else {}
        finally:
            db.close()

    def create_template(self, payload: dict):
        db = get_db().session()
        try:
            template = Template(
                tenant_uuid=_resolve_tenant_uuid(payload),
                name=payload.get("name") or "",
                description=payload.get("description"),
                content=payload.get("content") or "",
                status=payload.get("status") or "draft",
                review_status=payload.get("review_status") or "pending",
                review_comment=payload.get("review_comment"),
                reviewed_by=payload.get("reviewed_by"),
                reviewed_at=payload.get("reviewed_at"),
                publish_channel=payload.get("publish_channel"),
                published_at=payload.get("published_at"),
                cleanup_reason=payload.get("cleanup_reason"),
                cleaned_at=payload.get("cleaned_at"),
                created_at=_now(),
                updated_at=_now(),
            )
            db.add(template)
            db.commit()
            db.refresh(template)
            return _to_dict(template)
        finally:
            db.close()

    def update_template(self, template_id: str, payload: dict):
        db = get_db().session()
        try:
            template = db.execute(
                select(Template).where(Template.id == _parse_int(template_id))
            ).scalar_one_or_none()
            if not template:
                return {}
            if payload.get("name") is not None:
                template.name = payload["name"]
            if payload.get("description") is not None:
                template.description = payload["description"]
            if payload.get("content") is not None:
                template.content = payload["content"]
            if payload.get("status") is not None:
                template.status = payload["status"]
            if payload.get("review_status") is not None:
                template.review_status = payload["review_status"]
            if payload.get("review_comment") is not None:
                template.review_comment = payload["review_comment"]
            if payload.get("reviewed_by") is not None:
                template.reviewed_by = payload["reviewed_by"]
            if payload.get("reviewed_at") is not None:
                template.reviewed_at = payload["reviewed_at"]
            if payload.get("publish_channel") is not None:
                template.publish_channel = payload["publish_channel"]
            if payload.get("published_at") is not None:
                template.published_at = payload["published_at"]
            if payload.get("cleanup_reason") is not None:
                template.cleanup_reason = payload["cleanup_reason"]
            if payload.get("cleaned_at") is not None:
                template.cleaned_at = payload["cleaned_at"]
            template.updated_at = _now()
            db.commit()
            return _to_dict(template)
        finally:
            db.close()

    def delete_template(self, template_id: str):
        db = get_db().session()
        try:
            template = db.execute(
                select(Template).where(Template.id == _parse_int(template_id))
            ).scalar_one_or_none()
            if not template:
                return {"ok": False}
            db.delete(template)
            db.commit()
            return {"ok": True}
        finally:
            db.close()
