from app.entity.models import CustomerAccount
from app.entity.repository.base import BaseRepository


class CustomerRepository(BaseRepository):
    def list_accounts(self, tenant_uuid: str | None = None):
        filters = []
        if tenant_uuid:
            filters.append(CustomerAccount.tenant_uuid == tenant_uuid)
        return self.list(CustomerAccount, filters)

    def get_account(self, account_id: str):
        return self.get_by_id(CustomerAccount, account_id)

    def create(self, entity):
        return self.add(entity)
