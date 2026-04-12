from sqlalchemy import (
    BigInteger,
    Boolean,
    Column,
    DateTime,
    Float,
    Index,
    Integer,
    Text,
    func,
)
from sqlalchemy.dialects.postgresql import JSONB, UUID

from app.entity.models.base import Base


class OperationsSupportChannel(Base):
    __tablename__ = "operations_support_channels"
    __table_args__ = (Index("idx_support_channels_scope", "plugin_id", "tenant_uuid"),)

    id = Column(UUID(as_uuid=False), primary_key=True)
    plugin_id = Column(Text, nullable=True)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=True)
    channel = Column(Text, nullable=True)
    is_enabled = Column(Boolean, nullable=True)
    service_window = Column(JSONB, nullable=True)
    escalation_path = Column(JSONB, nullable=True)
    metadata_ = Column("metadata", JSONB, nullable=True)
    version = Column(Integer, nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class OperationsSupportTicket(Base):
    __tablename__ = "operations_support_tickets"
    __table_args__ = (Index("idx_support_tickets_scope", "plugin_id", "tenant_uuid"),)

    id = Column(UUID(as_uuid=False), primary_key=True)
    plugin_id = Column(Text, nullable=True)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False)
    channel_id = Column(Text, nullable=True)
    external_ref = Column(Text, nullable=True)
    subject = Column(Text, nullable=True)
    description = Column(Text, nullable=True)
    priority = Column(Text, nullable=True)
    status = Column(Text, nullable=True)
    requested_by = Column(JSONB, nullable=True)
    assigned_team = Column(Text, nullable=True)
    assigned_to = Column(Text, nullable=True)
    first_response_at = Column(DateTime(timezone=True), nullable=True)
    resolved_at = Column(DateTime(timezone=True), nullable=True)
    closed_at = Column(DateTime(timezone=True), nullable=True)
    csat_score = Column(Float, nullable=True)
    resolution_code = Column(Text, nullable=True)
    reopen_count = Column(Integer, nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class OperationsIncident(Base):
    __tablename__ = "operations_incidents"
    __table_args__ = (Index("idx_operations_incidents_scope", "plugin_id", "tenant_uuid"),)

    id = Column(UUID(as_uuid=False), primary_key=True)
    plugin_id = Column(Text, nullable=True)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=True)
    severity = Column(Text, nullable=True)
    status = Column(Text, nullable=True)
    detection_source = Column(Text, nullable=True)
    summary = Column(Text, nullable=True)
    impact = Column(JSONB, nullable=True)
    mitigation = Column(Text, nullable=True)
    root_cause = Column(Text, nullable=True)
    next_update_at = Column(DateTime(timezone=True), nullable=True)
    labels = Column(JSONB, nullable=True)
    confidentiality = Column(Text, nullable=True)
    detected_at = Column(DateTime(timezone=True), nullable=True)
    acknowledged_at = Column(DateTime(timezone=True), nullable=True)
    mitigated_at = Column(DateTime(timezone=True), nullable=True)
    resolved_at = Column(DateTime(timezone=True), nullable=True)
    closed_at = Column(DateTime(timezone=True), nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class OperationsIncidentUpdate(Base):
    __tablename__ = "operations_incident_updates"

    id = Column(UUID(as_uuid=False), primary_key=True)
    incident_id = Column(Text, index=True, nullable=True)
    entry_type = Column(Text, nullable=True)
    message = Column(Text, nullable=True)
    stakeholder_channel = Column(Text, nullable=True)
    author_role = Column(Text, nullable=True)
    posted_at = Column(DateTime(timezone=True), nullable=True)
    metadata_ = Column("metadata", JSONB, nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)


class OperationsIncidentChecklist(Base):
    __tablename__ = "operations_incident_checklist"

    id = Column(UUID(as_uuid=False), primary_key=True)
    incident_id = Column(Text, index=True, nullable=True)
    item_key = Column(Text, nullable=True)
    description = Column(Text, nullable=True)
    status = Column(Text, nullable=True)
    completed_at = Column(DateTime(timezone=True), nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class OperationsSupportTicketEvent(Base):
    __tablename__ = "operations_support_ticket_events"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    ticket_id = Column(Text, index=True, nullable=True)
    event_type = Column(Text, nullable=True)
    payload = Column(JSONB, nullable=True)
    emitted_at = Column(DateTime(timezone=True), nullable=True)
    webhook_status = Column(Text, nullable=True)
    retry_count = Column(Integer, nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)


class OperationsReadinessChecklistItem(Base):
    __tablename__ = "operations_readiness_checklist_items"
    __table_args__ = (Index("idx_operations_readiness_type", "plugin_id", "type"),)

    id = Column(UUID(as_uuid=False), primary_key=True)
    plugin_id = Column(Text, nullable=True)
    type = Column(Text, nullable=True)
    item_key = Column(Text, nullable=True)
    description = Column(Text, nullable=True)
    status = Column(Text, nullable=True)
    owner_role = Column(Text, nullable=True)
    due_date = Column(DateTime(timezone=True), nullable=True)
    completed_at = Column(DateTime(timezone=True), nullable=True)
    notes = Column(Text, nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class OperationsSLAProfile(Base):
    __tablename__ = "operations_sla_profiles"
    __table_args__ = (Index("idx_sla_profiles_plugin", "plugin_id"),)

    id = Column(UUID(as_uuid=False), primary_key=True)
    plugin_id = Column(Text, nullable=True)
    plan_type = Column(Text, nullable=True)
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
    notes = Column(Text, nullable=True)
    computed_at = Column(DateTime(timezone=True), nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class OperationsSLAAdjustment(Base):
    __tablename__ = "operations_sla_adjustments"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    plugin_id = Column(Text, nullable=True)
    plan_type = Column(Text, nullable=True)
    period_start = Column(DateTime(timezone=True), nullable=True)
    period_end = Column(DateTime(timezone=True), nullable=True)
    score_before = Column(Float, nullable=True)
    score_after = Column(Float, nullable=True)
    action = Column(Text, nullable=True)
    details = Column(Text, nullable=True)
    applied_by = Column(Text, nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
