#!/usr/bin/env python3

"""
Network Config Validator (Wi-Fi/Mininet schema)

Validates a JSON spec before handing it to the runner.
Check include:
   • Schema validation via Pydantic (structure, types, bounds)
   • Semantic validation (duplicates, unknown nodes, etc.)
   • Sanity checks (position format, channels, thresholds)
   • Optional warnings for suspicious but non-fatal issues

Supports backends:
- mininet
- mininet wifi

Updated: Supports 'logNormalShadowing' propagation model,
adds 'timeframe' in tests, and optional top-level credentials.

"""

# Imports
import sys, json, argparse, hashlib, re
from pathlib import Path
from typing import Literal, Optional, List, Dict, Tuple, DefaultDict
from collections import defaultdict
from pydantic import BaseModel, Field, ValidationError, model_validator, field_validator

# ---------------- Pydantic Models: Schema Definitions ----------------

# Meta Section
class Meta(BaseModel): 
    backend: Literal["mininet", "mininet-wifi"]      # backend: which simulation backend to use
    name: str = Field(min_length=1, max_length=64)   
    duration_s: int = Field(gt=0)                    # duration_s: duration of run in seconds

# Propagation Model
class PropagationModel(BaseModel):
    # UPDATED: allow both models; require 's' only for logNormalShadowing
    model: Literal["logDistance", "logNormalShadowing"]    
    exp: float = Field(gt=0)                             # exp: path-loss experiment (>0)
    s: Optional[float] = None                            # stddev for log-normal shadowing (dB)

    @model_validator(mode="after")
    def _require_s_for_lns(self):
        # Ensure 's' is specified and positive if model = logNormalShadowing. 
        if self.model == "logNormalShadowing":
            if self.s is None or not (self.s > 0):
                raise ValueError("propagation_model.s must be provided and > 0 when model='logNormalShadowing'")
        return self

# Network - level configuration
class Nets(BaseModel):
    noise_th: float = Field(le=0)      # dBm threshold (should be negative e.g., -91)
    propagation_model: PropagationModel

# Regex for position validation
_POSITION_RE = re.compile(r"^\s*(-?\d+(\.\d+)?)\s*,\s*(-?\d+(\.\d+)?)\s*,\s*(-?\d+(\.\d+)?)\s*$")

def _validate_position_str(v: str) -> str:
    # Validate that position is 'x,y,z' with numerics values. 
    if not isinstance(v, str) or not _POSITION_RE.match(v):
        raise ValueError("position must be 'x,y,z' with numeric components")
    return v

# Access Point
class AP(BaseModel):
    # Defines a wireless access point (AP) in the topology. 
    id: str = Field(min_length=1)
    mode: Literal["a", "b", "g", "n", "ac", "ax"]
    channel: int = Field(gt=0)
    ssid: str = Field(min_length=1, max_length=32)
    position: str

    @field_validator("position")
    @classmethod
    def _pos_ok(cls, v: str) -> str:
        return _validate_position_str(v)

# Station
class Station(BaseModel):
    # Defines a wireless station (STA) in the topology.
    id: str = Field(min_length=1)
    position: str

    @field_validator("position")
    @classmethod
    def _pos_ok(cls, v: str) -> str:
        return _validate_position_str(v)

# Full topology definition
class Topology(BaseModel):
    # Represents the entire network topology:
    # -Network parameters
    # -List of APs
    # -List of stations
    nets: Nets
    aps: List[AP] = Field(default_factory=list)
    stations: List[Station] = Field(default_factory=list)

# ----- Tests Definitions -----

# Node movement test
class TestMove(BaseModel):
    # Moves a station to a new position at a given timeframe. 
    name: str
    type: Literal["node movements"]
    timeframe: int = Field(ge=0)     # UPDATED: required timeframe
    node: str
    position: str

    @field_validator("position")
    @classmethod
    def _pos_ok(cls, v: str) -> str:
        return _validate_position_str(v)

# IW command test (optional)
class TestIw(BaseModel):
    # Allows running an 'iw' shell command using {interface} placeholder.
    name: str
    type: Literal["iw"]
    cmd: str            # should ideally contain {interface}

# Union type for test vairnts
TestVariant = TestMove | TestIw

# Complete specification
class Spec(BaseModel):
    # Root-level schema of the input JSON.
    # Includes:
    #  - meta (experimental data)
    #  - topo (network topology)
    #  - tests (movement or command tests)
    #  - credential (optional)
    schemaVersion: str
    meta: Meta
    topo: Topology
    tests: List[TestVariant]
    # NEW: top-level passthrough fields for credentials / address
    username: str = ""
    password: str = ""
    address: str = ""

# ---------------- Validate Semantics ----------------

