from fastapi import APIRouter

from app.contracts.response import ok

router = APIRouter(prefix="/admin")


@router.get("/manifest")
async def manifest():
    return ok({})


@router.get("/rbac")
async def rbac():
    return ok({})
