from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime, timezone, timedelta
from typing import Any, Iterable
from uuid import uuid4

from app.entity.models import (
    OperationsIncident,
    OperationsIncidentChecklist,
    OperationsIncidentUpdate,
    OperationsReadinessChecklistItem,
    OperationsSLAProfile,
    OperationsSupportChannel,
)
from app.entity.repository.operations_repository import OperationsRepository
from app.services._utils import to_dict, to_list


def _now() -> datetime:
    return datetime.now(tz=timezone.utc)


def _as_datetime(value: Any) -> datetime | None:
    if isinstance(value, datetime):
        return value
    if isinstance(value, str):
        raw = value.strip()
        if not raw:
            return None
        if raw.endswith("Z"):
            raw = raw[:-1] + "+00:00"
        try:
            return datetime.fromisoformat(raw)
        except ValueError:
            return None
    return None


def _normalize_list(values: Iterable[str] | None) -> list[str]:
    if not values:
        return []
    out: list[str] = []
    seen: set[str] = set()
    for val in values:
        clean = (val or "").strip().lower()
        if not clean or clean in seen:
            continue
        out.append(clean)
        seen.add(clean)
    return out


IncidentSeverityCadence = {
    "sev0": timedelta(minutes=10),
    "sev1": timedelta(minutes=15),
    "sev2": timedelta(minutes=30),
    "sev3": timedelta(minutes=60),
    "sev4": timedelta(hours=24),
}


VALID_SEVERITIES = set(IncidentSeverityCadence.keys())
VALID_STATUSES = [
    "detected",
    "acknowledged",
    "mitigated",
    "monitoring",
    "resolved",
    "closed",
]


def _clamp_percentage(value: float) -> float:
    if value < 0:
        return 0.0
    if value > 100:
        return 100.0
    return round(value, 2)


def _clamp_hours(value: float) -> float:
    if value < 0:
        return 0.0
    return round(value, 2)


def _compute_sla_score(profile: OperationsSLAProfile) -> float:
    support_component = 0.0
    if profile.support_frt_actual_hours and profile.support_frt_actual_hours > 0:
        ratio = profile.support_frt_target_hours / profile.support_frt_actual_hours * 100
        support_component = min(100.0, ratio)
    uptime = _clamp_percentage(profile.uptime_actual or 0)
    reliability = _clamp_percentage(profile.success_actual_pct or 0)
    score = 0.4 * uptime + 0.3 * support_component + 0.3 * reliability
    return round(score, 2)


@dataclass
class ReadinessItemTemplate:
    key: str
    description: str
    blocking: bool
    owner_role: str


READINESS_BLUEPRINT: dict[str, list[ReadinessItemTemplate]] = {
    "support_ready": [
        ReadinessItemTemplate(
            key="support_channels_configured",
            description="Support channels (Marketplace ticket, vendor email, emergency hotline) configured and verified",
            blocking=True,
            owner_role="agent",
        ),
        ReadinessItemTemplate(
            key="knowledge_base_published",
            description="README/FAQ/Troubleshooting/Support Policy published to documentation hub",
            blocking=False,
            owner_role="operations",
        ),
    ],
    "incident_ready": [
        ReadinessItemTemplate(
            key="sev_matrix_defined",
            description="SEV-0~SEV-4 matrix and response windows approved",
            blocking=True,
            owner_role="manager",
        ),
        ReadinessItemTemplate(
            key="communication_channels_tested",
            description="Support Hub, Hotline, security@powerx.io, status page notifications tested end-to-end",
            blocking=True,
            owner_role="liaison",
        ),
    ],
    "sla_ready": [
        ReadinessItemTemplate(
            key="sla_targets_committed",
            description="Plan-level SLA/SLO/SLI targets documented and accepted by stakeholders",
            blocking=True,
            owner_role="manager",
        ),
        ReadinessItemTemplate(
            key="sla_sampling_cron_configured",
            description="Daily/Monthly/Quarterly SLA aggregation jobs scheduled",
            blocking=True,
            owner_role="operations",
        ),
    ],
}


