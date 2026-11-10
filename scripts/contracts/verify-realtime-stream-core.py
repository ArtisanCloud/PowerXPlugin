#!/usr/bin/env python3
"""Exercise the local SDK against an isolated Core HTTP/auth/realtime fixture."""

import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("core_backend", type=Path)
    args = parser.parse_args()
    core = args.core_backend.resolve()
    root = Path(__file__).resolve().parents[2]
    sdk = root / "framework/backend/go"
    module = "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go"
    fixture = root / "scripts/contracts/testdata/realtime-stream-core_test.go"
    package = "internal/transport/http/openapi/capability_registry"
    env = dict(os.environ, GOWORK="off")
    with tempfile.TemporaryDirectory(prefix="powerx-realtime-core-") as directory:
        temporary = Path(directory)
        modfile = temporary / "go.mod"
        shutil.copy2(core / "go.mod", modfile)
        shutil.copy2(core / "go.sum", temporary / "go.sum")
        # Resolve Core's relative replacements against Core, not the temporary dir.
        config = json.loads(subprocess.check_output(
            ["go", "mod", "edit", "-json"], cwd=core, env=env
        ))
        edits = [f"-require={module}@v0.0.0", f"-replace={module}={sdk}"]
        for replacement in config.get("Replace", []):
            old, new = replacement["Old"], replacement["New"]
            if not new.get("Version") and not Path(new["Path"]).is_absolute():
                source = old["Path"]
                if old.get("Version"):
                    source += "@" + old["Version"]
                edits.append(f"-replace={source}={(core / new['Path']).resolve()}")
        subprocess.run(["go", "mod", "edit", f"-modfile={modfile}", *edits],
                       cwd=core, env=env, check=True)
        overlay = temporary / "overlay.json"
        overlay.write_text(json.dumps({"Replace": {
            str(core / package / "framework_realtime_interop_test.go"): str(fixture)
        }}))
        command = ["go", "test", "-mod=mod", f"-modfile={modfile}",
                   f"-overlay={overlay}", "-race", "-count=1", "-timeout=90s", "-v",
                   "./" + package, "-run", "^TestFrameworkRealtimeCredentialTenantInterop$"]
        print("CORE_INTEROP_COMMAND", " ".join(command), flush=True)
        subprocess.run(command, cwd=core, env=env, check=True)


if __name__ == "__main__":
    main()
