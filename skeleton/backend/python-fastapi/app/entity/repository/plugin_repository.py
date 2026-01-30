from app.entity.models import PluginCredentials, PluginTenantExt
from app.entity.repository.base import BaseRepository


class PluginRepository(BaseRepository):
    def list_tenant_ext(self):
        return self.list(PluginTenantExt)

    def get_tenant_ext(self, record_id: int):
        return self.get_by_id(PluginTenantExt, record_id)

    def list_credentials(self, tenant_uuid: str | None = None):
        filters = []
        if tenant_uuid:
            filters.append(PluginCredentials.tenant_uuid == tenant_uuid)
        return self.list(PluginCredentials, filters)

    def get_credential(self, credential_id: int):
        return self.get_by_id(PluginCredentials, credential_id)

    def create(self, entity):
        return self.add(entity)
