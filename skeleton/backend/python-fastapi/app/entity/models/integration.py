from sqlalchemy import Column, DateTime, Integer, String, func, text
from sqlalchemy.dialects.postgresql import JSONB, UUID

from app.entity.models.base import Base


class IntegrationWebhookSubscription(Base):
    __tablename__ = "integration_webhook_subscriptions"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False)
    event_type = Column(String, nullable=False)
    target_url = Column(String, nullable=False)
    secret = Column(String, nullable=True)
    retry_policy = Column(JSONB, nullable=True)
    status = Column(String, server_default=text("'ACTIVE'"), nullable=False)
    metadata_ = Column("metadata", JSONB, server_default=text("'{}'::jsonb"), nullable=False)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class IntegrationWebhookAttempt(Base):
    __tablename__ = "integration_webhook_attempts"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    subscription_id = Column(UUID(as_uuid=False), nullable=False)
    envelope_id = Column(UUID(as_uuid=False), nullable=True)
    status = Column(String, nullable=False)
    retry_count = Column(Integer, server_default=text("0"), nullable=False)
    last_error = Column(String, nullable=True)
    next_delivery_at = Column(DateTime(timezone=True), nullable=True)
    payload_snapshot = Column(JSONB, nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class IntegrationSecret(Base):
    __tablename__ = "integration_secrets"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False)
    integration_type = Column(String, nullable=False)
    current_secret_ref = Column(String, nullable=True)
    pending_secret_ref = Column(String, nullable=True)
    rotation_interval_days = Column(Integer, server_default=text("30"), nullable=False)
    last_rotated_at = Column(DateTime(timezone=True), nullable=True)
    next_rotation_due_at = Column(DateTime(timezone=True), nullable=True)
    status = Column(String, server_default=text("'ACTIVE'"), nullable=False)
    audit_log = Column(JSONB, server_default=text("'[]'::jsonb"), nullable=True)
    metadata_ = Column("metadata", JSONB, server_default=text("'{}'::jsonb"), nullable=False)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class IntegrationGrantMatrixOverride(Base):
    __tablename__ = "integration_grant_matrix_overrides"

    id = Column(String, primary_key=True)
    scope = Column(String, nullable=False)
    channel = Column(String, nullable=False)
    resource = Column(String, nullable=False)
    action = Column(String, nullable=False)
    constraints = Column(JSONB, nullable=False)
    status = Column(String, nullable=False)
    version = Column(Integer, nullable=False)
    approved_by = Column(String, nullable=True)
    approved_at = Column(DateTime(timezone=True), nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)
