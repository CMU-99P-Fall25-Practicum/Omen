# Network Config Validator
# This script validates a JSON spec describing network topologies
# before it is handed off to Mininet through the Controller. 
# The goal: Catch errors early (bad IDs, broken references, 
# impossible values) and normalize the config into a predictable 
# structure for the rest of the pipeline. 

#!/usr/bin/env python3
import sys, json, argparse, hashlib
from typing import Literal, Optional, List, Dict
from pydantic import BaseModel, Field, ValidationError, model_validator

# ---------------- Pydantic Models ----------------
#Defines what a 'valid' spec looks like and complains if the user feeds 
#wrong information.
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

# class Node(BaseModel):
#     id: str = Field(min_length=1)
#     role: Literal["ap", "sta", "host", "switch"]
#     tx_dbm: Optional[float] = Field(default=None)
#     rx_sensitivity_dbm: Optional[float] = Field(default=None)

    #@field_validator("tx_dbm")
    #def tx_range(cls, v):
    #    if v is None:
    #        return v
    #    if not (-10 <= v <= 10): 
    #        raise ValueError("tx_dbm expected between -10...30 dbm for wifi")
    #    return v
    
    #@field_validator("rx_sensitivity_dbm")
    #def rx_range(cls, v):
    #    if v is None:
    #        return v
    #    if not(-110 <= v <= -40):
    #        raise ValueError("rx_sensitivity_dbm expected between -110 and -40 dbm for wifi")
    #    return v 

# class Constraints(BaseModel):
#     loss_pkt: Optional[float] = Field(default=0.0, ge=0.0, le=100.0)
#     throughput_mbps: Optional[float] = Field(default=None, gt=0.0)
#     mtu: Optional[int] = Field(default=1500, ge=256, le=65000)
#     delay_ms: Optional[float] = Field(default=0.0, ge=0.0)

# class Link(BaseModel):
#     node_id_a: str
#     node_id_b: str
#     constraints: Constraints = Field(default_factory=Constraints)

#     @model_validator(mode="after")
#     def check_nodes(self):
#         if self.node_id_a == self.node_id_b:
#             raise ValueError("A link must connect a node to a different node")
#         return self

# class Topology(BaseModel):
#     nodes: List[Node]
#     links: List[Link] = Field(default_factory=list)

# class TestPing(BaseModel):
#     name: str
#     type: Literal["ping"]
#     src: str
#     dst: str
#     count: Optional[int] = Field(default=5, ge=1)
#     deadline_s: Optional[int] = Field(default=5, ge=1)

# class TestPingall(BaseModel):
#     name: str
#     type: Literal["pingall"]

# class TestTCPiperf(BaseModel):
#     name: str
#     type: Literal["iperf_tcp"]
#     src: str
#     dst: str
#     duration_s: int = Field(gt=0)

# class TestUDPiperf(BaseModel):
#     name: str
#     type: Literal["iperf_udp"]
#     src: str
#     dst: str
#     duration_s: int = Field(gt=0)
#     rate_mbps: float = Field(gt=0)

# TestVariant = TestPing | TestPingall | TestTCPiperf | TestUDPiperf

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
    cmd: str  # should contain {interface}

TestVariant = TestMove | TestIw

class Spec(BaseModel):
    schemaVersion: str
    meta: Meta
    topo: Topology
    tests: List[TestVariant]

