from app.entity.repository.customer_repository import CustomerRepository
from app.services._utils import to_list


class CustomerService:
    def __init__(self, repo: CustomerRepository | None = None) -> None:
        self._repo = repo or CustomerRepository()

    def list_accounts(self, tenant_uuid: str | None = None) -> list:
        return to_list(self._repo.list_accounts(tenant_uuid))
