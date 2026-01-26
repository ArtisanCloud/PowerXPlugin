from datetime import datetime

from sqlalchemy import Column, DateTime, Integer, String

from app.entity.models.base import Base


class Department(Base):
    __tablename__ = "departments"

    id = Column(String, primary_key=True)
    tenant_uuid = Column(String, nullable=False)
    name = Column(String, nullable=False)
    code = Column(String, nullable=False)
    parent_id = Column(Integer, nullable=True)
    description = Column(String, nullable=True)
    sort_order = Column(Integer, nullable=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow)
