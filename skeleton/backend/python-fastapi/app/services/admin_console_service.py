from app.entity.repository.admin_console_repository import AdminConsoleRepository
from app.services._utils import to_list


class AdminConsoleService:
    def __init__(self, repo: AdminConsoleRepository | None = None) -> None:
        self._repo = repo or AdminConsoleRepository()

    def list_audit_events(self) -> list:
        return to_list(self._repo.list_audit_events())

    def list_config_changes(self, audit_event_id: str | None = None) -> list:
        return to_list(self._repo.list_config_changes(audit_event_id))

    def list_job_runs(self, audit_event_id: str | None = None) -> list:
        return to_list(self._repo.list_job_runs(audit_event_id))
