from __future__ import annotations

from datetime import datetime
from typing import Any
from uuid import uuid4

from sqlalchemy import select

from app.entity.models import RuntimeSession
from app.entity.repository.db import get_db

_MCP_SESSIONS: dict[str, dict[str, Any]] = {}


def _now_iso() -> str:
    return datetime.utcnow().isoformat() + "Z"


class RuntimeSessionService:
    def register(self, payload: dict):
        session_id = payload.get("session_id") or uuid4().hex
        record = {
            "id": session_id,
            "runtime_assignment_id": payload.get("runtime_assignment_id") or "",
            "tenant_uuid": payload.get("tenant_uuid") or "tenant-demo",
            "state": payload.get("state") or "registered",
            "jwt_id": payload.get("jwt_id"),
            "capabilities_hash": payload.get("capabilities_hash"),
            "missed_heartbeats": 0,
            "last_ping_at": None,
            "closed_at": None,
            "created_at": _now_iso(),
            "updated_at": _now_iso(),
        }
        _MCP_SESSIONS[session_id] = record
        db = get_db().session()
        try:
            db.add(
                RuntimeSession(
                    id=record["id"],
                    runtime_assignment_id=record["runtime_assignment_id"],
                    tenant_uuid=record["tenant_uuid"],
                    state=record["state"],
                    jwt_id=record["jwt_id"],
                    capabilities_hash=record["capabilities_hash"],
                    missed_heartbeats=record["missed_heartbeats"],
                    last_ping_at=None,
                    closed_at=None,
                    created_at=datetime.utcnow(),
                    updated_at=datetime.utcnow(),
                )
            )
            db.commit()
        finally:
            db.close()
        return record

    def ack(self, session_id: str, payload: dict):
        record = _MCP_SESSIONS.get(session_id)
        if not record:
            record = {
                "id": session_id,
                "runtime_assignment_id": payload.get("runtime_assignment_id") or "",
                "tenant_uuid": payload.get("tenant_uuid") or "tenant-demo",
                "created_at": _now_iso(),
            }
        if payload.get("state"):
            record["state"] = payload["state"]
        if payload.get("capabilities_hash"):
            record["capabilities_hash"] = payload["capabilities_hash"]
        record["updated_at"] = _now_iso()
        _MCP_SESSIONS[session_id] = record
        self._update_status(session_id, record.get("state"))
        return record

    def heartbeat(self, session_id: str, payload: dict):
        record = _MCP_SESSIONS.get(session_id)
        if not record:
            record = {"id": session_id, "created_at": _now_iso()}
        record["missed_heartbeats"] = payload.get("missed_heartbeats", 0)
        record["last_ping_at"] = _now_iso()
        record["updated_at"] = _now_iso()
        _MCP_SESSIONS[session_id] = record
        return record

    def close(self, session_id: str, payload: dict):
        record = _MCP_SESSIONS.get(session_id)
        if not record:
            record = {"id": session_id, "created_at": _now_iso()}
        record["state"] = "closed"
        record["closed_at"] = _now_iso()
        record["updated_at"] = _now_iso()
        record["close_reason"] = payload.get("reason")
        _MCP_SESSIONS[session_id] = record
        self._update_status(session_id, "closed")
        return record

    def invoke(self, session_id: str, payload: dict):
        return {
            "status": "ok",
            "trace_id": payload.get("trace_id"),
            "correlation_id": payload.get("correlation_id"),
            "latency_ms": 0,
            "replay": None,
            "payload": {},
            "metadata": {},
        }

    def _update_status(self, session_id: str, status: str | None) -> None:
        if not status:
            return
        db = get_db().session()
        try:
            row = db.execute(
                select(RuntimeSession).where(RuntimeSession.id == session_id)
            ).scalar_one_or_none()
            if row:
                row.state = status
                row.updated_at = datetime.utcnow()
                db.commit()
        finally:
            db.close()
