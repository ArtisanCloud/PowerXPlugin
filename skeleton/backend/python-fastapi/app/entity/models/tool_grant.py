from sqlalchemy import Column, DateTime, String, func
from sqlalchemy.dialects.postgresql import JSONB, UUID
from sqlalchemy.sql import text

from app.entity.models.base import Base


class ToolGrantRevocation(Base):
    __tablename__ = "tool_grant_revocations"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False)
    toolgrant_id = Column(String, nullable=False)
    revoked_at = Column(DateTime(timezone=True), nullable=False)
    revoked_by = Column(String, nullable=False)
    reason = Column(String, nullable=True)
    ttl_expiry = Column(DateTime(timezone=True), nullable=False)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)


class ToolGrantUsageEvent(Base):
    __tablename__ = "tool_grant_usage_events"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False)
    toolgrant_id = Column(String, nullable=False)
    event_type = Column(String, nullable=False)
    capability = Column(String, nullable=False)
    agent_id = Column(String, nullable=False)
    occurred_at = Column(DateTime(timezone=True), nullable=False)
    metadata_ = Column("metadata", JSONB, nullable=True)
