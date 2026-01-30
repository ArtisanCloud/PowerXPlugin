from sqlalchemy import BigInteger, Column, DateTime, String, text
from sqlalchemy.dialects.postgresql import UUID

from app.entity.models.base import Base


class Tenant(Base):
    __tablename__ = "iam_tenants"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    uuid = Column(UUID(as_uuid=False), unique=True, nullable=False, index=True)
    key = Column(String(64), unique=True, nullable=False)
    name = Column(String(128), nullable=False)
    status = Column(String(32), nullable=False, server_default=text("'active'"))
    plan = Column(String(64), nullable=False, server_default=text("'free'"))
    created_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False, onupdate=text("now()"))
    deleted_at = Column(DateTime(timezone=True), nullable=True)
