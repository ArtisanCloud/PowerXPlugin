from sqlalchemy import (
    Boolean,
    Column,
    DateTime,
    Integer,
    Numeric,
    String,
    text,
)
from sqlalchemy.dialects.postgresql import JSONB, UUID

from app.entity.models.base import Base


class MarketplaceListing(Base):
    __tablename__ = "marketplace_listings"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    plugin_id = Column(String, nullable=False, index=True)
    vendor_id = Column(String, nullable=False, index=True)
    status = Column(String, server_default=text("'draft'"), nullable=False, index=True)
    title = Column(String, nullable=False)
    slug = Column(String, nullable=False)
    summary = Column(String, nullable=True)
    description = Column(String, nullable=True)
    cover_asset_id = Column(UUID(as_uuid=False), nullable=True)
    hero_video_asset_id = Column(UUID(as_uuid=False), nullable=True)
    categories = Column(JSONB, nullable=True)
    tags = Column(JSONB, nullable=True)
    locale = Column(String, server_default=text("'en'"), nullable=False)
    version = Column(String, nullable=True)
    ready_checklist_score = Column(Integer, server_default=text("0"), nullable=False)
    recommended_weight = Column(Numeric(10, 4), server_default=text("0"), nullable=False)
    published_at = Column(DateTime(timezone=True), nullable=True)
    reviewed_at = Column(DateTime(timezone=True), nullable=True)
    reviewer_id = Column(String, nullable=True)
    audit_notes = Column(String, nullable=True)
    branding_theme = Column(JSONB, nullable=True)
    created_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    deleted_at = Column(DateTime(timezone=True), nullable=True)


class MarketplaceListingVersion(Base):
    __tablename__ = "marketplace_listing_versions"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    listing_id = Column(UUID(as_uuid=False), nullable=False, index=True)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    version = Column(String, nullable=False)
    changelog = Column(String, nullable=True)
    metadata_ = Column("metadata", JSONB, nullable=True)
    submitted_by = Column(String, nullable=False)
    review_state = Column(String, server_default=text("'draft'"), nullable=False)
    reviewer_id = Column(String, nullable=True)
    reviewed_at = Column(DateTime(timezone=True), nullable=True)
    created_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)


class MarketplaceListingAsset(Base):
    __tablename__ = "marketplace_listing_assets"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    listing_id = Column(UUID(as_uuid=False), nullable=False, index=True)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    asset_type = Column(String, nullable=False)
    storage_uri = Column(String, nullable=False)
    checksum = Column(String, nullable=True)
    is_primary = Column(Boolean, server_default=text("false"), nullable=False)
    locale = Column(String, server_default=text("'en'"), nullable=False)
    weight = Column(Integer, server_default=text("0"), nullable=False)
    metadata_ = Column("metadata", JSONB, nullable=True)
    created_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)


class MarketplacePricingPlan(Base):
    __tablename__ = "marketplace_pricing_plans"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    listing_id = Column(UUID(as_uuid=False), nullable=False, index=True)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    plan_code = Column(String, nullable=False)
    plan_type = Column(String, nullable=False)
    currency = Column(String, nullable=False)
    amount = Column(Numeric(18, 4), nullable=True)
    billing_period = Column(String, nullable=True)
    trial_period_days = Column(Integer, nullable=True)
    quota_limit = Column(Numeric(18, 4), nullable=True)
    overage_policy = Column(String, nullable=True)
    feature_matrix = Column(JSONB, nullable=True)
    is_default = Column(Boolean, server_default=text("false"), nullable=False)
    status = Column(String, server_default=text("'active'"), nullable=False)
    created_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)


class MarketplacePlanTier(Base):
    __tablename__ = "marketplace_plan_tiers"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    plan_id = Column(UUID(as_uuid=False), nullable=False, index=True)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    metric = Column(String, nullable=False)
    range_from = Column(Numeric(18, 4), nullable=False)
    range_to = Column(Numeric(18, 4), nullable=True)
    unit_amount = Column(Numeric(18, 4), nullable=False)
    unit_name = Column(String, nullable=True)
    created_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)


class MarketplaceChecklistRun(Base):
    __tablename__ = "marketplace_checklist_runs"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    listing_id = Column(UUID(as_uuid=False), nullable=False, index=True)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    trigger_source = Column(String, nullable=False)
    run_number = Column(Integer, server_default=text("1"), nullable=False)
    status = Column(String, server_default=text("'pending'"), nullable=False)
    started_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    completed_at = Column(DateTime(timezone=True), nullable=True)
    summary = Column(String, nullable=True)
    ci_pipeline_id = Column(String, nullable=True)
    created_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)


class MarketplaceChecklistItem(Base):
    __tablename__ = "marketplace_checklist_items"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    checklist_run_id = Column(UUID(as_uuid=False), nullable=False, index=True)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    code = Column(String, nullable=False)
    description = Column(String, nullable=False)
    result = Column(String, nullable=False)
    evidence_uri = Column(String, nullable=True)
    notes = Column(String, nullable=True)
    auto_fix_link = Column(String, nullable=True)
    created_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)


