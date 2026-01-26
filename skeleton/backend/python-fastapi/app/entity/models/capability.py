from datetime import datetime

from sqlalchemy import Column, DateTime, String

from app.entity.models.base import Base


class Capability(Base):
    __tablename__ = "capabilities"

    id = Column(String, primary_key=True)
    name = Column(String, nullable=False)
    status = Column(String, nullable=False)
    version = Column(String, nullable=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow)
