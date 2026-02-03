from sqlalchemy import (
    Boolean,
    Column,
    DateTime,
    Index,
    Integer,
    Numeric,
    Text,
    UniqueConstraint,
    func,
    text,
)
from sqlalchemy.dialects.postgresql import JSONB, UUID

from app.entity.models.base import Base


class MarketplaceListing(Base):
    __tablename__ = "marketplace_listings"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    plugin_id = Column(Text, nullable=False, index=True)
    vendor_id = Column(Text, nullable=False, index=True)
    status = Column(Text, server_default=text("'draft'"), nullable=False, index=True)
    title = Column(Text, nullable=False)
    slug = Column(Text, nullable=False)
    summary = Column(Text, nullable=True)
    description = Column(Text, nullable=True)
    cover_asset_id = Column(UUID(as_uuid=False), nullable=True)
    hero_video_asset_id = Column(UUID(as_uuid=False), nullable=True)
    categories = Column(JSONB, nullable=True)
    tags = Column(JSONB, nullable=True)
    locale = Column(Text, server_default=text("'en'"), nullable=False)
    version = Column(Text, nullable=True)
    ready_checklist_score = Column(Integer, server_default=text("0"), nullable=False)
    recommended_weight = Column(Numeric(10, 4), server_default=text("0"), nullable=False)
    published_at = Column(DateTime(timezone=True), nullable=True)
    reviewed_at = Column(DateTime(timezone=True), nullable=True)
    reviewer_id = Column(Text, nullable=True)
    audit_notes = Column(Text, nullable=True)
    branding_theme = Column(JSONB, nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)
    deleted_at = Column(DateTime(timezone=True), nullable=True)


class MarketplaceListingVersion(Base):
    __tablename__ = "marketplace_listing_versions"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    listing_id = Column(UUID(as_uuid=False), nullable=False, index=True)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    version = Column(Text, nullable=False)
    changelog = Column(Text, nullable=True)
    metadata_ = Column("metadata", JSONB, nullable=True)
    submitted_by = Column(Text, nullable=False)
    review_state = Column(Text, server_default=text("'draft'"), nullable=False)
    reviewer_id = Column(Text, nullable=True)
    reviewed_at = Column(DateTime(timezone=True), nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)


