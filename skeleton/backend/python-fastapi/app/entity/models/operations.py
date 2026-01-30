from sqlalchemy import (
    Boolean,
    Column,
    DateTime,
    Float,
    Integer,
    Numeric,
    String,
    text,
)
from sqlalchemy.dialects.postgresql import JSONB, UUID

from app.entity.models.base import Base


class OperationsSupportChannel(Base):
    __tablename__ = "operations_support_channels"

    id = Column(UUID(as_uuid=False), primary_key=True)
    plugin_id = Column(String, index=True)
    tenant_uuid = Column(UUID(as_uuid=False), index=True, nullable=True)
    channel = Column(String, nullable=True)
    is_enabled = Column(Boolean, nullable=True)
    service_window = Column(JSONB, nullable=True)
    escalation_path = Column(JSONB, nullable=True)
    metadata_ = Column("metadata", JSONB, nullable=True)
    version = Column(Integer, nullable=True)
    created_at = Column(DateTime(timezone=True), nullable=True)
    updated_at = Column(DateTime(timezone=True), nullable=True)


class OperationsSupportTicket(Base):
    __tablename__ = "operations_support_tickets"

    id = Column(UUID(as_uuid=False), primary_key=True)
    plugin_id = Column(String, index=True)
    tenant_uuid = Column(UUID(as_uuid=False), index=True, nullable=False)
    channel_id = Column(String, nullable=True)
    external_ref = Column(String, nullable=True)
    subject = Column(String, nullable=True)
    description = Column(String, nullable=True)
    priority = Column(String, nullable=True)
    status = Column(String, nullable=True)
    requested_by = Column(JSONB, nullable=True)
    assigned_team = Column(String, nullable=True)
    assigned_to = Column(String, nullable=True)
    first_response_at = Column(DateTime(timezone=True), nullable=True)
    resolved_at = Column(DateTime(timezone=True), nullable=True)
    closed_at = Column(DateTime(timezone=True), nullable=True)
    csat_score = Column(Float, nullable=True)
    resolution_code = Column(String, nullable=True)
    reopen_count = Column(Integer, nullable=True)
    created_at = Column(DateTime(timezone=True), nullable=True)
    updated_at = Column(DateTime(timezone=True), nullable=True)


class OperationsIncident(Base):
    __tablename__ = "operations_incidents"

    id = Column(UUID(as_uuid=False), primary_key=True)
    plugin_id = Column(String, index=True)
    tenant_uuid = Column(UUID(as_uuid=False), index=True, nullable=True)
    severity = Column(String, nullable=True)
    status = Column(String, nullable=True)
    detection_source = Column(String, nullable=True)
    summary = Column(String, nullable=True)
    impact = Column(JSONB, nullable=True)
    mitigation = Column(String, nullable=True)
    root_cause = Column(String, nullable=True)
    next_update_at = Column(DateTime(timezone=True), nullable=True)
    labels = Column(JSONB, nullable=True)
    confidentiality = Column(String, nullable=True)
    detected_at = Column(DateTime(timezone=True), nullable=True)
    acknowledged_at = Column(DateTime(timezone=True), nullable=True)
    mitigated_at = Column(DateTime(timezone=True), nullable=True)
    resolved_at = Column(DateTime(timezone=True), nullable=True)
    closed_at = Column(DateTime(timezone=True), nullable=True)
    created_at = Column(DateTime(timezone=True), nullable=True)
    updated_at = Column(DateTime(timezone=True), nullable=True)


class OperationsIncidentUpdate(Base):
    __tablename__ = "operations_incident_updates"

    id = Column(UUID(as_uuid=False), primary_key=True)
    incident_id = Column(String, index=True, nullable=True)
    entry_type = Column(String, nullable=True)
    message = Column(String, nullable=True)
    stakeholder_channel = Column(String, nullable=True)
    author_role = Column(String, nullable=True)
    posted_at = Column(DateTime(timezone=True), nullable=True)
    metadata_ = Column("metadata", JSONB, nullable=True)
    created_at = Column(DateTime(timezone=True), nullable=True)


class OperationsIncidentChecklist(Base):
    __tablename__ = "operations_incident_checklist"

    id = Column(UUID(as_uuid=False), primary_key=True)
    incident_id = Column(String, index=True, nullable=True)
    item_key = Column(String, nullable=True)
    description = Column(String, nullable=True)
    status = Column(String, nullable=True)
    completed_at = Column(DateTime(timezone=True), nullable=True)
    created_at = Column(DateTime(timezone=True), nullable=True)
    updated_at = Column(DateTime(timezone=True), nullable=True)


class OperationsSupportTicketEvent(Base):
    __tablename__ = "operations_support_ticket_events"

    id = Column(Integer, primary_key=True, autoincrement=True)
    ticket_id = Column(String, index=True, nullable=True)
    event_type = Column(String, nullable=True)
    payload = Column(JSONB, nullable=True)
    emitted_at = Column(DateTime(timezone=True), nullable=True)
    webhook_status = Column(String, nullable=True)
    retry_count = Column(Integer, nullable=True)
    created_at = Column(DateTime(timezone=True), nullable=True)


class OperationsReadinessChecklistItem(Base):
    __tablename__ = "operations_readiness_checklist_items"

    id = Column(UUID(as_uuid=False), primary_key=True)
    plugin_id = Column(String, index=True, nullable=True)
    type = Column(String, index=True, nullable=True)
    item_key = Column(String, nullable=True)
    description = Column(String, nullable=True)
    status = Column(String, nullable=True)
    owner_role = Column(String, nullable=True)
    due_date = Column(DateTime(timezone=True), nullable=True)
    completed_at = Column(DateTime(timezone=True), nullable=True)
    notes = Column(String, nullable=True)
    created_at = Column(DateTime(timezone=True), nullable=True)
    updated_at = Column(DateTime(timezone=True), nullable=True)


class OperationsSLAProfile(Base):
    __tablename__ = "operations_sla_profiles"

    id = Column(UUID(as_uuid=False), primary_key=True)
    plugin_id = Column(String, index=True, nullable=True)
    plan_type = Column(String, nullable=True)
    uptime_target = Column(Float, nullable=True)
    uptime_actual = Column(Float, nullable=True)
    response_target_ms = Column(Integer, nullable=True)
    response_actual_ms = Column(Integer, nullable=True)
    success_target_pct = Column(Float, nullable=True)
    success_actual_pct = Column(Float, nullable=True)
    support_frt_target_hours = Column(Float, nullable=True)
    support_frt_actual_hours = Column(Float, nullable=True)
    sla_score = Column(Float, nullable=True)
    incentive_applied_at = Column(DateTime(timezone=True), nullable=True)
    penalty_applied_at = Column(DateTime(timezone=True), nullable=True)
    notes = Column(String, nullable=True)
    computed_at = Column(DateTime(timezone=True), nullable=True)
    created_at = Column(DateTime(timezone=True), nullable=True)
    updated_at = Column(DateTime(timezone=True), nullable=True)


class OperationsSLAAdjustment(Base):
    __tablename__ = "operations_sla_adjustments"

    id = Column(Integer, primary_key=True, autoincrement=True)
    plugin_id = Column(String, nullable=True)
    plan_type = Column(String, nullable=True)
    period_start = Column(DateTime(timezone=True), nullable=True)
    period_end = Column(DateTime(timezone=True), nullable=True)
    score_before = Column(Float, nullable=True)
    score_after = Column(Float, nullable=True)
    action = Column(String, nullable=True)
    details = Column(String, nullable=True)
    applied_by = Column(String, nullable=True)
    created_at = Column(DateTime(timezone=True), nullable=True)
