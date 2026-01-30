from sqlalchemy import Boolean, Column, DateTime, Integer, String, func, text
from sqlalchemy.dialects.postgresql import JSONB, UUID

from app.entity.models.base import Base


class AdminConsoleAuditEvent(Base):
    __tablename__ = "admin_console_audit_events"

    id = Column(UUID(as_uuid=False), primary_key=True)
    plugin_id = Column(String, nullable=False)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=True)
    actor_id = Column(String, nullable=False)
    actor_name = Column(String, nullable=True)
    actor_email = Column(String, nullable=True)
    permission_code = Column(String, nullable=False)
    action = Column(String, nullable=False)
    resource_type = Column(String, nullable=False)
    resource_ref = Column(String, nullable=True)
    summary = Column(String, nullable=True)
    diff = Column(JSONB, nullable=True)
    occurred_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)


class AdminConsoleConfigChange(Base):
    __tablename__ = "admin_console_config_changes"

    id = Column(UUID(as_uuid=False), primary_key=True)
    plugin_id = Column(String, nullable=False)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=True)
    section_key = Column(String, nullable=False)
    change_type = Column(String, nullable=False)
    previous_snapshot = Column(JSONB, nullable=True)
    next_snapshot = Column(JSONB, nullable=True)
    validation_summary = Column(JSONB, nullable=True)
    audit_event_id = Column(UUID(as_uuid=False), nullable=False)
    applied_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)


class AdminConsoleJobRun(Base):
    __tablename__ = "admin_console_job_runs"

    id = Column(UUID(as_uuid=False), primary_key=True)
    plugin_id = Column(String, nullable=False)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=True)
    environment = Column(String, nullable=True)
    job_type = Column(String, nullable=False)
    trigger_source = Column(String, nullable=False)
    status = Column(String, nullable=False)
    action = Column(String, nullable=True)
    scope_type = Column(String, nullable=True)
    scope_ref = Column(String, nullable=True)
    target_id = Column(String, nullable=True)
    reason = Column(String, nullable=True)
    dry_run = Column(Boolean, nullable=True)
    metadata_ = Column("metadata", JSONB, nullable=True)
    started_at = Column(DateTime(timezone=True), nullable=True)
    finished_at = Column(DateTime(timezone=True), nullable=True)
    duration_ms = Column(Integer, nullable=True)
    message = Column(String, nullable=True)
    retry_of = Column(UUID(as_uuid=False), nullable=True)
    audit_event_id = Column(UUID(as_uuid=False), nullable=True)
    created_by = Column(String, nullable=False)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)
