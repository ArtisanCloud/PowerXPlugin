from sqlalchemy import BigInteger, Column, DateTime, text
from sqlalchemy.dialects.postgresql import UUID
from sqlalchemy.orm import declarative_base

Base = declarative_base()


class BaseModel(Base):
    __abstract__ = True

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    created_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False, onupdate=text("now()"))
    deleted_at = Column(DateTime(timezone=True), nullable=True)


class BaseNoTenantModel(Base):
    __abstract__ = True

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    created_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False, onupdate=text("now()"))
    deleted_at = Column(DateTime(timezone=True), nullable=True)
