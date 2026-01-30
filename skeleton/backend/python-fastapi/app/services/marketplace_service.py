from app.entity.repository.marketplace_repository import MarketplaceRepository
from app.services._utils import to_list


class MarketplaceService:
    def __init__(self, repo: MarketplaceRepository | None = None) -> None:
        self._repo = repo or MarketplaceRepository()

    def list_listings(self, tenant_uuid: str | None = None, status: str | None = None) -> list:
        return to_list(self._repo.list_listings(tenant_uuid, status))

    def list_listing_assets(self, listing_id: str | None = None) -> list:
        return to_list(self._repo.list_listing_assets(listing_id))

    def list_pricing_plans(self, listing_id: str | None = None) -> list:
        return to_list(self._repo.list_pricing_plans(listing_id))

    def list_checklist_runs(self, listing_id: str | None = None) -> list:
        return to_list(self._repo.list_checklist_runs(listing_id))

    def list_licenses(self, tenant_uuid: str | None = None) -> list:
        return to_list(self._repo.list_licenses(tenant_uuid))

    def list_usage_envelopes(self, tenant_uuid: str | None = None) -> list:
        return to_list(self._repo.list_usage_envelopes(tenant_uuid))

    def list_usage_aggregates(self, tenant_uuid: str | None = None) -> list:
        return to_list(self._repo.list_usage_aggregates(tenant_uuid))

    def list_revenue_reports(self, tenant_uuid: str | None = None) -> list:
        return to_list(self._repo.list_revenue_reports(tenant_uuid))

    def list_notifications(self, tenant_uuid: str | None = None) -> list:
        return to_list(self._repo.list_notifications(tenant_uuid))
