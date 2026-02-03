import logging
import uuid

from app.config.settings import get_settings
from app.entity.repository.plugin_repository import PluginRepository
from app.services._utils import to_dict, to_list
from app.shared.crypto import derive_key32, encrypt_aes_gcm


logger = logging.getLogger(__name__)


class PluginService:
    def __init__(self, repo: PluginRepository | None = None) -> None:
        self._repo = repo or PluginRepository()

    def list_tenant_ext(self) -> list:
        return to_list(self._repo.list_tenant_ext())

    def list_credentials(self, tenant_uuid: str | None = None) -> list:
        return to_list(self._repo.list_credentials(tenant_uuid))

    def upsert_credentials(self, tenant_uuid: str, plugin_id: str, client_id: str, client_secret: str) -> dict:
        tenant_uuid = (tenant_uuid or "").strip()
        if not tenant_uuid:
            raise ValueError("invalid tenant_uuid")
        try:
            uuid.UUID(tenant_uuid)
        except ValueError as exc:
            raise ValueError("invalid tenant_uuid") from exc
        if not plugin_id or not client_id or not client_secret:
            raise ValueError("missing required fields")

        settings = get_settings()
        key_material = (settings.server_secret_key or "").strip()
        if not key_material:
            if settings.dev_mode or settings.server_dev_mode:
                logger.warning("server.secret_key is empty; using dev-only fallback key")
                key_material = "dev-only-change-me"
            else:
                raise ValueError("server.secret_key not configured")
        key = derive_key32(key_material)
        aad = f"tenant:{tenant_uuid}|plugin:{plugin_id}|cid:{client_id}".encode("utf-8")
        ciphertext, iv = encrypt_aes_gcm(key, client_secret.encode("utf-8"), aad)

        record = self._repo.upsert_credential(
            tenant_uuid=tenant_uuid,
            plugin_id=plugin_id,
            client_id=client_id,
            secret_ciphertext=ciphertext,
            iv_nonce=iv,
            key_version=1,
        )
        return to_dict(record)
