from app.entity.repository.plugin_repository import PluginRepository
from app.services._utils import to_list


class PluginService:
    def __init__(self, repo: PluginRepository | None = None) -> None:
        self._repo = repo or PluginRepository()

    def list_tenant_ext(self) -> list:
        return to_list(self._repo.list_tenant_ext())

    def list_credentials(self, tenant_uuid: str | None = None) -> list:
        return to_list(self._repo.list_credentials(tenant_uuid))
