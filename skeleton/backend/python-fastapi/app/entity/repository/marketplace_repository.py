from app.entity.models import (
    MarketplaceChecklistItem,
    MarketplaceChecklistRun,
    MarketplaceLicense,
    MarketplaceLicenseEvent,
    MarketplaceListing,
    MarketplaceListingAsset,
    MarketplaceListingVersion,
    MarketplaceNotification,
    MarketplacePlanTier,
    MarketplacePricingPlan,
    MarketplaceRevenueReport,
    MarketplaceTaxTransaction,
    MarketplaceUsageAggregate,
    MarketplaceUsageEnvelope,
)
from app.entity.repository.base import BaseRepository


class MarketplaceRepository(BaseRepository):
    def list_listings(self, tenant_uuid: str | None = None, status: str | None = None):
        filters = []
        if tenant_uuid:
            filters.append(MarketplaceListing.tenant_uuid == tenant_uuid)
        if status:
            filters.append(MarketplaceListing.status == status)
        return self.list(MarketplaceListing, filters)

    def get_listing(self, listing_id: str):
        return self.get_by_id(MarketplaceListing, listing_id)

    def create_listing(self, entity: MarketplaceListing):
        return self.add(entity)

    def update_listing(self, listing_id: str, updates: dict):
        return self.update_by_id(MarketplaceListing, listing_id, updates)

    def create_listing_versions(self, entities: list[MarketplaceListingVersion]):
        return self.add_all(entities)

    def create_listing_assets(self, entities: list[MarketplaceListingAsset]):
        return self.add_all(entities)

    def create_pricing_plans(self, entities: list[MarketplacePricingPlan]):
        return self.add_all(entities)

    def create_plan_tiers(self, entities: list[MarketplacePlanTier]):
        return self.add_all(entities)

    def create_checklist_runs(self, entities: list[MarketplaceChecklistRun]):
        return self.add_all(entities)

    def create_checklist_items(self, entities: list[MarketplaceChecklistItem]):
        return self.add_all(entities)

    def list_listing_versions(self, listing_id: str | None = None):
        filters = []
        if listing_id:
            filters.append(MarketplaceListingVersion.listing_id == listing_id)
        return self.list(MarketplaceListingVersion, filters)

    def list_listing_assets(self, listing_id: str | None = None):
        filters = []
        if listing_id:
            filters.append(MarketplaceListingAsset.listing_id == listing_id)
        return self.list(MarketplaceListingAsset, filters)

    def list_pricing_plans(self, listing_id: str | None = None):
        filters = []
        if listing_id:
            filters.append(MarketplacePricingPlan.listing_id == listing_id)
        return self.list(MarketplacePricingPlan, filters)

    def list_plan_tiers(self, plan_id: str | None = None):
        filters = []
        if plan_id:
            filters.append(MarketplacePlanTier.plan_id == plan_id)
        return self.list(MarketplacePlanTier, filters)

    def list_checklist_runs(self, listing_id: str | None = None):
        filters = []
        if listing_id:
            filters.append(MarketplaceChecklistRun.listing_id == listing_id)
        return self.list(MarketplaceChecklistRun, filters)

    def list_checklist_items(self, run_id: str | None = None):
        filters = []
        if run_id:
            filters.append(MarketplaceChecklistItem.checklist_run_id == run_id)
        return self.list(MarketplaceChecklistItem, filters)

    def list_licenses(self, tenant_uuid: str | None = None):
        filters = []
        if tenant_uuid:
            filters.append(MarketplaceLicense.tenant_uuid == tenant_uuid)
        return self.list(MarketplaceLicense, filters)

    def list_license_events(self, license_id: str | None = None):
        filters = []
        if license_id:
            filters.append(MarketplaceLicenseEvent.license_id == license_id)
        return self.list(MarketplaceLicenseEvent, filters)

    def list_tax_transactions(self, tenant_uuid: str | None = None):
        filters = []
        if tenant_uuid:
            filters.append(MarketplaceTaxTransaction.tenant_uuid == tenant_uuid)
        return self.list(MarketplaceTaxTransaction, filters)

    def list_usage_envelopes(self, tenant_uuid: str | None = None):
        filters = []
        if tenant_uuid:
            filters.append(MarketplaceUsageEnvelope.tenant_uuid == tenant_uuid)
        return self.list(MarketplaceUsageEnvelope, filters)

    def list_usage_aggregates(self, tenant_uuid: str | None = None):
        filters = []
        if tenant_uuid:
            filters.append(MarketplaceUsageAggregate.tenant_uuid == tenant_uuid)
        return self.list(MarketplaceUsageAggregate, filters)

    def list_revenue_reports(self, tenant_uuid: str | None = None):
        filters = []
        if tenant_uuid:
            filters.append(MarketplaceRevenueReport.tenant_uuid == tenant_uuid)
        return self.list(MarketplaceRevenueReport, filters)

    def list_notifications(self, tenant_uuid: str | None = None):
        filters = []
        if tenant_uuid:
            filters.append(MarketplaceNotification.tenant_uuid == tenant_uuid)
        return self.list(MarketplaceNotification, filters)

    def create_usage_envelope(self, entity: MarketplaceUsageEnvelope):
        return self.add(entity)

    def create_license(self, entity: MarketplaceLicense):
        return self.add(entity)

    def update_license(self, license_id: str, updates: dict):
        return self.update_by_id(MarketplaceLicense, license_id, updates)

    def create_license_event(self, entity: MarketplaceLicenseEvent):
        return self.add(entity)

    def create(self, entity):
        return self.add(entity)
