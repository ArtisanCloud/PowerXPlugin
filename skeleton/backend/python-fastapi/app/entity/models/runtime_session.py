from sqlalchemy import Column, DateTime, Integer, Text, func, text
from sqlalchemy.dialects.postgresql import UUID

from app.entity.models.base import Base


class RuntimeSession(Base):
    __tablename__ = "mcp_sessions"

    id = Column(UUID(as_uuid=False), primary_key=True)
    runtime_assignment_id = Column(UUID(as_uuid=False), nullable=False)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False)
    state = Column(Text, nullable=False)
    jwt_id = Column(Text, nullable=True)
    capabilities_hash = Column(Text, nullable=True)
    missed_heartbeats = Column(Integer, server_default=text("0"), nullable=False)
    last_ping_at = Column(DateTime(timezone=True), nullable=True)
    closed_at = Column(DateTime(timezone=True), nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)
