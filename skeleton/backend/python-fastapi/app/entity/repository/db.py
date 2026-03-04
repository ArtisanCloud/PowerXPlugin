from dataclasses import dataclass
from typing import Optional

from sqlalchemy import create_engine
from sqlalchemy.engine import Engine
from sqlalchemy.orm import sessionmaker


@dataclass
class DatabaseConfig:
    url: str
    echo: bool = False


class Database:
    def __init__(self, config: DatabaseConfig) -> None:
        self._engine: Engine = create_engine(config.url, echo=config.echo, future=True)
        self._session_factory = sessionmaker(bind=self._engine, autoflush=False, autocommit=False)

    @property
    def engine(self) -> Engine:
        return self._engine

    def session(self):
        return self._session_factory()


_db: Optional[Database] = None


def init_db(config: DatabaseConfig) -> Database:
    global _db
    _db = Database(config)
    return _db


def get_db() -> Database:
    if _db is None:
        raise RuntimeError("Database is not initialized")
    return _db
