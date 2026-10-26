#!/usr/bin/env python3
"""Run regression RLS checks on a temporary, agent-owned PostgreSQL server."""
import getpass
import json
import os
from pathlib import Path
import shutil
import socket
import subprocess
import tempfile

root = Path(__file__).resolve().parents[2]
for name in ('initdb', 'pg_ctl'):
    if not shutil.which(name):
        raise SystemExit('REGRESSION_POSTGRES_TOOLCHAIN_REQUIRED')
with tempfile.TemporaryDirectory(prefix='powerx-regression-postgres-') as work:
    directory = Path(work)
    data = directory / 'data'
    subprocess.run(['initdb', '-D', str(data), '-A', 'trust', '-U', getpass.getuser(), '--no-locale', '--encoding=UTF8'], check=True, stdout=subprocess.DEVNULL)
    with socket.socket() as reservation:
        reservation.bind(('127.0.0.1', 0))
        port = reservation.getsockname()[1]
    started = False
    try:
        command = ['pg_ctl', '-D', str(data), '-l', str(directory / 'server.log'), '-o', f'-h 127.0.0.1 -p {port} -k {directory}', '-w', 'start']
        subprocess.run(command, check=True, stdout=subprocess.DEVNULL)
        started = True
        pid = int((data / 'postmaster.pid').read_text().splitlines()[0])
        children = subprocess.run(['pgrep', '-P', str(pid)], text=True, capture_output=True).stdout.split()
        print(json.dumps({'parent_pid': os.getpid(), 'server_pid': pid, 'child_pids': children, 'port': port, 'start_command': command}), flush=True)
        env = dict(os.environ, REGRESSION_POSTGRES_TEST_DSN=f'host=127.0.0.1 port={port} user={getpass.getuser()} dbname=postgres sslmode=disable')
        result = subprocess.run(['go', 'test', './cmd/database/migrate', '-run', '^TestRegressionPostgresRLS$', '-count=1'], cwd=root / 'skeleton/backend/go-gin', env=env)
        if result.returncode:
            raise SystemExit(result.returncode)
    finally:
        if started:
            subprocess.run(['pg_ctl', '-D', str(data), '-m', 'fast', '-w', 'stop'], check=True, stdout=subprocess.DEVNULL)
            with socket.socket() as probe:
                released = probe.connect_ex(('127.0.0.1', port)) != 0
            print(json.dumps({'port': port, 'released': released}), flush=True)
            if not released:
                raise SystemExit('REGRESSION_TEMP_SERVER_NOT_RELEASED')
