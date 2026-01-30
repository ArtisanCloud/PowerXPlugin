from app.entity.repository.operations_repository import OperationsRepository
from app.services._utils import to_list


class OperationsService:
    def __init__(self, repo: OperationsRepository | None = None) -> None:
        self._repo = repo or OperationsRepository()

    def list_support_channels(self, plugin_id: str | None = None) -> list:
        return to_list(self._repo.list_support_channels(plugin_id))

    def list_support_tickets(self, tenant_uuid: str | None = None) -> list:
        return to_list(self._repo.list_support_tickets(tenant_uuid))

    def list_support_ticket_events(self, ticket_id: str | None = None) -> list:
        return to_list(self._repo.list_support_ticket_events(ticket_id))

    def list_incidents(self, plugin_id: str | None = None) -> list:
        return to_list(self._repo.list_incidents(plugin_id))

    def list_incident_updates(self, incident_id: str | None = None) -> list:
        return to_list(self._repo.list_incident_updates(incident_id))

    def list_incident_checklist_items(self, incident_id: str | None = None) -> list:
        return to_list(self._repo.list_incident_checklist_items(incident_id))

    def list_readiness_items(self, plugin_id: str | None = None) -> list:
        return to_list(self._repo.list_readiness_items(plugin_id))

    def list_sla_profiles(self, plugin_id: str | None = None) -> list:
        return to_list(self._repo.list_sla_profiles(plugin_id))

    def list_sla_adjustments(self, plugin_id: str | None = None) -> list:
        return to_list(self._repo.list_sla_adjustments(plugin_id))
