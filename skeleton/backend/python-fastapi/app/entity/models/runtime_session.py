from datetime import datetime

from sqlalchemy import Column, DateTime, String

from app.entity.models.base import Base


class RuntimeSession(Base):
    __tablename__ = "runtime_sessions"

    id = Column(String, primary_key=True)
    session_id = Column(String, nullable=False)
    status = Column(String, nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow)
