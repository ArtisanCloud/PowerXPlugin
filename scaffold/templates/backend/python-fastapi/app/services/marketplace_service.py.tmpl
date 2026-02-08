from __future__ import annotations

from datetime import datetime
from typing import Any
from uuid import uuid4

from app.entity.models import (
    MarketplaceChecklistItem,
    MarketplaceChecklistRun,
    MarketplaceLicense,
    MarketplaceLicenseEvent,
    MarketplaceListing,
    MarketplaceListingAsset,
    MarketplaceListingVersion,
    MarketplacePlanTier,
    MarketplacePricingPlan,
    MarketplaceUsageEnvelope,
)
from app.entity.repository.marketplace_repository import MarketplaceRepository
from app.services._utils import to_dict, to_list


def _now() -> datetime:
    return datetime.utcnow()


def _now_iso() -> str:
    return _now().isoformat() + "Z"


def _uuid(value: str | None) -> str:
    return value or uuid4().hex


class MarketplaceService:
    def __init__(self, repo: MarketplaceRepository | None = None) -> None:
        self._repo = repo or MarketplaceRepository()

    def list_listings(self, tenant_uuid: str | None = None, status: str | None = None) -> list:
        return to_list(self._repo.list_listings(tenant_uuid, status))

    def get_listing(self, listing_id: str) -> dict:
        return to_dict(self._repo.get_listing(listing_id))

    def create_listing(self, payload: dict) -> dict:
        listing = MarketplaceListing(
            id=payload.get("id"),
            tenant_uuid=payload.get("tenant_uuid") or payload.get("tenantUuid") or "tenant-demo",
            plugin_id=payload.get("plugin_id") or payload.get("pluginId") or "",
            vendor_id=payload.get("vendor_id") or payload.get("vendorId") or "",
            status=payload.get("status") or "draft",
            title=payload.get("title") or "",
            slug=payload.get("slug") or "",
            summary=payload.get("summary"),
            description=payload.get("description"),
            cover_asset_id=payload.get("cover_asset_id"),
            hero_video_asset_id=payload.get("hero_video_asset_id"),
            categories=payload.get("categories"),
            tags=payload.get("tags"),
            locale=payload.get("locale") or "en",
            version=payload.get("version"),
            ready_checklist_score=payload.get("ready_checklist_score") or 0,
            recommended_weight=payload.get("recommended_weight") or 0,
            reviewed_at=payload.get("reviewed_at"),
            reviewer_id=payload.get("reviewer_id"),
            audit_notes=payload.get("audit_notes"),
            branding_theme=payload.get("branding_theme"),
        )
        listing = self._repo.create_listing(listing)

        assets = []
        for asset in payload.get("assets") or []:
            assets.append(
                MarketplaceListingAsset(
                    id=asset.get("id"),
                    listing_id=listing.id,
                    tenant_uuid=listing.tenant_uuid,
                    asset_type=asset.get("asset_type") or asset.get("type") or "",
                    storage_uri=asset.get("storage_uri") or asset.get("uri") or "",
                    checksum=asset.get("checksum"),
                    is_primary=bool(asset.get("is_primary")),
                    locale=asset.get("locale") or listing.locale,
                    weight=asset.get("weight") or 0,
                    metadata_=asset.get("metadata"),
                )
            )
        if assets:
            self._repo.create_listing_assets(assets)

        plans = []
        tiers = []
        for plan in payload.get("pricing_plans") or []:
            plan_entity = MarketplacePricingPlan(
                id=plan.get("id"),
                listing_id=listing.id,
                tenant_uuid=listing.tenant_uuid,
                plan_code=plan.get("plan_code") or "",
                plan_type=plan.get("plan_type") or "",
                currency=plan.get("currency") or "",
                amount=plan.get("amount"),
                billing_period=plan.get("billing_period"),
                trial_period_days=plan.get("trial_days") or plan.get("trial_period_days"),
                quota_limit=plan.get("quota_limit"),
                overage_policy=plan.get("overage_policy"),
                feature_matrix=plan.get("feature_matrix"),
                is_default=bool(plan.get("is_default")),
                status=plan.get("status") or "active",
            )
            plans.append(plan_entity)
            for tier in plan.get("tiers") or []:
                tiers.append(
                    MarketplacePlanTier(
                        id=tier.get("id"),
                        plan_id=plan_entity.id,
                        tenant_uuid=listing.tenant_uuid,
                        metric=tier.get("metric") or "",
                        range_from=tier.get("range_from") or 0,
                        range_to=tier.get("range_to"),
                        unit_amount=tier.get("unit_amount") or 0,
                        unit_name=tier.get("unit_name"),
                    )
                )
        if plans:
            self._repo.create_pricing_plans(plans)
        if tiers:
            self._repo.create_plan_tiers(tiers)

        checklist = payload.get("checklist") or {}
        if checklist:
            run = MarketplaceChecklistRun(
                id=checklist.get("id"),
                listing_id=listing.id,
                tenant_uuid=listing.tenant_uuid,
                trigger_source=checklist.get("trigger_source") or "vendor",
                run_number=checklist.get("run_number") or 1,
                status=checklist.get("status") or "pending",
                started_at=_now(),
                completed_at=checklist.get("completed_at"),
                summary=checklist.get("summary"),
                ci_pipeline_id=checklist.get("ci_pipeline_id"),
            )
            run = self._repo.create_checklist_runs([run])[0]
            items = []
            for item in checklist.get("items") or []:
                items.append(
                    MarketplaceChecklistItem(
                        id=item.get("id"),
                        checklist_run_id=run.id,
                        tenant_uuid=listing.tenant_uuid,
                        code=item.get("code") or "",
                        description=item.get("description") or "",
                        result=item.get("result") or "",
                        evidence_uri=item.get("evidence_uri"),
                        notes=item.get("notes"),
                        auto_fix_link=item.get("auto_fix_link"),
                    )
                )
            if items:
                self._repo.create_checklist_items(items)

        return to_dict(listing)

    def update_listing(self, listing_id: str, payload: dict) -> dict:
        updates: dict[str, Any] = {}
        for field in (
            "title",
            "summary",
            "description",
            "categories",
            "tags",
            "branding_theme",
            "locale",
            "version",
            "status",
            "audit_notes",
            "reviewer_id",
            "reviewed_at",
            "published_at",
        ):
            if field in payload:
                updates[field] = payload.get(field)
        listing = self._repo.update_listing(listing_id, updates)
        return to_dict(listing)

    def update_listing_status(self, listing_id: str, status: str, payload: dict | None = None) -> dict:
        updates = {"status": status}
        if payload:
            if payload.get("reviewer_id"):
                updates["reviewer_id"] = payload.get("reviewer_id")
            if payload.get("reviewed_at"):
                updates["reviewed_at"] = payload.get("reviewed_at")
            if payload.get("audit_notes"):
                updates["audit_notes"] = payload.get("audit_notes")
        if status == "published":
            updates["published_at"] = payload.get("published_at") if payload else _now()
        listing = self._repo.update_listing(listing_id, updates)
        return to_dict(listing)

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

    def ingest_usage(self, payload: dict) -> dict:
        envelope = MarketplaceUsageEnvelope(
            id=payload.get("id"),
            tenant_uuid=payload.get("tenant_uuid") or payload.get("tenantUuid") or "tenant-demo",
            license_id=payload.get("license_id") or payload.get("licenseId") or "",
            plugin_id=payload.get("plugin_id") or payload.get("pluginId") or "",
            metrics=payload.get("metrics") or {},
            timestamp_start=payload.get("timestamp_start") or _now(),
            timestamp_end=payload.get("timestamp_end") or _now(),
            signature=payload.get("signature") or "",
            checksum=payload.get("checksum") or _uuid(None),
            ingest_status=payload.get("ingest_status") or "processed",
            ingested_at=_now(),
        )
        envelope = self._repo.create_usage_envelope(envelope)
        return to_dict(envelope)

    def list_usage_metrics(self, tenant_uuid: str, license_id: str) -> dict:
        aggregates = self._repo.list_usage_aggregates(tenant_uuid)
        return {"tenant_uuid": tenant_uuid, "license_id": license_id, "metrics": to_list(aggregates)}

    def create_license(self, payload: dict) -> dict:
        license_record = MarketplaceLicense(
            id=payload.get("id"),
            tenant_uuid=payload.get("tenant_uuid") or payload.get("tenantUuid") or "tenant-demo",
            listing_id=payload.get("listing_id") or payload.get("listingId") or "",
            plan_id=payload.get("plan_id") or payload.get("planId") or "",
            license_token=payload.get("license_token") or _uuid(None),
            status=payload.get("status") or "active",
            issued_at=payload.get("issued_at") or _now(),
            expires_at=payload.get("expires_at") or _now(),
            renewal_token=payload.get("renewal_token"),
            offline_until=payload.get("offline_until"),
            last_validated_at=payload.get("last_validated_at"),
            issued_by=payload.get("issued_by"),
            metadata_=payload.get("metadata"),
        )
        license_record = self._repo.create_license(license_record)
        return to_dict(license_record)

    def renew_license(self, license_id: str, payload: dict | None = None) -> dict:
        updates = {}
        if payload:
            if payload.get("expires_at"):
                updates["expires_at"] = payload.get("expires_at")
            if payload.get("status"):
                updates["status"] = payload.get("status")
        updates.setdefault("status", "active")
        license_record = self._repo.update_license(license_id, updates)
        event = MarketplaceLicenseEvent(
            id=None,
            tenant_uuid=license_record.tenant_uuid if license_record else "tenant-demo",
            license_id=license_id,
            event_type="renewed",
            event_payload=payload or {},
            emitted_at=_now(),
            actor_id=None,
            trace_id=None,
        )
        self._repo.create_license_event(event)
        return to_dict(license_record)

    def extend_offline(self, license_id: str, payload: dict | None = None) -> dict:
        updates = {}
        if payload and payload.get("offline_until"):
            updates["offline_until"] = payload.get("offline_until")
        license_record = self._repo.update_license(license_id, updates)
        event = MarketplaceLicenseEvent(
            id=None,
            tenant_uuid=license_record.tenant_uuid if license_record else "tenant-demo",
            license_id=license_id,
            event_type="offline_extend",
            event_payload=payload or {},
            emitted_at=_now(),
            actor_id=None,
            trace_id=None,
        )
        self._repo.create_license_event(event)
        return to_dict(license_record)