class OperationsService:
    def __init__(self, repo: OperationsRepository | None = None) -> None:
        self._repo = repo or OperationsRepository()

    def list_support_channels(self, plugin_id: str | None = None, tenant_uuid: str | None = None) -> list:
        return to_list(self._repo.list_support_channels(plugin_id, tenant_uuid))

    def list_support_tickets(self, plugin_id: str | None = None, tenant_uuid: str | None = None) -> list:
        return to_list(self._repo.list_support_tickets(plugin_id, tenant_uuid))

    def list_support_ticket_events(self, ticket_id: str | None = None) -> list:
        return to_list(self._repo.list_support_ticket_events(ticket_id))

    def list_incidents(
        self,
        plugin_id: str | None = None,
        severities: list[str] | None = None,
        statuses: list[str] | None = None,
        labels: list[str] | None = None,
        from_dt: datetime | None = None,
        to_dt: datetime | None = None,
    ) -> list:
        return to_list(
            self._repo.list_incidents(plugin_id, severities, statuses, labels, from_dt, to_dt)
        )

    def list_incident_updates(self, incident_id: str | None = None) -> list:
        return to_list(self._repo.list_incident_updates(incident_id))

    def list_incident_checklist_items(self, incident_id: str | None = None) -> list:
        return to_list(self._repo.list_incident_checklist_items(incident_id))

    def list_readiness_items(self, plugin_id: str | None = None) -> list:
        return to_list(self._repo.list_readiness_items(plugin_id))

    def list_sla_profiles(self, plugin_id: str | None = None) -> list:
        return to_list(self._repo.list_sla_profiles(plugin_id))

    def list_sla_adjustments(self, plugin_id: str | None = None) -> list:
        return to_list(self._repo.list_sla_adjustments(plugin_id))

    def get_support_playbook(self, plugin_id: str, tenant_uuid: str | None = None) -> dict:
        channels = self._repo.list_support_channels(plugin_id, tenant_uuid)
        payload = {
            "channels": [],
            "knowledge_base": [],
            "readiness": [],
        }
        blocking_map = {item.key: item.blocking for item in READINESS_BLUEPRINT.get("support_ready", [])}
        for channel in channels:
            if channel.channel == "knowledge_base":
                label = ""
                url = ""
                meta = channel.metadata_ or {}
                if isinstance(meta, dict):
                    label = str(meta.get("label") or "")
                    url = str(meta.get("url") or "")
                if label or url:
                    payload["knowledge_base"].append({"label": label, "url": url})
                continue

            dto = {
                "id": channel.id,
                "channel": channel.channel,
                "address": "",
                "escalates": [],
                "service_window": channel.service_window or {},
                "metadata": channel.metadata_ or {},
                "enabled": bool(channel.is_enabled),
            }
            meta = channel.metadata_ or {}
            if isinstance(meta, dict) and meta.get("address"):
                dto["address"] = meta.get("address")
            path = channel.escalation_path or {}
            if isinstance(path, dict):
                levels = path.get("levels") or []
                dto["escalates"] = [str(v) for v in levels]
            payload["channels"].append(dto)

        readiness_items = self._repo.list_readiness_by_type(plugin_id, "support_ready")
        for item in readiness_items:
            payload["readiness"].append(
                {
                    "key": item.item_key,
                    "status": item.status,
                    "blocking": blocking_map.get(item.item_key, False),
                    "completed": (item.status or "").lower() == "completed",
                    "notes": item.notes or "",
                }
            )
        return payload

    def configure_support_playbook(self, plugin_id: str, payload: dict) -> dict:
        tenant_uuid = (payload.get("tenant_uuid") or "").strip() or None
        channels = payload.get("channels") or []
        knowledge = payload.get("knowledge_base") or []

        self._repo.delete_support_channels(plugin_id, tenant_uuid)

        created: list[OperationsSupportChannel] = []
        for raw in channels:
            if not isinstance(raw, dict):
                continue
            channel_name = str(raw.get("channel") or "").strip()
            if not channel_name:
                continue
            meta = dict(raw.get("metadata") or {})
            address = raw.get("address")
            if address:
                meta["address"] = address
            entry = OperationsSupportChannel(
                id=str(uuid4()),
                plugin_id=plugin_id,
                tenant_uuid=tenant_uuid,
                channel=channel_name,
                is_enabled=bool(raw.get("enabled", True)),
                service_window=raw.get("service_window") or {},
                escalation_path={"levels": raw.get("escalates") or []},
                metadata_=meta,
                created_at=_now(),
                updated_at=_now(),
            )
            created.append(entry)

        for raw in knowledge:
            if not isinstance(raw, dict):
                continue
            label = str(raw.get("label") or "").strip()
            url = str(raw.get("url") or "").strip()
            if not label and not url:
                continue
            entry = OperationsSupportChannel(
                id=str(uuid4()),
                plugin_id=plugin_id,
                tenant_uuid=tenant_uuid,
                channel="knowledge_base",
                is_enabled=True,
                metadata_={"label": label, "url": url},
                created_at=_now(),
                updated_at=_now(),
            )
            created.append(entry)

        for item in created:
            self._repo.create_support_channel(item)

        self._update_support_readiness(plugin_id, bool(channels), bool(knowledge))
        return self.get_support_playbook(plugin_id, tenant_uuid)

    def compute_support_metrics(self, plugin_id: str) -> dict:
        tickets = self._repo.list_support_tickets(plugin_id)
        frt_total = frt_count = mttr_total = mttr_count = 0.0
        csat_total = csat_count = 0.0
        resolved = total = 0
        for ticket in tickets:
            total += 1
            if ticket.first_response_at:
                frt_total += (ticket.first_response_at - ticket.created_at).total_seconds() / 3600
                frt_count += 1
            if ticket.resolved_at:
                mttr_total += (ticket.resolved_at - ticket.created_at).total_seconds() / 3600
                mttr_count += 1
                resolved += 1
            if ticket.csat_score is not None:
                csat_total += float(ticket.csat_score)
                csat_count += 1
        metrics = {
            "first_response_hours": frt_total / frt_count if frt_count else 0,
            "resolution_hours": mttr_total / mttr_count if mttr_count else 0,
            "csat_average": csat_total / csat_count if csat_count else 0,
            "resolution_rate": resolved / total if total else 0,
        }
        return metrics

    def create_incident(self, plugin_id: str, payload: dict) -> dict:
        severity = (payload.get("severity") or "").strip().lower()
        if severity not in VALID_SEVERITIES:
            raise ValueError(f"invalid severity {payload.get('severity')}")
        detection_source = (payload.get("detection_source") or "").strip()
        summary = (payload.get("summary") or "").strip()
        if not detection_source:
            raise ValueError("detection_source is required")
        if not summary:
            raise ValueError("summary is required")
        tenant_uuid = (payload.get("tenant_uuid") or "").strip() or None
        now = _now()
        next_update_at = _as_datetime(payload.get("next_update_at"))
        if next_update_at is None:
            cadence = IncidentSeverityCadence.get(severity)
            if cadence:
                next_update_at = now + cadence
        incident = OperationsIncident(
            id=str(uuid4()),
            plugin_id=plugin_id,
            tenant_uuid=tenant_uuid,
            severity=severity,
            status="detected",
            detection_source=detection_source,
            summary=summary,
            impact=payload.get("impact"),
            mitigation=payload.get("mitigation"),
            root_cause=payload.get("root_cause"),
            next_update_at=next_update_at,
            labels=payload.get("labels") or {},
            confidentiality=payload.get("confidentiality"),
            detected_at=_as_datetime(payload.get("detected_at")) or now,
            created_at=now,
            updated_at=now,
        )
        self._repo.create_incident(incident)
        self._set_readiness_status(plugin_id, "incident_ready", "sev_matrix_defined", True, "")
        return self.get_incident_response(plugin_id, incident.id)

    def update_incident(self, plugin_id: str, incident_id: str, payload: dict) -> dict:
        incident = self._repo.get_incident(plugin_id, incident_id)
        if not incident:
            return {}
        updates: dict[str, Any] = {}
        if "status" in payload and payload.get("status") is not None:
            status = str(payload.get("status") or "").strip().lower()
            if status not in VALID_STATUSES:
                raise ValueError(f"invalid status {payload.get('status')}")
            updates["status"] = status
            now = _now()
            if status == "acknowledged":
                updates["acknowledged_at"] = now
            elif status == "mitigated":
                updates["mitigated_at"] = now
            elif status == "resolved":
                updates["resolved_at"] = now
            elif status == "closed":
                updates["closed_at"] = now
        if "mitigation" in payload:
            updates["mitigation"] = payload.get("mitigation")
        if "root_cause" in payload:
            updates["root_cause"] = payload.get("root_cause")
        if "next_update_at" in payload:
            updates["next_update_at"] = _as_datetime(payload.get("next_update_at"))
        if "labels" in payload and payload.get("labels") is not None:
            labels = incident.labels or {}
            for key, value in (payload.get("labels") or {}).items():
                labels[key] = value
            updates["labels"] = labels
        if "confidentiality" in payload:
            updates["confidentiality"] = payload.get("confidentiality")
        updates["updated_at"] = _now()
        self._repo.update_incident(plugin_id, incident_id, updates)
        return self.get_incident_response(plugin_id, incident_id)

    def add_incident_timeline(self, plugin_id: str, incident_id: str, payload: dict) -> dict:
        entry_type = (payload.get("entry_type") or "").strip()
        message = (payload.get("message") or "").strip()
        if not entry_type:
            raise ValueError("entry_type is required")
        if not message:
            raise ValueError("message is required")
        entry = OperationsIncidentUpdate(
            id=str(uuid4()),
            incident_id=incident_id,
            entry_type=entry_type,
            message=message,
            stakeholder_channel=payload.get("stakeholder_channel"),
            author_role=payload.get("author_role"),
            posted_at=_as_datetime(payload.get("posted_at")) or _now(),
            metadata_=payload.get("metadata"),
            created_at=_now(),
        )
        self._repo.create_incident_update(entry)
        incident = self._repo.get_incident(plugin_id, incident_id)
        if incident and incident.severity in IncidentSeverityCadence:
            next_update_at = _now() + IncidentSeverityCadence[incident.severity]
            self._repo.update_incident(plugin_id, incident_id, {"next_update_at": next_update_at, "updated_at": _now()})
        if payload.get("stakeholder_channel"):
            self._set_readiness_status(plugin_id, "incident_ready", "communication_channels_tested", True, "")
        return to_dict(entry)

    def get_incident_response(self, plugin_id: str, incident_id: str) -> dict:
        incident = self._repo.get_incident(plugin_id, incident_id)
        if not incident:
            return {}
        timeline = self._repo.list_incident_updates(incident_id)
        checklist = self._repo.list_incident_checklist_items(incident_id)
        summary = self._build_readiness_summary(plugin_id)
        return {
            "incident": to_dict(incident),
            "timeline": to_list(timeline),
            "checklist": to_list(checklist),
            "checklist_status": summary,
        }

    def add_incident_checklist_item(self, incident_id: str, payload: dict) -> dict:
        item = OperationsIncidentChecklist(
            id=payload.get("id") or str(uuid4()),
            incident_id=incident_id,
            item_key=payload.get("item_key"),
            description=payload.get("description"),
            status=payload.get("status"),
            completed_at=_as_datetime(payload.get("completed_at")),
            created_at=_now(),
            updated_at=_now(),
        )
        item = self._repo.create_incident_checklist_item(item)
        return to_dict(item)

    def upsert_sla_profile(self, plugin_id: str, payload: dict) -> dict:
        plan_type = (payload.get("planType") or payload.get("plan_type") or "").strip()
        targets = payload.get("targets") or {}
        if not plan_type:
            raise ValueError("planType is required")
        existing = self._repo.get_sla_profile(plugin_id, plan_type)
        profile = OperationsSLAProfile(
            id=str(uuid4()),
            plugin_id=plugin_id,
            plan_type=plan_type,
            uptime_target=_clamp_percentage(float(targets.get("uptimeTarget") or targets.get("uptime_target") or 0)),
            uptime_actual=float(payload.get("uptimeActual") or payload.get("uptime_actual") or 0),
            response_target_ms=int(targets.get("responseTargetMs") or targets.get("response_target_ms") or 0),
            response_actual_ms=int(payload.get("responseActualMs") or payload.get("response_actual_ms") or 0),
            success_target_pct=_clamp_percentage(
                float(targets.get("successTargetPct") or targets.get("success_target_pct") or 0)
            ),
            success_actual_pct=float(payload.get("successActualPct") or payload.get("success_actual_pct") or 0),
            support_frt_target_hours=_clamp_hours(
                targets.get("supportFrtTargetHours") or targets.get("support_frt_target_hours") or 0
            ),
            support_frt_actual_hours=float(
                payload.get("supportFrtActualHours") or payload.get("support_frt_actual_hours") or 0
            ),
            sla_score=float(payload.get("slaScore") or payload.get("sla_score") or 0),
            incentive_applied_at=_as_datetime(payload.get("incentiveAppliedAt") or payload.get("incentive_applied_at")),
            penalty_applied_at=_as_datetime(payload.get("penaltyAppliedAt") or payload.get("penalty_applied_at")),
            notes=payload.get("notes") or "",
            computed_at=_as_datetime(payload.get("computedAt") or payload.get("computed_at")) or _now(),
            created_at=_as_datetime(payload.get("createdAt") or payload.get("created_at")) or _now(),
            updated_at=_now(),
        )
        if existing:
            profile.id = existing.id
            profile.created_at = existing.created_at
            profile.uptime_actual = existing.uptime_actual
            profile.response_actual_ms = existing.response_actual_ms
            profile.success_actual_pct = existing.success_actual_pct
            profile.support_frt_actual_hours = existing.support_frt_actual_hours
            profile.sla_score = existing.sla_score
            profile.incentive_applied_at = existing.incentive_applied_at
            profile.penalty_applied_at = existing.penalty_applied_at
            profile.notes = existing.notes or profile.notes
            profile.computed_at = existing.computed_at or profile.computed_at
        saved = self._repo.upsert_sla_profile(profile)
        self._update_sla_readiness(plugin_id, saved)
        return to_dict(saved) if isinstance(saved, OperationsSLAProfile) else {}

    def update_sla_actuals(self, plugin_id: str, plan_type: str, payload: dict) -> dict:
        plan = (plan_type or "").strip()
        existing = self._repo.get_sla_profile(plugin_id, plan)
        if not existing:
            raise ValueError("sla profile not found")
        actuals = payload.get("actuals") or {}
        uptime_actual = _clamp_percentage(float(actuals.get("uptimeActual") or actuals.get("uptime_actual") or 0))
        success_actual = _clamp_percentage(float(actuals.get("successActualPct") or actuals.get("success_actual_pct") or 0))
        support_frt_actual = _clamp_hours(
            float(actuals.get("supportFrtActualHours") or actuals.get("support_frt_actual_hours") or 0)
        )
        existing.uptime_actual = uptime_actual
        existing.response_actual_ms = int(actuals.get("responseActualMs") or actuals.get("response_actual_ms") or 0)
        existing.success_actual_pct = success_actual
        existing.support_frt_actual_hours = support_frt_actual
        existing.sla_score = _compute_sla_score(existing)
        updates = {
            "uptime_actual": existing.uptime_actual,
            "response_actual_ms": existing.response_actual_ms,
            "success_actual_pct": existing.success_actual_pct,
            "support_frt_actual_hours": existing.support_frt_actual_hours,
            "sla_score": existing.sla_score,
            "computed_at": _now(),
            "updated_at": _now(),
        }
        saved = self._repo.update_by_id(OperationsSLAProfile, existing.id, updates)
        return to_dict(saved) if isinstance(saved, OperationsSLAProfile) else {}

    def recompute_sla(self, plugin_id: str) -> list:
        profiles = self._repo.list_sla_profiles(plugin_id)
        for profile in profiles:
            if not isinstance(profile, OperationsSLAProfile):
                continue
            profile.sla_score = _compute_sla_score(profile)
            profile.computed_at = _now()
            profile.updated_at = _now()
            self._repo.upsert_sla_profile(profile)
        return to_list(profiles)

    def _update_support_readiness(self, plugin_id: str, has_channels: bool, has_docs: bool) -> None:
        items = self._ensure_readiness_items(plugin_id, "support_ready")
        for item in items:
            if item.item_key == "support_channels_configured":
                self._toggle_readiness_item(item, has_channels)
            elif item.item_key == "knowledge_base_published":
                self._toggle_readiness_item(item, has_docs)
            self._repo.upsert_readiness_item(item)

    def _update_sla_readiness(self, plugin_id: str, profile: OperationsSLAProfile | None) -> None:
        items = self._ensure_readiness_items(plugin_id, "sla_ready")
        for item in items:
            if item.item_key == "sla_targets_committed":
                ready = bool(profile and profile.uptime_target and profile.response_target_ms and profile.success_target_pct)
                self._toggle_readiness_item(item, ready)
            elif item.item_key == "sla_sampling_cron_configured":
                self._toggle_readiness_item(item, False)
            self._repo.upsert_readiness_item(item)

    def _set_readiness_status(self, plugin_id: str, checklist_type: str, key: str, completed: bool, notes: str) -> None:
        items = self._ensure_readiness_items(plugin_id, checklist_type)
        for item in items:
            if item.item_key != key:
                continue
            self._toggle_readiness_item(item, completed)
            if notes:
                item.notes = notes
            self._repo.upsert_readiness_item(item)
            break

    def _toggle_readiness_item(self, item: OperationsReadinessChecklistItem, completed: bool) -> None:
        if completed:
            item.status = "completed"
            item.completed_at = _now()
        else:
            item.status = "pending"
            item.completed_at = None
        item.updated_at = _now()

    def _ensure_readiness_items(self, plugin_id: str, checklist_type: str) -> list[OperationsReadinessChecklistItem]:
        existing = self._repo.list_readiness_by_type(plugin_id, checklist_type)
        by_key = {item.item_key: item for item in existing}
        results: list[OperationsReadinessChecklistItem] = []
        for template in READINESS_BLUEPRINT.get(checklist_type, []):
            item = by_key.get(template.key)
            if item is None:
                item = OperationsReadinessChecklistItem(
                    id=str(uuid4()),
                    plugin_id=plugin_id,
                    type=checklist_type,
                    item_key=template.key,
                    description=template.description,
                    status="pending",
                    owner_role=template.owner_role,
                    created_at=_now(),
                    updated_at=_now(),
                )
            else:
                item.description = template.description
                item.owner_role = template.owner_role
                item.updated_at = _now()
            self._repo.upsert_readiness_item(item)
            results.append(item)
        return results

    def _build_readiness_summary(self, plugin_id: str) -> dict:
        blocking_items: list[str] = []
        summary = {"support_ready": True, "incident_ready": True, "sla_ready": True, "blocking_items": []}
        for readiness_type, templates in READINESS_BLUEPRINT.items():
            items = self._ensure_readiness_items(plugin_id, readiness_type)
            by_key = {item.item_key: item for item in items}
            all_blocking_done = True
            for template in templates:
                item = by_key.get(template.key)
                if template.blocking and (item is None or (item.status or "").lower() != "completed"):
                    all_blocking_done = False
                    blocking_items.append(template.key)
            summary[readiness_type] = all_blocking_done
        summary["blocking_items"] = sorted(set(blocking_items))
        return summary
