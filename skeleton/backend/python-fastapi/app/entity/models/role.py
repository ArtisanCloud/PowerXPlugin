from datetime import datetime

from sqlalchemy import Column, DateTime, String

from app.entity.models.base import Base


class Role(Base):
    __tablename__ = "roles"

    id = Column(String, primary_key=True)
    tenant_uuid = Column(String, nullable=False)
    code = Column(String, nullable=False)
    name = Column(String, nullable=False)
    description = Column(String, nullable=True)
    scope_type = Column(String, nullable=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow)
