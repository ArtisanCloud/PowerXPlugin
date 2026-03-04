from __future__ import annotations

from datetime import datetime, timezone
from uuid import uuid4

from app.entity.models import SecurityVulnerabilityAdvisory
from app.entity.repository.security_repository import SecurityRepository
from app.services._utils import to_dict, to_list


def _now() -> datetime:
    return datetime.utcnow()


class SecurityService:
    def __init__(self, repo: SecurityRepository | None = None) -> None:
        self._repo = repo or SecurityRepository()

    def list_baselines(self) -> list:
        return to_list(self._repo.list_baselines())

    def list_audit_reports(self, limit: int = 0) -> list:
        return to_list(self._repo.list_audit_reports(limit))

    def list_advisories(self, severity: list[str] | None = None, status: list[str] | None = None, limit: int = 0) -> list:
        items = self._repo.list_advisories(severity, status, limit)
        return [_advisory_payload(item) for item in items]

    def list_distributions(self, tenant_uuid: str | None = None) -> list:
        return to_list(self._repo.list_distributions(tenant_uuid))

    def create_advisory(self, payload: dict) -> dict:
        advisory = SecurityVulnerabilityAdvisory(
            id=payload.get("id"),
            reference=payload.get("reference") or "",
            severity=payload.get("severity") or "",
            status=payload.get("status") or "OPEN",
            affected_versions=payload.get("affected_versions") or [],
            patched_in_version=payload.get("patched_in_version"),
            summary=payload.get("summary") or "",
            details_markdown=payload.get("details_markdown"),
            published_at=payload.get("published_at"),
            patched_at=payload.get("patched_at"),
            closed_at=payload.get("closed_at"),
            sla_deadline=payload.get("sla_deadline"),
            created_at=_now(),
            updated_at=_now(),
        )
        advisory = self._repo.create_advisory(advisory)
        return _advisory_payload(advisory)

    def publish_advisory(self, advisory_id: str, payload: dict) -> dict | None:
        updates = {
            "status": "PUBLISHED",
            "patched_in_version": payload.get("patched_in_version"),
            "published_at": payload.get("published_at") or _now(),
            "patched_at": payload.get("patched_at") or _now(),
            "updated_at": _now(),
        }
        advisory = self._repo.update_advisory(advisory_id, updates)
        if not advisory:
            return None
        return _advisory_payload(advisory)


def _fmt(dt: datetime | None) -> str:
    if not dt:
        return ""
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=timezone.utc)
    return dt.astimezone(timezone.utc).isoformat().replace("+00:00", "Z")


def _advisory_payload(advisory) -> dict:
    if not advisory:
        return {}
    payload = {
        "id": advisory.id,
        "reference": advisory.reference,
        "severity": advisory.severity,
        "status": advisory.status,
        "affected_versions": advisory.affected_versions or [],
        "patched_in_version": advisory.patched_in_version,
        "summary": advisory.summary,
        "details_markdown": advisory.details_markdown,
        "published_at": _fmt(advisory.published_at),
        "patched_at": _fmt(advisory.patched_at),
        "closed_at": _fmt(advisory.closed_at),
        "sla_deadline": _fmt(advisory.sla_deadline),
        "created_at": _fmt(advisory.created_at),
    }
    return payload
