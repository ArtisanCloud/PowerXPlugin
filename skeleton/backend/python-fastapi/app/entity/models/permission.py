from datetime import datetime

from sqlalchemy import Column, DateTime, String

from app.entity.models.base import Base


class Permission(Base):
    __tablename__ = "permissions"

    id = Column(String, primary_key=True)
    plugin = Column(String, nullable=False)
    resource = Column(String, nullable=False)
    action = Column(String, nullable=False)
    effect = Column(String, nullable=False)
    status = Column(String, nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow)
