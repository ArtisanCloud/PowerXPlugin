from app.entity.models.base import BaseModel


class SecurityBaselineChecklist(BaseModel):
    __tablename__ = "security_baseline_checklists"


class SecurityAuditReport(BaseModel):
    __tablename__ = "security_audit_reports"


class SecurityVulnerabilityAdvisory(BaseModel):
    __tablename__ = "security_vulnerability_advisories"


class SecurityAdvisoryDistribution(BaseModel):
    __tablename__ = "security_advisory_distributions"
