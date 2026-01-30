from sqlalchemy import Column, DateTime, String, func, text
from sqlalchemy.dialects.postgresql import JSONB, UUID

from app.entity.models.base import Base


class PrivacyDataClassification(Base):
    __tablename__ = "privacy_data_classifications"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False)
    asset_key = Column(String, nullable=False)
    category = Column(String, nullable=False)
    lawful_basis = Column(String, nullable=False)
    retention_policy = Column(JSONB, nullable=True)
    purpose = Column(String, nullable=False)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class PrivacyConsentToken(Base):
    __tablename__ = "privacy_consent_tokens"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False)
    consent_token = Column(String, nullable=False)
    scope = Column(JSONB, nullable=False)
    expires_at = Column(DateTime(timezone=True), nullable=True)
    issued_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    issued_by = Column(String, nullable=False)
    status = Column(String, server_default=text("'ACTIVE'"), nullable=False)
    revoked_at = Column(DateTime(timezone=True), nullable=True)
    revoked_reason = Column(String, nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class PrivacyLifecycleEvent(Base):
    __tablename__ = "privacy_lifecycle_events"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False)
    event_type = Column(String, nullable=False)
    asset_key = Column(String, nullable=False)
    payload = Column(JSONB, nullable=True)
    occurred_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    recorded_by = Column(String, nullable=False)
    status = Column(String, server_default=text("'PENDING'"), nullable=False)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
