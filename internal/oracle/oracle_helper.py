"""pyhotlint oracle helper.

Spawned by internal/oracle/subprocess.go as a Python subprocess
attached to the project's venv. Speaks newline-delimited JSON over
stdin/stdout. The protocol is intentionally minimal:

  > stdin   {"id": 1, "method": "subclasses_nn_module", "params": {"qualname": "torchvision.models.resnet.ResNet"}}
  < stdout  {"id": 1, "result": {"known": true, "value": "yes"}}

On startup the helper prints `{"ready": true}` so the Go side knows
the interpreter and JSON loop are alive.

This MVP supports two methods:

  device_of(expr)             -> always Unknown (real impl needs symbol
                                 resolution against user runtime; that
                                 is the next iteration).
  subclasses_nn_module(qual)  -> imports the module and checks
                                 issubclass against torch.nn.Module.
                                 Returns Unknown if torch is missing,
                                 the module fails to import, or the
                                 class is not found.
"""

from __future__ import annotations

import importlib
import json
import sys
import traceback


def _result(known: bool, value: str = "") -> dict:
    return {"known": known, "value": value}


def _device_of(_params: dict) -> dict:
    # Symbol resolution against the user's runtime is non-trivial and
    # not implemented in this MVP — return Unknown.
    return _result(False)


def _subclasses_nn_module(params: dict) -> dict:
    qual = params.get("qualname", "")
    if not isinstance(qual, str) or "." not in qual:
        return _result(False)
    try:
        import torch.nn as nn  # noqa: F401  # may not be installed in the project's venv
    except Exception:
        return _result(False)
    mod_path, _, cls_name = qual.rpartition(".")
    try:
        mod = importlib.import_module(mod_path)
        cls = getattr(mod, cls_name, None)
        if cls is None:
            return _result(False)
        from torch import nn as _nn
        return _result(True, "yes" if issubclass(cls, _nn.Module) else "no")
    except Exception:
        return _result(False)


_DISPATCH = {
    "device_of": _device_of,
    "subclasses_nn_module": _subclasses_nn_module,
}


def _serve() -> None:
    sys.stdout.write(json.dumps({"ready": True}) + "\n")
    sys.stdout.flush()
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        try:
            req = json.loads(line)
        except json.JSONDecodeError as exc:
            sys.stdout.write(json.dumps({"id": 0, "error": f"bad json: {exc}"}) + "\n")
            sys.stdout.flush()
            continue
        rid = req.get("id", 0)
        method = req.get("method", "")
        handler = _DISPATCH.get(method)
        if handler is None:
            sys.stdout.write(json.dumps({"id": rid, "error": f"unknown method {method!r}"}) + "\n")
            sys.stdout.flush()
            continue
        try:
            res = handler(req.get("params", {}))
        except Exception:
            traceback.print_exc(file=sys.stderr)
            sys.stdout.write(json.dumps({"id": rid, "result": _result(False)}) + "\n")
            sys.stdout.flush()
            continue
        sys.stdout.write(json.dumps({"id": rid, "result": res}) + "\n")
        sys.stdout.flush()


if __name__ == "__main__":
    _serve()
