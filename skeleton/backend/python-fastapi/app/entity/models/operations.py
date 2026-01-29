from app.entity.models.base import BaseModel


class OperationsSupportChannel(BaseModel):
    __tablename__ = "operations_support_channels"


class OperationsSupportTicket(BaseModel):
    __tablename__ = "operations_support_tickets"


class OperationsIncident(BaseModel):
    __tablename__ = "operations_incidents"


class OperationsIncidentUpdate(BaseModel):
    __tablename__ = "operations_incident_updates"


class OperationsIncidentChecklist(BaseModel):
    __tablename__ = "operations_incident_checklist"


class OperationsSupportTicketEvent(BaseModel):
    __tablename__ = "operations_support_ticket_events"


class OperationsReadinessChecklistItem(BaseModel):
    __tablename__ = "operations_readiness_checklist_items"


class OperationsSLAProfile(BaseModel):
    __tablename__ = "operations_sla_profiles"


class OperationsSLAAdjustment(BaseModel):
    __tablename__ = "operations_sla_adjustments"