def _spec_hash(spec_dict: dict) -> str:
    # Compute a short SHA256 fingerprint of the spec for reproducibility.
    raw = json.dumps(spec_dict, sort_keys=True).encode()
    return hashlib.sha256(raw).hexdigest()[:12]

def _positions_tuple(pos: str) -> Tuple[float, float, float]:
    # Convert 'x,y,z' string into a float tuple. 
    m = _POSITION_RE.match(pos)
    assert m
    return (float(m.group(1)), float(m.group(3)), float(m.group(5)))

def validate_semantics(spec: Spec) -> Dict[str, List[dict]]:
    # Performs additional non-schema checks:
    #  - duplicate IDs
    #  - invalid positions
    #  - unkown nodes in tests
    #  - timeframe ordering 
    #  - backend mismatch
    #  - unrealistic noise threshold
    errors: List[dict] = []
    warnings: List[dict] = []

    # Unique IDs across APs + Stations check
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

    # Validate APs
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
    
    # Validate Stations 
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
    # Track per-station timeframe monotonicity
    last_tf: DefaultDict[str, int] = defaultdict(lambda: -1)
    
    # Validate Tests
    for i, t in enumerate(spec.tests):
        if isinstance(t, TestMove):
            # Check if station exists
            if t.node not in sta_set:
                errors.append({
                    "loc": f"tests[{i}].node",
                    "code": "unknown_station",
                    "msg": f"'{t.node}' is not a known station id"
                })
            # Check timeframe monotnicity
            # NEW: gentle sanity — non-decreasing timeframe per station
            prev = last_tf[t.node]
            if prev > t.timeframe:
                warnings.append({
                    "loc": f"tests[{i}].timeframe",
                    "code": "non_monotonic_timeframe",
                    "msg": f"timeframe {t.timeframe} for {t.node} is less than previous {prev}; check ordering"
                })
            last_tf[t.node] = max(prev, t.timeframe)

        elif isinstance(t, TestIw):
            if "{interface}" not in t.cmd:
                warnings.append({
                    "loc": f"tests[{i}].cmd",
                    "code": "missing_placeholder",
                    "msg": "cmd does not include '{interface}' placeholder"
                })

    # Backend mismatch warning(Wi-Fi entities under plain mininet)
    if spec.meta.backend == "mininet" and (spec.topo.aps or spec.topo.stations):
        warnings.append({
            "loc": "meta.backend",
            "code": "backend_role_mismatch",
            "msg": "APs/stations present but backend='mininet'. Use 'mininet-wifi' for Wi-Fi behavior."
        })

    # Noise threshold sanity
    if spec.topo.nets.noise_th > -30:
        warnings.append({
            "loc": "topo.nets.noise_th",
            "code": "suspicious_noise_th",
            "msg": f"noise_th {spec.topo.nets.noise_th} dBm is unusually high (less negative)"
        })

    return {"errors": errors, "warnings": warnings}

# ---------------- CLI ----------------

def parse_args():
    # Define CLI flags
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
    # Nicely print validation messages to stderr.
    print(prefix, file=sys.stderr)
    for it in items:
        loc = it.get("loc", "?")
        code = it.get("code", "error")
        msg = it.get("msg", "")
        print(f" - {loc} [{code}]: {msg}", file=sys.stderr)

# ---------------- Main Execution Logic ----------------
def main():
    args = parse_args()

    # Resolve config relative to this script's directory if not absolute and load json.
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
    
    # Schema validation
    # Load + structural validation
    try:
        with open(cfg_path) as f:
            data = json.load(f)
        spec = Spec(**data)
    except ValidationError as ve:
        # Structural (schema) errors
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
        # JSON parsing or file read errors
        out = {"ok": False, "errors": [{"loc": "root", "code": "load_error", "msg": str(e)}], "warnings": []}
        print(json.dumps(out, indent=2))
        _print_stderr("VALIDATION_ERROR: load error", out["errors"])
        sys.exit(1)

    # Semantic checks
    res = validate_semantics(spec)
    ok = len(res["errors"]) == 0
    
    # Add schema hash
    # Normalize + fingerprint
    norm = spec.model_dump()
    norm.setdefault("meta", {})["schema_hash"] = _spec_hash(norm)

    # Write normalized spec only if requested
    if ok and args.emit_spec:
        with open(args.emit_spec, "w") as f:
            json.dump(norm, f, indent=2)

    # Print results
    out = {"ok": ok, "errors": res["errors"], "warnings": res["warnings"]}
    if args.include_spec and ok:
        out["spec"] = norm
    print(json.dumps(out, indent=2))

    # Exit code
    if not ok:
        _print_stderr("VALIDATION_ERROR: semantic validation failed", res["errors"])
        sys.exit(1)

    sys.exit(0)

if __name__ == "__main__":
    main()
