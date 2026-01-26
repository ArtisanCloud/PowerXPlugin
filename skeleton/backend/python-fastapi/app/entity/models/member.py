from datetime import datetime

from sqlalchemy import Column, DateTime, String

from app.entity.models.base import Base


class Member(Base):
    __tablename__ = "members"

    id = Column(String, primary_key=True)
    tenant_uuid = Column(String, nullable=False)
    user_id = Column(String, nullable=False)
    username = Column(String, nullable=False)
    display_name = Column(String, nullable=True)
    status = Column(String, nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow)
