from sqlalchemy import BigInteger, Column, DateTime, text

from app.entity.models.base import Base


class IAMMemberRole(Base):
    __tablename__ = "iam_member_roles"

    member_id = Column(BigInteger, primary_key=True)
    role_id = Column(BigInteger, primary_key=True)
    created_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
