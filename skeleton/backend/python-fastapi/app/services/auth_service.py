class AuthService:
    def login(self, payload: dict):
        return {"token_type": "Bearer", "access_token": "", "expires_in": 0}

    def register(self, payload: dict):
        return {"user": {}, "member": {}, "tenant": {}}

    def logout(self, payload: dict):
        return {"ok": True}

    def refresh(self, payload: dict):
        return {"token_type": "Bearer", "access_token": "", "expires_in": 0}

    def me(self):
        return {}

    def profile(self, payload: dict):
        return payload

    def change_password(self, payload: dict):
        return {"ok": True}

    def reset_password(self, payload: dict):
        return {"ok": True}

    def reset_password_confirm(self, payload: dict):
        return {"ok": True}

    def validate(self):
        return {"valid": True}

    def permissions(self):
        return []
