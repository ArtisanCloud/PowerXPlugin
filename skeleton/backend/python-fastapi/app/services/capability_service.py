from __future__ import annotations

from datetime import datetime
from typing import Any
from uuid import uuid4

from sqlalchemy import select

from app.entity.models import Capability
from app.entity.repository.db import get_db

_CAPABILITY_REGISTRY: list[dict[str, Any]] = []
_CAPABILITY_LIFECYCLES: dict[str, dict[str, Any]] = {}
_CAPABILITY_EXPOSURES: dict[str, dict[str, Any]] = {}
_CAPABILITY_QUOTAS: dict[str, list[dict[str, Any]]] = {}


def _now_iso() -> str:
    return datetime.utcnow().isoformat() + "Z"


def _to_dict(obj: Any) -> dict:
    data = {}
    for key in obj.__table__.columns.keys():
        data[key] = getattr(obj, key)
    return data


class CapabilityService:
    def list_capabilities(self):
        db = get_db().session()
        try:
            items = db.execute(select(Capability)).scalars().all()
            return [
                {
                    "id": item.id,
                    "version": item.version or "v1",
                    "descriptor": item.name,
                    "module": None,
                    "kind": None,
                    "tags": [],
                    "checksum": "",
                    "execution": {"mode": "sync"},
                    "protocols": {},
                }
                for item in items
            ]
        finally:
            db.close()

    def register_template(self):
        return {
            "namespace": "com.powerx.plugins",
            "sensitivity_options": ["public", "internal", "restricted"],
            "async_modes": ["sync", "async"],
            "tag_suggestions": [],
            "field_hints": {},
            "schema_placeholders": {"input": "{}", "output": "{}"},
            "protocol_samples": {},
            "identifier_example": "com.powerx.plugins.demo.action",
        }

    def register(self, payload: dict):
        capability_id = uuid4().hex
        name = payload.get("name") or {}
        display_name = name.get("zh") or name.get("en") or payload.get("resource") or "capability"
        record = {
            **payload,
            "capability_id": capability_id,
            "status": "draft" if payload.get("draft", False) else "submitted",
            "created_at": _now_iso(),
            "updated_at": _now_iso(),
        }
        _CAPABILITY_REGISTRY.append(record)
        db = get_db().session()
        try:
            cap = Capability(
                id=capability_id,
                name=display_name,
                status=record["status"],
                version=payload.get("metadata", {}).get("version") if payload.get("metadata") else None,
                created_at=datetime.utcnow(),
                updated_at=datetime.utcnow(),
            )
            db.add(cap)
            db.commit()
        finally:
            db.close()
        return record

    def validate(self, payload: dict):
        return {"capability_id": payload.get("capability_id") or uuid4().hex, "valid": True, "errors": []}

    def lifecycle_template(self):
        return {
            "statuses": ["draft", "reviewing", "approved", "rejected", "published"],
            "channels": ["public", "private"],
        }

    def list_lifecycle(self):
        return list(_CAPABILITY_LIFECYCLES.values())

    def create_lifecycle(self, payload: dict):
        plan_id = payload.get("plan_id") or uuid4().hex
        record = {
            **payload,
            "plan_id": plan_id,
            "status": payload.get("status") or "draft",
            "created_at": _now_iso(),
            "updated_at": _now_iso(),
        }
        _CAPABILITY_LIFECYCLES[plan_id] = record
        return record

    def update_lifecycle_status(self, plan_id: str, payload: dict):
        record = _CAPABILITY_LIFECYCLES.get(plan_id, {"plan_id": plan_id})
        if payload.get("status"):
            record["status"] = payload["status"]
        record["updated_at"] = _now_iso()
        _CAPABILITY_LIFECYCLES[plan_id] = record
        return record

    def exposure_template(self):
        return {
            "channels": ["public", "private"],
            "regions": ["global"],
        }

    def exposure_detail(self, capability_id: str):
        return _CAPABILITY_EXPOSURES.get(capability_id, {"capability_id": capability_id})

    def update_exposure(self, capability_id: str, payload: dict):
        record = {**payload, "capability_id": capability_id, "updated_at": _now_iso()}
        _CAPABILITY_EXPOSURES[capability_id] = record
        return record

    def list_quotas(self, capability_id: str):
        return _CAPABILITY_QUOTAS.get(capability_id, [])

    def update_quotas(self, capability_id: str, payload: dict):
        quotas = payload.get("quotas") or []
        _CAPABILITY_QUOTAS[capability_id] = quotas
        return {"capability_id": capability_id, "quotas": quotas}