class MarketplaceListingAsset(Base):
    __tablename__ = "marketplace_listing_assets"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    listing_id = Column(UUID(as_uuid=False), nullable=False, index=True)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    asset_type = Column(Text, nullable=False)
    storage_uri = Column(Text, nullable=False)
    checksum = Column(Text, nullable=True)
    is_primary = Column(Boolean, server_default=text("false"), nullable=False)
    locale = Column(Text, server_default=text("'en'"), nullable=False)
    weight = Column(Integer, server_default=text("0"), nullable=False)
    metadata_ = Column("metadata", JSONB, nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class MarketplacePricingPlan(Base):
    __tablename__ = "marketplace_pricing_plans"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    listing_id = Column(UUID(as_uuid=False), nullable=False, index=True)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    plan_code = Column(Text, nullable=False)
    plan_type = Column(Text, nullable=False)
    currency = Column(Text, nullable=False)
    amount = Column(Numeric(18, 4), nullable=True)
    billing_period = Column(Text, nullable=True)
    trial_period_days = Column(Integer, nullable=True)
    quota_limit = Column(Numeric(18, 4), nullable=True)
    overage_policy = Column(Text, nullable=True)
    feature_matrix = Column(JSONB, nullable=True)
    is_default = Column(Boolean, server_default=text("false"), nullable=False)
    status = Column(Text, server_default=text("'active'"), nullable=False)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class MarketplacePlanTier(Base):
    __tablename__ = "marketplace_plan_tiers"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    plan_id = Column(UUID(as_uuid=False), nullable=False, index=True)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    metric = Column(Text, nullable=False)
    range_from = Column(Numeric(18, 4), nullable=False)
    range_to = Column(Numeric(18, 4), nullable=True)
    unit_amount = Column(Numeric(18, 4), nullable=False)
    unit_name = Column(Text, nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class MarketplaceChecklistRun(Base):
    __tablename__ = "marketplace_checklist_runs"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    listing_id = Column(UUID(as_uuid=False), nullable=False, index=True)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    trigger_source = Column(Text, nullable=False)
    run_number = Column(Integer, server_default=text("1"), nullable=False)
    status = Column(Text, server_default=text("'pending'"), nullable=False)
    started_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    completed_at = Column(DateTime(timezone=True), nullable=True)
    summary = Column(Text, nullable=True)
    ci_pipeline_id = Column(Text, nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)


class MarketplaceChecklistItem(Base):
    __tablename__ = "marketplace_checklist_items"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    checklist_run_id = Column(UUID(as_uuid=False), nullable=False, index=True)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    code = Column(Text, nullable=False)
    description = Column(Text, nullable=False)
    result = Column(Text, nullable=False)
    evidence_uri = Column(Text, nullable=True)
    notes = Column(Text, nullable=True)
    auto_fix_link = Column(Text, nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class MarketplaceLicense(Base):
    __tablename__ = "marketplace_licenses"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    listing_id = Column(UUID(as_uuid=False), nullable=False, index=True)
    plan_id = Column(UUID(as_uuid=False), nullable=False, index=True)
    license_token = Column(Text, nullable=False)
    status = Column(Text, server_default=text("'active'"), nullable=False)
    issued_at = Column(DateTime(timezone=True), nullable=False)
    expires_at = Column(DateTime(timezone=True), nullable=False)
    renewal_token = Column(Text, nullable=True)
    offline_until = Column(DateTime(timezone=True), nullable=True)
    last_validated_at = Column(DateTime(timezone=True), nullable=True)
    issued_by = Column(Text, nullable=True)
    metadata_ = Column("metadata", JSONB, nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class MarketplaceLicenseEvent(Base):
    __tablename__ = "marketplace_license_events"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    license_id = Column(UUID(as_uuid=False), nullable=False, index=True)
    event_type = Column(Text, nullable=False)
    event_payload = Column(JSONB, nullable=True)
    emitted_at = Column(DateTime(timezone=True), nullable=False)
    actor_id = Column(Text, nullable=True)
    trace_id = Column(Text, nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)


class MarketplaceTaxTransaction(Base):
    __tablename__ = "marketplace_tax_transactions"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    billing_id = Column(Text, nullable=False)
    external_provider = Column(Text, nullable=False)
    external_transaction_id = Column(Text, nullable=True)
    jurisdiction = Column(Text, nullable=True)
    tax_amount = Column(Numeric(18, 4), nullable=False)
    currency = Column(Text, nullable=False)
    settlement_currency = Column(Text, nullable=True)
    exchange_rate = Column(Numeric(18, 6), nullable=True)
    tax_amount_settlement = Column(Numeric(18, 4), nullable=True)
    raw_payload = Column(JSONB, nullable=True)
    status = Column(Text, server_default=text("'pending'"), nullable=False)
    synced_at = Column(DateTime(timezone=True), nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class MarketplaceUsageEnvelope(Base):
    __tablename__ = "marketplace_usage_envelopes"
    __table_args__ = (UniqueConstraint("checksum", name="uq_usage_checksum"),)

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    license_id = Column(UUID(as_uuid=False), nullable=False, index=True)
    plugin_id = Column(Text, nullable=False, index=True)
    metrics = Column(JSONB, nullable=False)
    timestamp_start = Column(DateTime(timezone=True), nullable=False)
    timestamp_end = Column(DateTime(timezone=True), nullable=False)
    signature = Column(Text, nullable=False)
    checksum = Column(Text, nullable=False)
    ingest_status = Column(Text, server_default=text("'processed'"), nullable=False)
    ingested_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class MarketplaceUsageAggregate(Base):
    __tablename__ = "marketplace_usage_aggregates"
    __table_args__ = (Index("idx_usage_agg_metric", "metric", "window", "time_bucket"),)

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    license_id = Column(UUID(as_uuid=False), nullable=False, index=True)
    metric = Column(Text, nullable=False)
    window = Column(Text, nullable=False)
    time_bucket = Column(DateTime(timezone=True), nullable=False)
    total = Column(Numeric(20, 4), nullable=False)
    delta = Column(Numeric(20, 4), nullable=False)
    currency = Column(Text, nullable=True)
    revenue = Column(Numeric(18, 4), server_default=text("0"), nullable=False)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class MarketplaceRevenueReport(Base):
    __tablename__ = "marketplace_revenue_share_reports"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    vendor_id = Column(Text, nullable=False, index=True)
    period_start = Column(DateTime(timezone=True), nullable=False)
    period_end = Column(DateTime(timezone=True), nullable=False)
    gross_amount = Column(Numeric(18, 4), nullable=False)
    vendor_share = Column(Numeric(18, 4), nullable=False)
    platform_share = Column(Numeric(18, 4), nullable=False)
    fees = Column(Numeric(18, 4), nullable=False)
    currency = Column(Text, nullable=False)
    status = Column(Text, server_default=text("'draft'"), nullable=False)
    generated_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    export_uri = Column(Text, nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)


class MarketplaceNotification(Base):
    __tablename__ = "marketplace_notifications"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    recipient_type = Column(Text, nullable=False)
    recipient_id = Column(Text, nullable=False)
    channel = Column(Text, nullable=False)
    template_code = Column(Text, nullable=False)
    payload = Column(JSONB, nullable=True)
    scheduled_at = Column(DateTime(timezone=True), nullable=True)
    sent_at = Column(DateTime(timezone=True), nullable=True)
    status = Column(Text, server_default=text("'pending'"), nullable=False)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)
