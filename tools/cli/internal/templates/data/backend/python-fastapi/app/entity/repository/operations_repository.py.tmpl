from __future__ import annotations

from datetime import datetime
from typing import Iterable

from sqlalchemy import delete, select, update

from app.entity.models import (
    OperationsIncident,
    OperationsIncidentChecklist,
    OperationsIncidentUpdate,
    OperationsReadinessChecklistItem,
    OperationsSLAAdjustment,
    OperationsSLAProfile,
    OperationsSupportChannel,
    OperationsSupportTicket,
    OperationsSupportTicketEvent,
)
from app.entity.repository.base import BaseRepository


class OperationsRepository(BaseRepository):
    def list_support_channels(self, plugin_id: str | None = None, tenant_uuid: str | None = None):
        db = self._session()
        try:
            query = select(OperationsSupportChannel)
            if plugin_id:
                query = query.where(OperationsSupportChannel.plugin_id == plugin_id)
            if tenant_uuid:
                query = query.where(OperationsSupportChannel.tenant_uuid == tenant_uuid)
            else:
                query = query.where(OperationsSupportChannel.tenant_uuid.is_(None))
            query = query.order_by(OperationsSupportChannel.channel.asc())
            return db.execute(query).scalars().all()
        finally:
            db.close()

    def delete_support_channels(self, plugin_id: str, tenant_uuid: str | None = None) -> None:
        db = self._session()
        try:
            query = delete(OperationsSupportChannel).where(OperationsSupportChannel.plugin_id == plugin_id)
            if tenant_uuid:
                query = query.where(OperationsSupportChannel.tenant_uuid == tenant_uuid)
            else:
                query = query.where(OperationsSupportChannel.tenant_uuid.is_(None))
            db.execute(query)
            db.commit()
        finally:
            db.close()

    def list_support_tickets(self, plugin_id: str | None = None, tenant_uuid: str | None = None):
        filters: list[Iterable] = []
        if plugin_id:
            filters.append(OperationsSupportTicket.plugin_id == plugin_id)
        if tenant_uuid:
            filters.append(OperationsSupportTicket.tenant_uuid == tenant_uuid)
        return self.list(OperationsSupportTicket, filters)

    def list_support_ticket_events(self, ticket_id: str | None = None):
        db = self._session()
        try:
            query = select(OperationsSupportTicketEvent)
            if ticket_id:
                query = query.where(OperationsSupportTicketEvent.ticket_id == ticket_id)
            query = query.order_by(OperationsSupportTicketEvent.emitted_at.desc())
            return db.execute(query).scalars().all()
        finally:
            db.close()

    def list_incidents(
        self,
        plugin_id: str | None = None,
        severities: list[str] | None = None,
        statuses: list[str] | None = None,
        labels: list[str] | None = None,
        from_dt: datetime | None = None,
        to_dt: datetime | None = None,
    ):
        db = self._session()
        try:
            query = select(OperationsIncident)
            if plugin_id:
                query = query.where(OperationsIncident.plugin_id == plugin_id)
            if severities:
                query = query.where(OperationsIncident.severity.in_(severities))
            if statuses:
                query = query.where(OperationsIncident.status.in_(statuses))
            if labels:
                for label in labels:
                    if not label:
                        continue
                    query = query.where(OperationsIncident.labels.contains({label: True}))
            if from_dt:
                query = query.where(OperationsIncident.detected_at >= from_dt)
            if to_dt:
                query = query.where(OperationsIncident.detected_at <= to_dt)
            query = query.order_by(OperationsIncident.detected_at.desc())
            return db.execute(query).scalars().all()
        finally:
            db.close()

    def get_incident(self, plugin_id: str, incident_id: str):
        db = self._session()
        try:
            query = select(OperationsIncident).where(
                OperationsIncident.id == incident_id,
                OperationsIncident.plugin_id == plugin_id,
            )
            return db.execute(query).scalar_one_or_none()
        finally:
            db.close()

    def update_incident(self, plugin_id: str, incident_id: str, updates: dict):
        db = self._session()
        try:
            db.execute(
                update(OperationsIncident)
                .where(
                    OperationsIncident.id == incident_id,
                    OperationsIncident.plugin_id == plugin_id,
                )
                .values(**updates)
            )
            db.commit()
            return db.execute(
                select(OperationsIncident).where(
                    OperationsIncident.id == incident_id,
                    OperationsIncident.plugin_id == plugin_id,
                )
            ).scalar_one_or_none()
        finally:
            db.close()

    def list_incident_updates(self, incident_id: str | None = None):
        db = self._session()
        try:
            query = select(OperationsIncidentUpdate)
            if incident_id:
                query = query.where(OperationsIncidentUpdate.incident_id == incident_id)
            query = query.order_by(OperationsIncidentUpdate.posted_at.asc())
            return db.execute(query).scalars().all()
        finally:
            db.close()

    def list_incident_checklist_items(self, incident_id: str | None = None):
        db = self._session()
        try:
            query = select(OperationsIncidentChecklist)
            if incident_id:
                query = query.where(OperationsIncidentChecklist.incident_id == incident_id)
            query = query.order_by(OperationsIncidentChecklist.item_key.asc())
            return db.execute(query).scalars().all()
        finally:
            db.close()

    def list_readiness_items(self, plugin_id: str | None = None):
        filters = []
        if plugin_id:
            filters.append(OperationsReadinessChecklistItem.plugin_id == plugin_id)
        return self.list(OperationsReadinessChecklistItem, filters)

    def list_readiness_by_type(self, plugin_id: str, checklist_type: str):
        db = self._session()
        try:
            query = (
                select(OperationsReadinessChecklistItem)
                .where(
                    OperationsReadinessChecklistItem.plugin_id == plugin_id,
                    OperationsReadinessChecklistItem.type == checklist_type,
                )
                .order_by(OperationsReadinessChecklistItem.item_key.asc())
            )
            return db.execute(query).scalars().all()
        finally:
            db.close()

    def upsert_readiness_item(self, item: OperationsReadinessChecklistItem):
        if not item.id:
            return self.add(item)
        updates = {
            "plugin_id": item.plugin_id,
            "type": item.type,
            "item_key": item.item_key,
            "description": item.description,
            "status": item.status,
            "owner_role": item.owner_role,
            "due_date": item.due_date,
            "completed_at": item.completed_at,
            "notes": item.notes,
            "updated_at": item.updated_at,
        }
        return self.update_by_id(OperationsReadinessChecklistItem, item.id, updates)

    def list_sla_profiles(self, plugin_id: str | None = None):
        db = self._session()
        try:
            query = select(OperationsSLAProfile)
            if plugin_id:
                query = query.where(OperationsSLAProfile.plugin_id == plugin_id)
            query = query.order_by(OperationsSLAProfile.plan_type.asc())
            return db.execute(query).scalars().all()
        finally:
            db.close()

    def get_sla_profile(self, plugin_id: str, plan_type: str):
        db = self._session()
        try:
            query = select(OperationsSLAProfile).where(
                OperationsSLAProfile.plugin_id == plugin_id,
                OperationsSLAProfile.plan_type == plan_type,
            )
            return db.execute(query).scalar_one_or_none()
        finally:
            db.close()

    def upsert_sla_profile(self, profile: OperationsSLAProfile):
        if not profile:
            return None
        existing = None
        if profile.plugin_id and profile.plan_type:
            existing = self.get_sla_profile(profile.plugin_id, profile.plan_type)
        if existing is None:
            return self.add(profile)
        updates = {
            "uptime_target": profile.uptime_target,
            "uptime_actual": profile.uptime_actual,
            "response_target_ms": profile.response_target_ms,
            "response_actual_ms": profile.response_actual_ms,
            "success_target_pct": profile.success_target_pct,
            "success_actual_pct": profile.success_actual_pct,
            "support_frt_target_hours": profile.support_frt_target_hours,
            "support_frt_actual_hours": profile.support_frt_actual_hours,
            "sla_score": profile.sla_score,
            "incentive_applied_at": profile.incentive_applied_at,
            "penalty_applied_at": profile.penalty_applied_at,
            "notes": profile.notes,
            "computed_at": profile.computed_at,
            "updated_at": profile.updated_at,
        }
        return self.update_by_id(OperationsSLAProfile, existing.id, updates)

    def list_sla_adjustments(self, plugin_id: str | None = None):
        filters = []
        if plugin_id:
            filters.append(OperationsSLAAdjustment.plugin_id == plugin_id)
        return self.list(OperationsSLAAdjustment, filters)

    def create_support_channel(self, entity: OperationsSupportChannel):
        return self.add(entity)

    def create_support_ticket(self, entity: OperationsSupportTicket):
        return self.add(entity)

    def create_support_ticket_event(self, entity: OperationsSupportTicketEvent):
        return self.add(entity)

    def create_incident(self, entity: OperationsIncident):
        return self.add(entity)

    def create_incident_update(self, entity: OperationsIncidentUpdate):
        return self.add(entity)

    def create_incident_checklist_item(self, entity: OperationsIncidentChecklist):
        return self.add(entity)
