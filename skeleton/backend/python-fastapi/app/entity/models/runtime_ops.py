from sqlalchemy import Boolean, Column, DateTime, Integer, Numeric, Text, func, text
from sqlalchemy.dialects.postgresql import JSONB, UUID

from app.entity.models.base import Base


class RuntimeAuditEvent(Base):
    __tablename__ = "runtime_audit_events"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    plugin_id = Column(Text, nullable=False)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=True)
    event_type = Column(Text, nullable=False)
    payload = Column(JSONB, nullable=True)
    occurred_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)


class MarketplaceOverage(Base):
    __tablename__ = "marketplace_overages"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    plugin_id = Column(Text, nullable=False)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=True)
    hour_window = Column(DateTime(timezone=True), nullable=False)
    quota_metric = Column(Text, nullable=False)
    breach_count = Column(Integer, server_default=text("0"), nullable=False)
    last_breach_at = Column(DateTime(timezone=True), nullable=True)
    reported = Column(Boolean, server_default=text("false"), nullable=False)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class QuotaLedger(Base):
    __tablename__ = "quota_ledgers"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    scope_type = Column(Text, nullable=False)
    scope_ref = Column(Text, nullable=False)
    window_start = Column(DateTime(timezone=True), nullable=False)
    window_end = Column(DateTime(timezone=True), nullable=False)
    tokens_consumed = Column(Numeric, server_default=text("0"), nullable=True)
    cpu_seconds = Column(Numeric, server_default=text("0"), nullable=True)
    bandwidth_mb = Column(Numeric, server_default=text("0"), nullable=True)
    invocations = Column(Numeric, server_default=text("0"), nullable=True)
    over_limit_action = Column(Text, nullable=True)
    reported_at = Column(DateTime(timezone=True), nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
