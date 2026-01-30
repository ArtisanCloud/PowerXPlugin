from sqlalchemy import BigInteger, Column, DateTime, Integer, LargeBinary, String, UniqueConstraint, func, text
from sqlalchemy.dialects.postgresql import UUID

from app.entity.models.base import Base


class PluginCredentials(Base):
    __tablename__ = "plugin_credentials"
    __table_args__ = (
        UniqueConstraint("tenant_uuid", "plugin_id", name="uq_plugin_credentials_tenant_plugin"),
    )

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    plugin_id = Column(String(128), nullable=False)
    client_id = Column(String(255), nullable=False)
    secret_ciphertext = Column(LargeBinary, nullable=False)
    iv_nonce = Column(LargeBinary, nullable=False)
    key_version = Column(Integer, server_default=text("1"), nullable=False)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)
    deleted_at = Column(DateTime(timezone=True), nullable=True)
