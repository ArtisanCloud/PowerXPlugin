class TemplateService:
    def list_templates(self, params: dict):
        return []

    def get_template(self, template_id: str):
        return {}

    def create_template(self, payload: dict):
        return payload

    def update_template(self, template_id: str, payload: dict):
        return payload

    def delete_template(self, template_id: str):
        return {"ok": True}
