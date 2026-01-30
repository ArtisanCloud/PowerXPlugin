from sqlalchemy import BigInteger, Column, DateTime, String, text
from sqlalchemy import UniqueConstraint

from app.entity.models.base import Base


class Permission(Base):
    __tablename__ = "iam_permissions"
    __table_args__ = (UniqueConstraint("resource", "action", name="idx_iam_permissions_resource_action"),)

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    resource = Column(String(128), nullable=False)
    action = Column(String(64), nullable=False)
    description = Column(String(255), nullable=True)
    created_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False, onupdate=text("now()"))
