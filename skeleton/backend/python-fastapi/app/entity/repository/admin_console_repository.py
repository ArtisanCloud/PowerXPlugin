from app.entity.models import AdminConsoleAuditEvent, AdminConsoleConfigChange, AdminConsoleJobRun
from app.entity.repository.base import BaseRepository


class AdminConsoleRepository(BaseRepository):
    def list_audit_events(self):
        return self.list(AdminConsoleAuditEvent)

    def list_config_changes(self, audit_event_id: str | None = None):
        filters = []
        if audit_event_id:
            filters.append(AdminConsoleConfigChange.audit_event_id == audit_event_id)
        return self.list(AdminConsoleConfigChange, filters)

    def list_job_runs(self, audit_event_id: str | None = None):
        filters = []
        if audit_event_id:
            filters.append(AdminConsoleJobRun.audit_event_id == audit_event_id)
        return self.list(AdminConsoleJobRun, filters)

    def create(self, entity):
        return self.add(entity)
