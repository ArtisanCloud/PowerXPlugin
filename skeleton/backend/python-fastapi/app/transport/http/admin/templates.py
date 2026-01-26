from fastapi import APIRouter

from app.contracts.response import ok
from app.services.template_service import TemplateService

router = APIRouter(prefix="/admin")
service = TemplateService()


@router.get("/templates")
async def list_templates():
    return ok(service.list_templates({}))


@router.get("/templates/{template_id}")
async def get_template(template_id: str):
    return ok(service.get_template(template_id))


@router.post("/templates")
async def create_template(payload: dict):
    return ok(service.create_template(payload))


@router.put("/templates/{template_id}")
async def update_template(template_id: str, payload: dict):
    return ok(service.update_template(template_id, payload))


@router.delete("/templates/{template_id}")
async def delete_template(template_id: str):
    return ok(service.delete_template(template_id))
