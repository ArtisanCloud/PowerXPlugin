class CapabilityService:
    def list_capabilities(self):
        return []

    def register_template(self):
        return {}

    def register(self, payload: dict):
        return {}

    def validate(self, payload: dict):
        return {"valid": True}

    def lifecycle_template(self):
        return {}

    def list_lifecycle(self):
        return []

    def create_lifecycle(self, payload: dict):
        return {}

    def update_lifecycle_status(self, plan_id: str, payload: dict):
        return {}

    def exposure_template(self):
        return {}

    def exposure_detail(self, capability_id: str):
        return {}

    def update_exposure(self, capability_id: str, payload: dict):
        return {}

    def list_quotas(self, capability_id: str):
        return []

    def update_quotas(self, capability_id: str, payload: dict):
        return {}
