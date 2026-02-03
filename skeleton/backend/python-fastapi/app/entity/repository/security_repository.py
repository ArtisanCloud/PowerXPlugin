from sqlalchemy import select

from app.entity.models import (
    SecurityAdvisoryDistribution,
    SecurityAuditReport,
    SecurityBaselineChecklist,
    SecurityVulnerabilityAdvisory,
)
from app.entity.repository.base import BaseRepository


class SecurityRepository(BaseRepository):
    def list_baselines(self):
        return self.list(SecurityBaselineChecklist)

    def get_baseline(self, baseline_id: str):
        return self.get_by_id(SecurityBaselineChecklist, baseline_id)

    def list_audit_reports(self, limit: int = 0):
        db = self._session()
        try:
            stmt = select(SecurityAuditReport).order_by(SecurityAuditReport.created_at.desc())
            if limit and limit > 0:
                stmt = stmt.limit(limit)
            return db.execute(stmt).scalars().all()
        finally:
            db.close()

    def get_audit_report(self, report_id: str):
        return self.get_by_id(SecurityAuditReport, report_id)

    def list_advisories(self, severity: list[str] | None = None, status: list[str] | None = None, limit: int = 0):
        db = self._session()
        try:
            stmt = select(SecurityVulnerabilityAdvisory)
            if severity:
                stmt = stmt.where(SecurityVulnerabilityAdvisory.severity.in_(severity))
            if status:
                stmt = stmt.where(SecurityVulnerabilityAdvisory.status.in_(status))
            stmt = stmt.order_by(SecurityVulnerabilityAdvisory.created_at.desc())
            if limit and limit > 0:
                stmt = stmt.limit(limit)
            return db.execute(stmt).scalars().all()
        finally:
            db.close()

    def get_advisory(self, advisory_id: str):
        return self.get_by_id(SecurityVulnerabilityAdvisory, advisory_id)

    def create_advisory(self, entity: SecurityVulnerabilityAdvisory):
        return self.add(entity)

    def update_advisory(self, advisory_id: str, updates: dict):
        return self.update_by_id(SecurityVulnerabilityAdvisory, advisory_id, updates)

    def list_distributions(self, tenant_uuid: str | None = None):
        filters = []
        if tenant_uuid:
            filters.append(SecurityAdvisoryDistribution.tenant_uuid == tenant_uuid)
        return self.list(SecurityAdvisoryDistribution, filters)

    def create_distribution(self, entity: SecurityAdvisoryDistribution):
        return self.add(entity)