# ----------------Validate Semantics ----------------

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

    # There must be unique node IDs
    # ids = [n.id for n in spec.topo.nodes]
    # Unique IDs across APs + Stations
    ap_ids = [a.id for a in spec.topo.aps]
    sta_ids = [s.id for s in spec.topo.stations]
    all_ids = ap_ids + sta_ids
    if len(ids) != len(set(ids)):
        errors.append({"loc": "topo.nodes[*].id", "code": "duplicate_id",
                       "msg": "duplicate node IDs found"})

    # node_set = set(ids)
    sta_set = set(sta_ids)
    ap_set = set(ap_ids)

    # Basic plausibility checks
    for i, ap in enumerate(spec.topo.aps):
        # parse position (already validated format)
        try:
            _positions_tuple(ap.position)
        except Exception:
            errors.append({"loc": f"topo.aps[{i}].position", "code": "bad_position",
                           "msg": "cannot parse 'x,y,z' as floats"})
        if ap.mode == "a" and ap.channel <= 0:
            errors.append({"loc": f"topo.aps[{i}].channel", "code": "bad_channel",
                           "msg": "5GHz channel must be a positive integer (e.g., 36)"})

    for i, sta in enumerate(spec.topo.stations):
        try:
            _positions_tuple(sta.position)
        except Exception:
            errors.append({"loc": f"topo.stations[{i}].position", "code": "bad_position",
                           "msg": "cannot parse 'x,y,z' as floats"})

    # Tests
    for i, t in enumerate(spec.tests):
        if isinstance(t, TestMove):
            if t.node not in sta_set:
                errors.append({"loc": f"tests[{i}].node", "code": "unknown_station",
                               "msg": f"'{t.node}' is not a known station id"})
            # position format already checked via field validator
        elif isinstance(t, TestIw):
            if "{interface}" not in t.cmd:
                warnings.append({"loc": f"tests[{i}].cmd", "code": "missing_placeholder",
                                 "msg": "cmd does not include '{interface}' placeholder"})

    # Backend mismatch (Wi-Fi entities under plain mininet)
    if spec.meta.backend == "mininet" and (spec.topo.aps or spec.topo.stations):
        warnings.append({"loc": "meta.backend", "code": "backend_role_mismatch",
                         "msg": "APs/stations present but backend='mininet'. Use 'mininet-wifi' for Wi-Fi behavior."})

    # Nets sanity
    if spec.topo.nets.noise_th > -30:
        warnings.append({"loc": "topo.nets.noise_th", "code": "suspicious_noise_th",
                         "msg": f"noise_th {spec.topo.nets.noise_th} dBm is unusually high (less negative)"})

    return {"errors": errors, "warnings": warnings}

    # # Links reference valid nodes
    # for i, link in enumerate(spec.topo.links):
    #     if link.node_id_a not in node_set:
    #         errors.append({"loc": f"topo.links[{i}].node_id_a", "code": "unknown_node",
    #                        "msg": f"'{link.node_id_a}' not found"})
    #     if link.node_id_b not in node_set:
    #         errors.append({"loc": f"topo.links[{i}].node_id_b", "code": "unknown_node",
    #                        "msg": f"'{link.node_id_b}' not found"})
    #     c = link.constraints
    #     if c.loss_pkt is not None and c.loss_pkt > 50:
    #         warnings.append({"loc": f"topo.links[{i}].constraints.loss_pkt",
    #                          "code": "suspicious_loss",
    #                          "msg": f"loss {c.loss_pkt}% is very high"})

    # # Tests reference valid nodes
    # for i, t in enumerate(spec.tests):
    #     if hasattr(t, "src") and getattr(t, "src") not in node_set:
    #         errors.append({"loc": f"tests[{i}].src", "code": "unknown_node",
    #                        "msg": f"src '{getattr(t, 'src')}' not found in topo.nodes"})
    #     if hasattr(t, "dst") and getattr(t, "dst") not in node_set:
    #         errors.append({"loc": f"tests[{i}].dst", "code": "unknown_node",
    #                        "msg": f"dst '{getattr(t, 'dst')}' not found in topo.nodes"})

    # # Backend/role mismatch hint
    # if spec.meta.backend == "mininet":
    #     if any(n.role in ("ap", "sta") for n in spec.topo.nodes):
    #         warnings.append({"loc": "meta.backend", "code": "backend_role_mismatch",
    #                          "msg": "roles 'ap/sta' suggest WiFi; backend is 'mininet'. Downstream may map to host/switch."})

    # return {"errors": errors, "warnings": warnings}

# ---------------- CLI ----------------

def parse_args():
    ap = argparse.ArgumentParser(description="Validate and normalize network spec")
    ap.add_argument("config", help="input config JSON")
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


# CLI Entrypoint 
def main():
    args = parse_args()

    # Load + structural validation
    try:
        data = json.load(open(args.config))
        spec = Spec(**data)
    except ValidationError as ve:
        out = {
            "ok": False,
            "errors": [{"loc": ".".join(map(str, e["loc"])), "code": "pydantic", "msg": e["msg"]}
                       for e in ve.errors()],
            "warnings": []
        }
        print(json.dumps(out, indent=2))
        sys.exit(1)
    except Exception as e:
        out = {"ok": False, "errors": [{"loc": "root", "code": "load_error", "msg": str(e)}], "warnings": []}
        print(json.dumps(out, indent=2))
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
    # sys.exit(0 if ok else 1)

    if not ok:
        _print_stderr("VALIDATION_ERROR: semantic validation failed", res["errors"])
        sys.exit(1)

    sys.exit(0)

if __name__ == "__main__":
    main()
