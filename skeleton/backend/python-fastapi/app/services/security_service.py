from app.entity.repository.security_repository import SecurityRepository
from app.services._utils import to_list


class SecurityService:
    def __init__(self, repo: SecurityRepository | None = None) -> None:
        self._repo = repo or SecurityRepository()

    def list_baselines(self) -> list:
        return to_list(self._repo.list_baselines())

    def list_audit_reports(self) -> list:
        return to_list(self._repo.list_audit_reports())

    def list_advisories(self) -> list:
        return to_list(self._repo.list_advisories())

    def list_distributions(self, tenant_uuid: str | None = None) -> list:
        return to_list(self._repo.list_distributions(tenant_uuid))
