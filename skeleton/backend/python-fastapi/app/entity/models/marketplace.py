from app.entity.models.base import BaseModel


class MarketplaceListing(BaseModel):
    __tablename__ = "marketplace_listings"


class MarketplaceListingAsset(BaseModel):
    __tablename__ = "marketplace_listing_assets"


class MarketplaceListingVersion(BaseModel):
    __tablename__ = "marketplace_listing_versions"


class MarketplacePricingPlan(BaseModel):
    __tablename__ = "marketplace_pricing_plans"


class MarketplacePlanTier(BaseModel):
    __tablename__ = "marketplace_plan_tiers"


class MarketplaceChecklistRun(BaseModel):
    __tablename__ = "marketplace_checklist_runs"


class MarketplaceChecklistItem(BaseModel):
    __tablename__ = "marketplace_checklist_items"


class MarketplaceLicense(BaseModel):
    __tablename__ = "marketplace_licenses"


class MarketplaceLicenseEvent(BaseModel):
    __tablename__ = "marketplace_license_events"


class MarketplaceTaxTransaction(BaseModel):
    __tablename__ = "marketplace_tax_transactions"


class MarketplaceUsageEnvelope(BaseModel):
    __tablename__ = "marketplace_usage_envelopes"


class MarketplaceUsageAggregate(BaseModel):
    __tablename__ = "marketplace_usage_aggregates"


class MarketplaceRevenueReport(BaseModel):
    __tablename__ = "marketplace_revenue_share_reports"


class MarketplaceNotification(BaseModel):
    __tablename__ = "marketplace_notifications"
