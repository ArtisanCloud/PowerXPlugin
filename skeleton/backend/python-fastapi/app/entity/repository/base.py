from __future__ import annotations

from typing import Iterable, Type

from sqlalchemy import select, update, delete

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

    def add_all(self, entities: list):
        db = self._session()
        try:
            db.add_all(entities)
            db.commit()
            for entity in entities:
                db.refresh(entity)
            return entities
        finally:
            db.close()

    def update_by_id(self, model: Type, entity_id, updates: dict):
        if not updates:
            return self.get_by_id(model, entity_id)
        db = self._session()
        try:
            db.execute(update(model).where(model.id == entity_id).values(**updates))
            db.commit()
            return db.execute(select(model).where(model.id == entity_id)).scalar_one_or_none()
        finally:
            db.close()

    def delete_by_id(self, model: Type, entity_id) -> None:
        db = self._session()
        try:
            db.execute(delete(model).where(model.id == entity_id))
            db.commit()
        finally:
            db.close()
