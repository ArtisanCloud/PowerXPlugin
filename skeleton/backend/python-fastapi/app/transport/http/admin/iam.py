from fastapi import APIRouter

from app.contracts.response import ok
from app.services.iam_service import IAMService

router = APIRouter(prefix="/admin")
service = IAMService()


@router.get("/iam/tenants")
async def list_tenants():
    return ok(service.list_tenants({}))


@router.get("/iam/tenants/{tenant_id}")
async def get_tenant(tenant_id: str):
    return ok(service.get_tenant(tenant_id))


@router.post("/iam/tenants")
async def create_tenant(payload: dict):
    return ok(payload)


@router.patch("/iam/tenants/{tenant_id}")
async def update_tenant(tenant_id: str, payload: dict):
    return ok(payload)


@router.get("/iam/roles")
async def list_roles():
    return ok(service.list_roles({}))


@router.get("/iam/roles/{role_id}")
async def get_role(role_id: str):
    return ok(service.get_role(role_id))


@router.post("/iam/roles")
async def create_role(payload: dict):
    return ok(payload)


@router.patch("/iam/roles/{role_id}")
async def update_role(role_id: str, payload: dict):
    return ok(payload)


@router.delete("/iam/roles/{role_id}")
async def delete_role(role_id: str):
    return ok({"deleted": True})


@router.get("/iam/roles/{role_id}/permissions")
async def list_role_permissions(role_id: str):
    return ok([])


@router.put("/iam/roles/{role_id}/permissions")
async def update_role_permissions(role_id: str, payload: dict):
    return ok(payload)


@router.get("/iam/roles/{role_id}/members")
async def list_role_members(role_id: str):
    return ok([])


@router.post("/iam/roles/{role_id}/members")
async def add_role_members(role_id: str, payload: dict):
    return ok(payload)


@router.delete("/iam/roles/{role_id}/members")
async def remove_role_members(role_id: str, payload: dict):
    return ok({"deleted": True})


@router.get("/iam/permissions")
async def list_permissions():
    return ok(service.list_permissions())


@router.get("/iam/permissions/catalog")
async def list_permission_catalog():
    return ok({})


@router.post("/iam/permissions")
async def create_permission(payload: dict):
    return ok(payload)


@router.put("/iam/permissions/{permission_id}")
async def update_permission(permission_id: str, payload: dict):
    return ok(payload)


@router.delete("/iam/permissions/{permission_id}")
async def delete_permission(permission_id: str):
    return ok({"deleted": True})


@router.post("/iam/permissions/sync")
async def sync_permissions():
    return ok({"ok": True})


@router.get("/iam/departments")
async def list_departments():
    return ok(service.list_departments({}))


@router.get("/iam/departments/tree")
async def list_department_tree():
    return ok([])


@router.post("/iam/departments")
async def create_department(payload: dict):
    return ok(payload)


@router.patch("/iam/departments/{department_id}")
async def update_department(department_id: str, payload: dict):
    return ok(payload)


@router.delete("/iam/departments/{department_id}")
async def delete_department(department_id: str):
    return ok({"deleted": True})


@router.get("/iam/members")
async def list_members():
    return ok([])


@router.post("/iam/members")
async def create_member(payload: dict):
    return ok(payload)


@router.patch("/iam/members/{member_id}")
async def update_member(member_id: str, payload: dict):
    return ok(payload)


@router.post("/iam/members/import")
async def import_members(payload: dict):
    return ok(payload)
