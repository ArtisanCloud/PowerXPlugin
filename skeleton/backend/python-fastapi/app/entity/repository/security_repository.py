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

    def list_audit_reports(self):
        return self.list(SecurityAuditReport)

    def get_audit_report(self, report_id: str):
        return self.get_by_id(SecurityAuditReport, report_id)

    def list_advisories(self):
        return self.list(SecurityVulnerabilityAdvisory)

    def get_advisory(self, advisory_id: str):
        return self.get_by_id(SecurityVulnerabilityAdvisory, advisory_id)

    def list_distributions(self, tenant_uuid: str | None = None):
        filters = []
        if tenant_uuid:
            filters.append(SecurityAdvisoryDistribution.tenant_uuid == tenant_uuid)
        return self.list(SecurityAdvisoryDistribution, filters)

    def create(self, entity):
        return self.add(entity)
