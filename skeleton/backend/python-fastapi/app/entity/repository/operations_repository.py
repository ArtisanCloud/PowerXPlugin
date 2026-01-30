from app.entity.models import (
    OperationsIncident,
    OperationsIncidentChecklist,
    OperationsIncidentUpdate,
    OperationsReadinessChecklistItem,
    OperationsSLAAdjustment,
    OperationsSLAProfile,
    OperationsSupportChannel,
    OperationsSupportTicket,
    OperationsSupportTicketEvent,
)
from app.entity.repository.base import BaseRepository


class OperationsRepository(BaseRepository):
    def list_support_channels(self, plugin_id: str | None = None):
        filters = []
        if plugin_id:
            filters.append(OperationsSupportChannel.plugin_id == plugin_id)
        return self.list(OperationsSupportChannel, filters)

    def list_support_tickets(self, tenant_uuid: str | None = None):
        filters = []
        if tenant_uuid:
            filters.append(OperationsSupportTicket.tenant_uuid == tenant_uuid)
        return self.list(OperationsSupportTicket, filters)

    def list_support_ticket_events(self, ticket_id: str | None = None):
        filters = []
        if ticket_id:
            filters.append(OperationsSupportTicketEvent.ticket_id == ticket_id)
        return self.list(OperationsSupportTicketEvent, filters)

    def list_incidents(self, plugin_id: str | None = None):
        filters = []
        if plugin_id:
            filters.append(OperationsIncident.plugin_id == plugin_id)
        return self.list(OperationsIncident, filters)

    def list_incident_updates(self, incident_id: str | None = None):
        filters = []
        if incident_id:
            filters.append(OperationsIncidentUpdate.incident_id == incident_id)
        return self.list(OperationsIncidentUpdate, filters)

    def list_incident_checklist_items(self, incident_id: str | None = None):
        filters = []
        if incident_id:
            filters.append(OperationsIncidentChecklist.incident_id == incident_id)
        return self.list(OperationsIncidentChecklist, filters)

    def list_readiness_items(self, plugin_id: str | None = None):
        filters = []
        if plugin_id:
            filters.append(OperationsReadinessChecklistItem.plugin_id == plugin_id)
        return self.list(OperationsReadinessChecklistItem, filters)

    def list_sla_profiles(self, plugin_id: str | None = None):
        filters = []
        if plugin_id:
            filters.append(OperationsSLAProfile.plugin_id == plugin_id)
        return self.list(OperationsSLAProfile, filters)

    def list_sla_adjustments(self, plugin_id: str | None = None):
        filters = []
        if plugin_id:
            filters.append(OperationsSLAAdjustment.plugin_id == plugin_id)
        return self.list(OperationsSLAAdjustment, filters)

    def create(self, entity):
        return self.add(entity)
