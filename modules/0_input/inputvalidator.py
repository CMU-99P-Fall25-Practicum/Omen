#!/usr/bin/env python3

# Network Config Validator (Wi-Fi schema)
# Validates a JSON spec before handing it to the runner.

import sys, json, argparse, hashlib, re
from pathlib import Path
from typing import Literal, Optional, List, Dict, Tuple
from pydantic import BaseModel, Field, ValidationError, model_validator, field_validator

# ---------------- Pydantic Models ----------------

class Meta(BaseModel):
    backend: Literal["mininet", "mininet-wifi"]
    name: str = Field(min_length=1, max_length=64)
    duration_s: int = Field(gt=0)

class PropagationModel(BaseModel):
    model: Literal["logDistance"]
    exp: float = Field(gt=0)

class Nets(BaseModel):
    noise_th: float = Field(le=0)  # dBm threshold (e.g., -91)
    propagation_model: PropagationModel

_POSITION_RE = re.compile(r"^\s*(-?\d+(\.\d+)?)\s*,\s*(-?\d+(\.\d+)?)\s*,\s*(-?\d+(\.\d+)?)\s*$")

def _validate_position_str(v: str) -> str:
    if not isinstance(v, str) or not _POSITION_RE.match(v):
        raise ValueError("position must be 'x,y,z' with numeric components")
    return v

class AP(BaseModel):
    id: str = Field(min_length=1)
    mode: Literal["a", "b", "g", "n", "ac", "ax"]
    channel: int = Field(gt=0)
    ssid: str = Field(min_length=1, max_length=32)
    position: str

    @field_validator("position")
    @classmethod
    def _pos_ok(cls, v: str) -> str:
        return _validate_position_str(v)

class Station(BaseModel):
    id: str = Field(min_length=1)
    position: str

    @field_validator("position")
    @classmethod
    def _pos_ok(cls, v: str) -> str:
        return _validate_position_str(v)

class Topology(BaseModel):
    nets: Nets
    aps: List[AP] = Field(default_factory=list)
    stations: List[Station] = Field(default_factory=list)

# ----- Tests -----

class TestMove(BaseModel):
    name: str
    type: Literal["node movements"]
    node: str
    position: str

    @field_validator("position")
    @classmethod
    def _pos_ok(cls, v: str) -> str:
        return _validate_position_str(v)

class TestIw(BaseModel):
    name: str
    type: Literal["iw"]
    cmd: str  # should ideally contain {interface}

TestVariant = TestMove | TestIw

class Spec(BaseModel):
    schemaVersion: str
    meta: Meta
    topo: Topology
    tests: List[TestVariant]

# ---------------- Validate Semantics ----------------

def _spec_hash(spec_dict: dict) -> str:
    raw = json.dumps(spec_dict, sort_keys=True).encode()
    return hashlib.sha256(raw).hexdigest()[:12]

def _positions_tuple(pos: str) -> Tuple[float, float, float]:
    m = _POSITION_RE.match(pos)
    assert m
    return (float(m.group(1)), float(m.group(3)), float(m.group(5)))

