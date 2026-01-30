from sqlalchemy import BigInteger, Column, String, text

from app.entity.models.base import BaseModel


class Department(BaseModel):
    __tablename__ = "iam_departments"

    name = Column(String(128), nullable=False, index=True)
    code = Column(String(64), nullable=False, index=True)
    parent_id = Column(BigInteger, nullable=True, index=True)
    description = Column(String(255), nullable=True)
    path = Column(String(512), nullable=False, server_default=text("''"), index=True)
    sort_order = Column(BigInteger, nullable=False, server_default=text("0"), index=True)
