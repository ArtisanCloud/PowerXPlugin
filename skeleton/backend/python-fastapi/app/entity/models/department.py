from sqlalchemy import BigInteger, Column, String

from app.entity.models.base import BaseModel


class Department(BaseModel):
    __tablename__ = "iam_departments"

    name = Column(String, nullable=False)
    code = Column(String, nullable=False)
    parent_id = Column(BigInteger, nullable=True)
    description = Column(String, nullable=True)
    path = Column(String, nullable=False, default="")
    sort_order = Column(BigInteger, nullable=False, default=0)
