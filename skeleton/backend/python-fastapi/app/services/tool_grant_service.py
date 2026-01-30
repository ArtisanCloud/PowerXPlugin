from app.entity.repository.tool_grant_repository import ToolGrantRepository
from app.services._utils import to_list


class ToolGrantService:
    def __init__(self, repo: ToolGrantRepository | None = None) -> None:
        self._repo = repo or ToolGrantRepository()

    def list_revocations(self, tenant_uuid: str | None = None) -> list:
        return to_list(self._repo.list_revocations(tenant_uuid))

    def list_usage_events(self, tenant_uuid: str | None = None) -> list:
        return to_list(self._repo.list_usage_events(tenant_uuid))
