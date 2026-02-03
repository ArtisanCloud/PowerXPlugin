from sqlalchemy import select

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

    def get_credential_by_tenant_plugin(self, tenant_uuid: str, plugin_id: str):
        db = self._session()
        try:
            return (
                db.execute(
                    select(PluginCredentials).where(
                        PluginCredentials.tenant_uuid == tenant_uuid,
                        PluginCredentials.plugin_id == plugin_id,
                    )
                )
                .scalars()
                .first()
            )
        finally:
            db.close()

    def upsert_credential(
        self,
        tenant_uuid: str,
        plugin_id: str,
        client_id: str,
        secret_ciphertext: bytes,
        iv_nonce: bytes,
        key_version: int = 1,
    ):
        db = self._session()
        try:
            record = (
                db.execute(
                    select(PluginCredentials).where(
                        PluginCredentials.tenant_uuid == tenant_uuid,
                        PluginCredentials.plugin_id == plugin_id,
                    )
                )
                .scalars()
                .first()
            )
            if record:
                record.client_id = client_id
                record.secret_ciphertext = secret_ciphertext
                record.iv_nonce = iv_nonce
                record.key_version = key_version
                db.commit()
                return record
            record = PluginCredentials(
                tenant_uuid=tenant_uuid,
                plugin_id=plugin_id,
                client_id=client_id,
                secret_ciphertext=secret_ciphertext,
                iv_nonce=iv_nonce,
                key_version=key_version,
            )
            db.add(record)
            db.commit()
            db.refresh(record)
            return record
        finally:
            db.close()
