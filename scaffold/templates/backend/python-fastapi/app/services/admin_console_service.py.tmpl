from __future__ import annotations

from datetime import datetime

from app.entity.models import AdminConsoleJobRun
from app.entity.repository.admin_console_repository import AdminConsoleRepository
from app.services._utils import to_dict, to_list


def _now() -> datetime:
    return datetime.utcnow()


class AdminConsoleService:
    def __init__(self, repo: AdminConsoleRepository | None = None) -> None:
        self._repo = repo or AdminConsoleRepository()

    def list_audit_events(self) -> list:
        return to_list(self._repo.list_audit_events())

    def list_config_changes(self, audit_event_id: str | None = None) -> list:
        return to_list(self._repo.list_config_changes(audit_event_id))

    def list_job_runs(self, audit_event_id: str | None = None) -> list:
        return to_list(self._repo.list_job_runs(audit_event_id))

    def list_config_sections(self) -> list:
        return to_list(self._repo.list_config_changes())

    def update_config_section(self, section_key: str, payload: dict) -> dict:
        return {"section_key": section_key, "payload": payload}

    def export_audit_events(self) -> dict:
        return {"items": self.list_audit_events()}

    def retry_job_run(self, run_id: str) -> dict:
        run = AdminConsoleJobRun(
            id=None,
            plugin_id="",
            tenant_uuid=None,
            environment=None,
            job_type="retry",
            trigger_source="manual",
            status="retrying",
            action=None,
            scope_type=None,
            scope_ref=None,
            target_id=None,
            reason=None,
            dry_run=False,
            metadata_=None,
            started_at=_now(),
            finished_at=None,
            duration_ms=None,
            message=None,
            retry_of=None,
            audit_event_id=None,
            created_by="",
            created_at=_now(),
            updated_at=_now(),
        )
        self._repo.create(run)
        return to_dict(run)

    def execute_safe_op(self, payload: dict) -> dict:
        return {"ok": True, "action": payload.get("action"), "requested_at": _now().isoformat() + "Z"}

    def troubleshoot_summary(self) -> dict:
        return {"ok": True, "summary": {}}