class MarketplaceLicense(Base):
    __tablename__ = "marketplace_licenses"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    listing_id = Column(UUID(as_uuid=False), nullable=False, index=True)
    plan_id = Column(UUID(as_uuid=False), nullable=False, index=True)
    license_token = Column(String, nullable=False)
    status = Column(String, server_default=text("'active'"), nullable=False)
    issued_at = Column(DateTime(timezone=True), nullable=False)
    expires_at = Column(DateTime(timezone=True), nullable=False)
    renewal_token = Column(String, nullable=True)
    offline_until = Column(DateTime(timezone=True), nullable=True)
    last_validated_at = Column(DateTime(timezone=True), nullable=True)
    issued_by = Column(String, nullable=True)
    metadata_ = Column("metadata", JSONB, nullable=True)
    created_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)


class MarketplaceLicenseEvent(Base):
    __tablename__ = "marketplace_license_events"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    license_id = Column(UUID(as_uuid=False), nullable=False, index=True)
    event_type = Column(String, nullable=False)
    event_payload = Column(JSONB, nullable=True)
    emitted_at = Column(DateTime(timezone=True), nullable=False)
    actor_id = Column(String, nullable=True)
    trace_id = Column(String, nullable=True)
    created_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)


class MarketplaceTaxTransaction(Base):
    __tablename__ = "marketplace_tax_transactions"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    billing_id = Column(String, nullable=False)
    external_provider = Column(String, nullable=False)
    external_transaction_id = Column(String, nullable=True)
    jurisdiction = Column(String, nullable=True)
    tax_amount = Column(Numeric(18, 4), nullable=False)
    currency = Column(String, nullable=False)
    settlement_currency = Column(String, nullable=True)
    exchange_rate = Column(Numeric(18, 6), nullable=True)
    tax_amount_settlement = Column(Numeric(18, 4), nullable=True)
    raw_payload = Column(JSONB, nullable=True)
    status = Column(String, server_default=text("'pending'"), nullable=False)
    synced_at = Column(DateTime(timezone=True), nullable=True)
    created_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)


class MarketplaceUsageEnvelope(Base):
    __tablename__ = "marketplace_usage_envelopes"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    license_id = Column(UUID(as_uuid=False), nullable=False, index=True)
    plugin_id = Column(String, nullable=False, index=True)
    metrics = Column(JSONB, nullable=False)
    timestamp_start = Column(DateTime(timezone=True), nullable=False)
    timestamp_end = Column(DateTime(timezone=True), nullable=False)
    signature = Column(String, nullable=False)
    checksum = Column(String, nullable=False, unique=True)
    ingest_status = Column(String, server_default=text("'processed'"), nullable=False)
    ingested_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    created_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)


class MarketplaceUsageAggregate(Base):
    __tablename__ = "marketplace_usage_aggregates"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    license_id = Column(UUID(as_uuid=False), nullable=False, index=True)
    metric = Column(String, nullable=False, index=True)
    window = Column(String, nullable=False, index=True)
    time_bucket = Column(DateTime(timezone=True), nullable=False, index=True)
    total = Column(Numeric(20, 4), nullable=False)
    delta = Column(Numeric(20, 4), nullable=False)
    currency = Column(String, nullable=True)
    revenue = Column(Numeric(18, 4), server_default=text("0"), nullable=False)
    created_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)


class MarketplaceRevenueReport(Base):
    __tablename__ = "marketplace_revenue_share_reports"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    vendor_id = Column(String, nullable=False, index=True)
    period_start = Column(DateTime(timezone=True), nullable=False)
    period_end = Column(DateTime(timezone=True), nullable=False)
    gross_amount = Column(Numeric(18, 4), nullable=False)
    vendor_share = Column(Numeric(18, 4), nullable=False)
    platform_share = Column(Numeric(18, 4), nullable=False)
    fees = Column(Numeric(18, 4), nullable=False)
    currency = Column(String, nullable=False)
    status = Column(String, server_default=text("'draft'"), nullable=False)
    generated_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    export_uri = Column(String, nullable=True)
    created_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)


class MarketplaceNotification(Base):
    __tablename__ = "marketplace_notifications"

    id = Column(UUID(as_uuid=False), primary_key=True, server_default=text("gen_random_uuid()"))
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    recipient_type = Column(String, nullable=False)
    recipient_id = Column(String, nullable=False)
    channel = Column(String, nullable=False)
    template_code = Column(String, nullable=False)
    payload = Column(JSONB, nullable=True)
    scheduled_at = Column(DateTime(timezone=True), nullable=True)
    sent_at = Column(DateTime(timezone=True), nullable=True)
    status = Column(String, server_default=text("'pending'"), nullable=False)
    created_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