def validate_semantics(spec: Spec) -> Dict[str, List[dict]]:
    errors: List[dict] = []
    warnings: List[dict] = []

    # Unique IDs across APs + Stations
    ap_ids = [a.id for a in spec.topo.aps]
    sta_ids = [s.id for s in spec.topo.stations]
    all_ids = ap_ids + sta_ids
    if len(all_ids) != len(set(all_ids)):
        errors.append({
            "loc": "topo.[aps|stations].id",
            "code": "duplicate_id",
            "msg": "duplicate node IDs across APs and Stations"
        })

    sta_set = set(sta_ids)

    # Basic plausibility checks
    for i, ap in enumerate(spec.topo.aps):
        try:
            _positions_tuple(ap.position)
        except Exception:
            errors.append({
                "loc": f"topo.aps[{i}].position",
                "code": "bad_position",
                "msg": "cannot parse 'x,y,z' as floats"
            })
        if ap.mode == "a" and ap.channel <= 0:
            errors.append({
                "loc": f"topo.aps[{i}].channel",
                "code": "bad_channel",
                "msg": "5GHz channel must be a positive integer (e.g., 36)"
            })

    for i, sta in enumerate(spec.topo.stations):
        try:
            _positions_tuple(sta.position)
        except Exception:
            errors.append({
                "loc": f"topo.stations[{i}].position",
                "code": "bad_position",
                "msg": "cannot parse 'x,y,z' as floats"
            })

    # Tests
    for i, t in enumerate(spec.tests):
        if isinstance(t, TestMove):
            if t.node not in sta_set:
                errors.append({
                    "loc": f"tests[{i}].node",
                    "code": "unknown_station",
                    "msg": f"'{t.node}' is not a known station id"
                })
        elif isinstance(t, TestIw):
            if "{interface}" not in t.cmd:
                warnings.append({
                    "loc": f"tests[{i}].cmd",
                    "code": "missing_placeholder",
                    "msg": "cmd does not include '{interface}' placeholder"
                })

    # Backend mismatch (Wi-Fi entities under plain mininet)
    if spec.meta.backend == "mininet" and (spec.topo.aps or spec.topo.stations):
        warnings.append({
            "loc": "meta.backend",
            "code": "backend_role_mismatch",
            "msg": "APs/stations present but backend='mininet'. Use 'mininet-wifi' for Wi-Fi behavior."
        })

    # Nets sanity
    if spec.topo.nets.noise_th > -30:
        warnings.append({
            "loc": "topo.nets.noise_th",
            "code": "suspicious_noise_th",
            "msg": f"noise_th {spec.topo.nets.noise_th} dBm is unusually high (less negative)"
        })

    return {"errors": errors, "warnings": warnings}

# ---------------- CLI ----------------

def parse_args():
    ap = argparse.ArgumentParser(description="Validate and normalize network spec (Wi-Fi schema)")
    ap.add_argument(
        "config",
        nargs="?",
        default="input.json",
        help="input config JSON (default: input.json next to this script)",
    )
    ap.add_argument("--emit-spec", help="path to write normalized spec.json")
    ap.add_argument("--include-spec", action="store_true",
                    help="include normalized spec in stdout summary")
    return ap.parse_args()

def _print_stderr(prefix: str, items: List[dict]):
    print(prefix, file=sys.stderr)
    for it in items:
        loc = it.get("loc", "?")
        code = it.get("code", "error")
        msg = it.get("msg", "")
        print(f" - {loc} [{code}]: {msg}", file=sys.stderr)

def main():
    args = parse_args()

    # Resolve config relative to this script's directory if not absolute
    script_dir = Path(__file__).resolve().parent
    cfg_path = Path(args.config)
    if not cfg_path.is_absolute():
        cfg_path = script_dir / cfg_path

    if not cfg_path.exists():
        err = f"Config file not found: {cfg_path}"
        out = {"ok": False, "errors": [{"loc": "config", "code": "not_found", "msg": err}], "warnings": []}
        print(json.dumps(out, indent=2))
        print(f"VALIDATION_ERROR: {err}", file=sys.stderr)
        sys.exit(1)

    # Load + structural validation
    try:
        with open(cfg_path) as f:
            data = json.load(f)
        spec = Spec(**data)
    except ValidationError as ve:
        out = {
            "ok": False,
            "errors": [{"loc": ".".join(map(str, e["loc"])), "code": "pydantic", "msg": e["msg"]}
                       for e in ve.errors()],
            "warnings": []
        }
        print(json.dumps(out, indent=2))
        _print_stderr("VALIDATION_ERROR: schema validation failed", out["errors"])
        sys.exit(1)
    except Exception as e:
        out = {"ok": False, "errors": [{"loc": "root", "code": "load_error", "msg": str(e)}], "warnings": []}
        print(json.dumps(out, indent=2))
        _print_stderr("VALIDATION_ERROR: load error", out["errors"])
        sys.exit(1)

    # Semantic checks
    res = validate_semantics(spec)
    ok = len(res["errors"]) == 0

    # Normalize + fingerprint
    norm = spec.model_dump()
    norm.setdefault("meta", {})["schema_hash"] = _spec_hash(norm)

    # Write normalized spec only if valid
    if ok and args.emit_spec:
        with open(args.emit_spec, "w") as f:
            json.dump(norm, f, indent=2)

    # stdout summary
    out = {"ok": ok, "errors": res["errors"], "warnings": res["warnings"]}
    if args.include_spec and ok:
        out["spec"] = norm
    print(json.dumps(out, indent=2))

    if not ok:
        _print_stderr("VALIDATION_ERROR: semantic validation failed", res["errors"])
        sys.exit(1)

    sys.exit(0)

if __name__ == "__main__":
    main()