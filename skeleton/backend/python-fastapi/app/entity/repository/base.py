from __future__ import annotations

from typing import Iterable, Type

from sqlalchemy import select

from app.entity.repository.db import get_db


class BaseRepository:
    def _session(self):
        return get_db().session()

    def list(self, model: Type, filters: Iterable | None = None, limit: int | None = None, offset: int | None = None):
        db = self._session()
        try:
            query = select(model)
            if filters:
                for condition in filters:
                    query = query.where(condition)
            if offset:
                query = query.offset(offset)
            if limit:
                query = query.limit(limit)
            return db.execute(query).scalars().all()
        finally:
            db.close()

    def get_by_id(self, model: Type, entity_id):
        db = self._session()
        try:
            return db.execute(select(model).where(model.id == entity_id)).scalar_one_or_none()
        finally:
            db.close()

    def add(self, entity):
        db = self._session()
        try:
            db.add(entity)
            db.commit()
            db.refresh(entity)
            return entity
        finally:
            db.close()
