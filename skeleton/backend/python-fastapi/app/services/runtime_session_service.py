class RuntimeSessionService:
    def register(self, payload: dict):
        return {"session_id": ""}

    def ack(self, session_id: str, payload: dict):
        return {"ok": True}

    def heartbeat(self, session_id: str, payload: dict):
        return {"ok": True}

    def close(self, session_id: str, payload: dict):
        return {"ok": True}

    def invoke(self, session_id: str, payload: dict):
        return {}
