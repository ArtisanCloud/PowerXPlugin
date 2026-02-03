from sqlalchemy import BigInteger, Boolean, Column, DateTime, Text, text
from sqlalchemy.dialects.postgresql import JSONB, UUID

from app.entity.models.base import Base


class AdminConsoleAuditEvent(Base):
    __tablename__ = "admin_console_audit_events"

    id = Column(UUID(as_uuid=False), primary_key=True)
    plugin_id = Column(Text, nullable=False)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=True)
    actor_id = Column(Text, nullable=False)
    actor_name = Column(Text, nullable=True)
    actor_email = Column(Text, nullable=True)
    permission_code = Column(Text, nullable=False)
    action = Column(Text, nullable=False)
    resource_type = Column(Text, nullable=False)
    resource_ref = Column(Text, nullable=True)
    summary = Column(Text, nullable=True)
    diff = Column(JSONB, nullable=True)
    occurred_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    created_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)


class AdminConsoleConfigChange(Base):
    __tablename__ = "admin_console_config_changes"

    id = Column(UUID(as_uuid=False), primary_key=True)
    plugin_id = Column(Text, nullable=False)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=True)
    section_key = Column(Text, nullable=False)
    change_type = Column(Text, nullable=False)
    previous_snapshot = Column(JSONB, nullable=True)
    next_snapshot = Column(JSONB, nullable=True)
    validation_summary = Column(JSONB, nullable=True)
    audit_event_id = Column(UUID(as_uuid=False), nullable=False)
    applied_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)


class AdminConsoleJobRun(Base):
    __tablename__ = "admin_console_job_runs"

    id = Column(UUID(as_uuid=False), primary_key=True)
    plugin_id = Column(Text, nullable=False)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=True)
    environment = Column(Text, nullable=True)
    job_type = Column(Text, nullable=False)
    trigger_source = Column(Text, nullable=False)
    status = Column(Text, nullable=False)
    action = Column(Text, nullable=True)
    scope_type = Column(Text, nullable=True)
    scope_ref = Column(Text, nullable=True)
    target_id = Column(Text, nullable=True)
    reason = Column(Text, nullable=True)
    dry_run = Column(Boolean, nullable=True)
    metadata_ = Column("metadata", JSONB, nullable=True)
    started_at = Column(DateTime(timezone=True), nullable=True)
    finished_at = Column(DateTime(timezone=True), nullable=True)
    duration_ms = Column(BigInteger, nullable=True)
    message = Column(Text, nullable=True)
    retry_of = Column(UUID(as_uuid=False), nullable=True)
    audit_event_id = Column(UUID(as_uuid=False), nullable=True)
    created_by = Column(Text, nullable=False)
    created_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
