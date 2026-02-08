from sqlalchemy import Column, DateTime, Integer, String, Text, func, text
from sqlalchemy.dialects.postgresql import JSONB, UUID

from app.entity.models.base import Base


class IntegrationWebhookSubscription(Base):
    __tablename__ = "integration_webhook_subscriptions"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False)
    event_type = Column(Text, nullable=False)
    target_url = Column(Text, nullable=False)
    secret = Column(Text, nullable=True)
    retry_policy = Column(JSONB, nullable=True)
    status = Column(Text, server_default=text("'ACTIVE'"), nullable=False)
    metadata_ = Column("metadata", JSONB, server_default=text("'{}'::jsonb"), nullable=False)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class IntegrationWebhookAttempt(Base):
    __tablename__ = "integration_webhook_attempts"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    subscription_id = Column(UUID(as_uuid=False), nullable=False)
    envelope_id = Column(UUID(as_uuid=False), nullable=True)
    status = Column(Text, nullable=False)
    retry_count = Column(Integer, server_default=text("0"), nullable=False)
    last_error = Column(Text, nullable=True)
    next_delivery_at = Column(DateTime(timezone=True), nullable=True)
    payload_snapshot = Column(JSONB, nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class IntegrationSecret(Base):
    __tablename__ = "integration_secrets"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False)
    integration_type = Column(Text, nullable=False)
    current_secret_ref = Column(Text, nullable=True)
    pending_secret_ref = Column(Text, nullable=True)
    rotation_interval_days = Column(Integer, server_default=text("30"), nullable=False)
    last_rotated_at = Column(DateTime(timezone=True), nullable=True)
    next_rotation_due_at = Column(DateTime(timezone=True), nullable=True)
    status = Column(Text, server_default=text("'ACTIVE'"), nullable=False)
    audit_log = Column(JSONB, server_default=text("'[]'::jsonb"), nullable=True)
    metadata_ = Column("metadata", JSONB, server_default=text("'{}'::jsonb"), nullable=False)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class IntegrationGrantMatrixOverride(Base):
    __tablename__ = "integration_grant_matrix_overrides"

    id = Column(String, primary_key=True)
    scope = Column(Text, nullable=False)
    channel = Column(Text, nullable=False)
    resource = Column(Text, nullable=False)
    action = Column(Text, nullable=False)
    constraints = Column(JSONB, nullable=False)
    status = Column(Text, nullable=False)
    version = Column(Integer, nullable=False)
    approved_by = Column(Text, nullable=True)
    approved_at = Column(DateTime(timezone=True), nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class IntegrationChangeApproval(Base):
    __tablename__ = "integration_change_approvals"

    id = Column(String, primary_key=True)
    target_type = Column(Text, nullable=False)
    target_id = Column(Text, nullable=False)
    payload = Column(JSONB, nullable=False)
    status = Column(Text, nullable=False)
    submitted_by = Column(Text, nullable=False)
    submitted_at = Column(DateTime(timezone=True), nullable=False)
    reviewed_by = Column(Text, nullable=True)
    reviewed_at = Column(DateTime(timezone=True), nullable=True)
    reason = Column(Text, nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class IntegrationIdempotencyRecord(Base):
    __tablename__ = "integration_idempotency_records"

    key = Column(String, primary_key=True)
    tenant_uuid = Column(String, nullable=False)
    scope = Column(Text, nullable=True)
    operation = Column(Text, nullable=True)
    payload_hash = Column(Text, nullable=True)
    response_data = Column(JSONB, nullable=True)
    metadata_ = Column("metadata", JSONB, nullable=True)
    expires_at = Column(DateTime(timezone=True), nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
