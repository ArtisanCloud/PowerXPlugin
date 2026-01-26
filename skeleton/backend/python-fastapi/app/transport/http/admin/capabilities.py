from fastapi import APIRouter

from app.contracts.response import ok
from app.services.capability_service import CapabilityService

router = APIRouter(prefix="/admin")
service = CapabilityService()


@router.get("/capabilities")
async def list_capabilities():
    return ok(service.list_capabilities())


@router.get("/capabilities/register/template")
async def register_template():
    return ok(service.register_template())


@router.post("/capabilities/register")
async def register(payload: dict):
    return ok(service.register(payload))


@router.post("/capabilities/register/validate")
async def validate(payload: dict):
    return ok(service.validate(payload))


@router.get("/capabilities/lifecycle/template")
async def lifecycle_template():
    return ok(service.lifecycle_template())


@router.get("/capabilities/lifecycle")
async def list_lifecycle():
    return ok(service.list_lifecycle())


@router.post("/capabilities/lifecycle")
async def create_lifecycle(payload: dict):
    return ok(service.create_lifecycle(payload))


@router.post("/capabilities/lifecycle/{plan_id}/status")
async def update_lifecycle_status(plan_id: str, payload: dict):
    return ok(service.update_lifecycle_status(plan_id, payload))


@router.get("/capabilities/exposure/template")
async def exposure_template():
    return ok(service.exposure_template())


@router.get("/capabilities/exposure/{capability_id}")
async def exposure_detail(capability_id: str):
    return ok(service.exposure_detail(capability_id))


@router.put("/capabilities/exposure/{capability_id}")
async def update_exposure(capability_id: str, payload: dict):
    return ok(service.update_exposure(capability_id, payload))


@router.get("/capabilities/quotas/{capability_id}")
async def list_quotas(capability_id: str):
    return ok(service.list_quotas(capability_id))


@router.post("/capabilities/quotas/{capability_id}")
async def update_quotas(capability_id: str, payload: dict):
    return ok(service.update_quotas(capability_id, payload))
