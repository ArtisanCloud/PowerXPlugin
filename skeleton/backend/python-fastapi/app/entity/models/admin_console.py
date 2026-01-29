from app.entity.models.base import BaseNoTenantModel


class AdminConsoleAuditEvent(BaseNoTenantModel):
    __tablename__ = "admin_console_audit_events"


class AdminConsoleConfigChange(BaseNoTenantModel):
    __tablename__ = "admin_console_config_changes"


class AdminConsoleJobRun(BaseNoTenantModel):
    __tablename__ = "admin_console_job_runs"
