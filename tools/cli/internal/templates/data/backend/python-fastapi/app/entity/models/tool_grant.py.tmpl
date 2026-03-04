from sqlalchemy import Column, DateTime, Text, func
from sqlalchemy.dialects.postgresql import JSONB, UUID
from sqlalchemy.sql import text

from app.entity.models.base import Base


class ToolGrantRevocation(Base):
    __tablename__ = "tool_grant_revocations"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False)
    toolgrant_id = Column(Text, nullable=False)
    revoked_at = Column(DateTime(timezone=True), nullable=False)
    revoked_by = Column(Text, nullable=False)
    reason = Column(Text, nullable=True)
    ttl_expiry = Column(DateTime(timezone=True), nullable=False)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)


class ToolGrantUsageEvent(Base):
    __tablename__ = "tool_grant_usage_events"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False)
    toolgrant_id = Column(Text, nullable=False)
    event_type = Column(Text, nullable=False)
    capability = Column(Text, nullable=False)
    agent_id = Column(Text, nullable=False)
    occurred_at = Column(DateTime(timezone=True), nullable=False)
    metadata_ = Column("metadata", JSONB, nullable=True)
