from datetime import datetime

from sqlalchemy import Column, DateTime, Integer, String

from app.entity.models.base import Base


class RuntimeSession(Base):
    __tablename__ = "mcp_sessions"

    id = Column(String, primary_key=True)
    runtime_assignment_id = Column(String, nullable=False)
    tenant_uuid = Column(String, nullable=False)
    state = Column(String, nullable=False)
    jwt_id = Column(String, nullable=True)
    capabilities_hash = Column(String, nullable=True)
    missed_heartbeats = Column(Integer, nullable=False, default=0)
    last_ping_at = Column(DateTime, nullable=True)
    closed_at = Column(DateTime, nullable=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow)
