from app.entity.models.base import BaseModel


class ToolGrantRevocation(BaseModel):
    __tablename__ = "tool_grant_revocations"


class ToolGrantUsageEvent(BaseModel):
    __tablename__ = "tool_grant_usage_events"
