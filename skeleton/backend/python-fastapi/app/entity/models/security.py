from sqlalchemy import Column, DateTime, Text, func, text
from sqlalchemy.dialects.postgresql import JSONB, UUID

from app.entity.models.base import Base


class SecurityBaselineChecklist(Base):
    __tablename__ = "security_baseline_checklists"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    version = Column(Text, nullable=False)
    controls = Column(JSONB, nullable=False)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    retired_at = Column(DateTime(timezone=True), nullable=True)


class SecurityAuditReport(Base):
    __tablename__ = "security_audit_reports"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    baseline_id = Column(UUID(as_uuid=False), nullable=False)
    initiated_by = Column(Text, nullable=False)
    status = Column(Text, nullable=False)
    findings = Column(JSONB, nullable=True)
    artifact_path = Column(Text, nullable=True)
    sarif_path = Column(Text, nullable=True)
    report_hash = Column(Text, nullable=True)
    checklist_version = Column(Text, nullable=False)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)


class SecurityVulnerabilityAdvisory(Base):
    __tablename__ = "security_vulnerability_advisories"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    reference = Column(Text, nullable=False)
    severity = Column(Text, nullable=False)
    status = Column(Text, nullable=False)
    affected_versions = Column(JSONB, nullable=False)
    patched_in_version = Column(Text, nullable=True)
    summary = Column(Text, nullable=False)
    details_markdown = Column(Text, nullable=True)
    published_at = Column(DateTime(timezone=True), nullable=True)
    patched_at = Column(DateTime(timezone=True), nullable=True)
    closed_at = Column(DateTime(timezone=True), nullable=True)
    sla_deadline = Column(DateTime(timezone=True), nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class SecurityAdvisoryDistribution(Base):
    __tablename__ = "security_advisory_distributions"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    advisory_id = Column(UUID(as_uuid=False), nullable=False)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False)
    channel = Column(Text, nullable=False)
    delivered_at = Column(DateTime(timezone=True), nullable=True)
    status = Column(Text, nullable=False)
    metadata_ = Column("metadata", JSONB, nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)
