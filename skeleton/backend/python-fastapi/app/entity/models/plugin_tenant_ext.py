from sqlalchemy import BigInteger, Column, DateTime, SmallInteger, String, Text, func, text
from sqlalchemy.dialects.postgresql import JSONB

from app.entity.models.base import Base


class PluginTenantExt(Base):
    __tablename__ = "plugin_tenant_ext"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    status = Column(SmallInteger, server_default=text("0"), nullable=False)
    plan = Column(String(32), server_default=text("'free'"), nullable=False)
    flags = Column(JSONB, server_default=text("'{}'::jsonb"), nullable=False)
    config = Column(JSONB, server_default=text("'{}'::jsonb"), nullable=False)
    region = Column(String(32), nullable=True)
    expire_at = Column(DateTime(timezone=True), nullable=True)
    last_sync_at = Column(DateTime(timezone=True), nullable=True)
    last_error = Column(Text, nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)
    deleted_at = Column(DateTime(timezone=True), nullable=True)
