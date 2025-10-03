import json,sys, argparse
from pydantic import BaseModel, Field, field_validator, ValidationError, model_validator
from typing import Literal, Optional, List, Dict

#Pydantic Models#
#Adding normalisation of inputs later#

class Meta(BaseModel):
    backend: Literal["mininet", "mininet-wifi"]
    name: str = Field(min_length = 1, max_length = 64)
    duration_s: int = Field(gt = 0)

class Nodes(BaseModel):
    id: str = Field(min_length = 1)
    role: Literal["ap", "sta", "host", "switch"]
    tx_dbm: Optional[float] = Field(default=None)               #for mininet propagation model/ to be set later
    rx_sensitivity_dbm: Optional[float] = Field(default=None)   #for mininet propagation model/ to be set later

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
        
class Constraints(BaseModel):
    loss_pkt: Optional[float] = Field(default = 0.0, ge=0.0, le=100.0)
    throughput_mbps:  Optional[float] = Field(default = None, gt = 0.0)
    mtu:  Optional[int] = Field(default = 1500, ge=256, le=65000)      #65000 for jumbograms, ipv6
    delay_ms:  Optional[float] = Field(default = 0.0, ge=0.0)

class Link(BaseModel):
    node_id_a: str
    node_id_b: str
    constraints: Constraints = Field(default_factory= Constraints)

    @model_validator(mode = "after")
    def check_nodes(self):
        if self.node_id_a == self.node_id_b:
            raise ValueError("A link must connect two different nodes")
        return self

class Topology(BaseModel):
    nodes: List[Nodes]
    links: List[Link] = Field(default_factory=list)

class TestPing(BaseModel):
    name: str
    type: Literal["ping"]
    src: str
    dst: str 
    count: Optional[int] = Field(default = 5, ge = 1)                #sending 5 packets
    deadline_s: Optional[int] = Field(default = 5, ge = 1)           #total run time that ping takes

class TestPingall(BaseModel):
    name: str
    type: Literal["pingall"]

class TestTCPiperf(BaseModel):
    name: str
    type: Literal["iperf_tcp"]
    src: str
    dst: str
    duration_s: int = Field(gt=0)

class TestUDPiperf(BaseModel):
    name: str
    type: Literal["iperf_udp"]
    src: str
    dst: str
    duration_s: int = Field(gt=0)
    rate_mbps: float = Field(gt=0)

TestVariant =  TestPing | TestPingall | TestTCPiperf | TestUDPiperf

class Spec(BaseModel):
    schemaVersion: str
    meta: Meta
    topo: Topology
    tests: List[TestVariant]
    
#Cross Check References to be added#

def validate_semantics(spec: Spec) -> Dict[str, List[str]]:
    errors: List[str] = []
    warnings: List[str] = []

    # Uniqueness of node IDs
    ids = [n.id for n in spec.topo.nodes]
    if len(ids) != len(set(ids)):
        errors.append("duplicate node IDs found")

    # Links point to valid nodes
    node_set = set(ids)
    for i, link in enumerate(spec.topo.links):
        if link.node_id_a not in node_set:
            errors.append(f"links[{i}].node_id_a '{link.node_id_a}' not found")
        if link.node_id_b not in node_set:
            errors.append(f"links[{i}].node_id_b '{link.node_id_b}' not found")

    # Tests reference valid nodes (where applicable)
    for i, t in enumerate(spec.tests):
        if hasattr(t, "src") and t.src not in node_set:
            errors.append(f"tests[{i}].src '{t.src}' not found in topo.nodes")
        if hasattr(t, "dst") and t.dst not in node_set:
            errors.append(f"tests[{i}].dst '{t.dst}' not found in topo.nodes")

    return {"errors": errors, "warnings": warnings}
 
# ---------------- topo.py generator --------------

def generate_topo_py(spec: Spec) -> str:
    s = spec.model_dump()
    nodes = s["topo"]["nodes"]
    links = s["topo"]["links"]
    tests = s["tests"]
    backend = s["meta"]["backend"]

    # if backend == "mininet-wifi":
    #     # ---------- Mininet-WiFi script ----------
    #     def node_decl(n):
    #         role = n["role"]
    #         if role == "ap":
    #             # basic Wi-Fi params; tweak later as needed
    #             return f"    {n['id']} = net.addAccessPoint('{n['id']}', ssid='{n['id']}-ssid', mode='g', channel='1')"
    #         elif role == "sta":
    #             return f"    {n['id']} = net.addStation('{n['id']}')"
    #         elif role == "host":
    #             return f"    {n['id']} = net.addHost('{n['id']}')"
    #         elif role == "switch":
    #             return f"    {n['id']} = net.addSwitch('{n['id']}')"
    #         else:
    #             return f"    # unknown role: {role}"

    #     # For Wi-Fi, you usually don't need wired links between sta<->ap,
    #     # but if the spec declares links, treat them as wired management links.
    #     def link_decl(l):
    #         c = l.get("constraints", {}) or {}
    #         bw = int(c.get("throughput_mbps", 100) or 100)
    #         loss = float(c.get("loss_pkt", 0) or 0)
    #         delay = int(c.get("delay_ms", 0) or 0)
    #         return f"    net.addLink({l['node_id_a']}, {l['node_id_b']}, bw={bw}, loss={loss}, delay='{delay}ms')"

    #     lines = [
    #         "#!/usr/bin/env python3",
    #         "import json",
    #         "from mn_wifi.net import Mininet_wifi",
    #         "from mn_wifi.node import Controller, OVSKernelAP",
    #         "from mn_wifi.log import setLogLevel",
    #         "",
    #         "def main():",
    #         "    setLogLevel('warning')",
    #         "    net = Mininet_wifi(controller=Controller, accessPoint=OVSKernelAP)",
    #         "    c0 = net.addController('c0')",
    #     ]
    #     lines += [node_decl(n) for n in nodes]
    #     lines += ["    net.configureWifiNodes()"]
    #     lines += [link_decl(l) for l in links]
    #     lines += [
    #         "    net.build()",
    #         "    c0.start()",
    #         # start all APs with controller
    #         "    for ap in net.aps: ap.start([c0])",
    #         "    results = {}",
    #     ]

    #     # tests
    #     for t in tests:
    #         if t["type"] == "ping":
    #             count = int(t.get("count", 5)); deadline = int(t.get("deadline_s", 5))
    #             lines += [
    #                 f"    out = {t['src']}.cmd('ping -c {count} -w {deadline} ' + {t['dst']}.IP())",
    #                 f"    sent={count}; recv=out.count('time=')",
    #                 f"    results['{t['name']}'] = {{'sent': sent, 'recv': recv, 'loss_pct': 100.0*(sent-recv)/sent}}",
    #             ]
    #         elif t["type"] == "iperf_tcp":
    #             dur = int(t["duration_s"])
    #             lines += [
    #                 f"    {t['dst']}.cmd('iperf3 -s -D')",
    #                 f"    out = {t['src']}.cmd('iperf3 -c ' + {t['dst']}.IP() + ' -t {dur}')",
    #                 f"    results['{t['name']}'] = {{'raw': out}}",
    #                 f"    {t['dst']}.cmd('pkill -f iperf3')",
    #             ]
    #         elif t["type"] == "iperf_udp":
    #             dur = int(t["duration_s"]); rate = float(t['rate_mbps'])
    #             lines += [
    #                 f"    {t['dst']}.cmd('iperf3 -s -D')",
    #                 f"    out = {t['src']}.cmd('iperf3 -u -b {rate}M -t {dur} -c ' + {t['dst']}.IP())",
    #                 f"    results['{t['name']}'] = {{'raw': out}}",
    #                 f"    {t['dst']}.cmd('pkill -f iperf3')",
    #             ]

    #     lines += [
    #         "    net.stop()",
    #         "    print(json.dumps({'ok': True, 'metrics': results}))",
    #         "",
    #         "if __name__ == '__main__':",
    #         "    main()",
    #     ]
    #     return "\n".join(lines) + "\n"

    def node_decl(n):
        if n["role"] in ("host", "sta"):
            return f"    {n['id']} = net.addHost('{n['id']}')"
        elif n["role"] == "ap":
            return f"    {n['id']} = net.addSwitch('{n['id']}')  # treating AP as L2 switch in classic Mininet"
        else:
            return f"    {n['id']} = net.addSwitch('{n['id']}')"

    def link_decl(l):
        c = l.get("constraints", {}) or {}
        bw = int(c.get("throughput_mbps", 100) or 100)
        loss = float(c.get("loss_pkt", 0) or 0)
        delay = int(c.get("delay_ms", 0) or 0)
        return f"    net.addLink({l['node_id_a']}, {l['node_id_b']}, bw={bw}, loss={loss}, delay='{delay}ms')"

    lines = [
        "#!/usr/bin/env python3",
        "import json",
        "from mininet.net import Mininet",
        "from mininet.node import OVSController",
        "from mininet.link import TCLink",
        "from mininet.log import setLogLevel",
        "",
        "def main():",
        "    setLogLevel('warning')",
        "    net = Mininet(controller=OVSController, link=TCLink)",
    ]
    lines += [node_decl(n) for n in nodes]
    lines += [link_decl(l) for l in links]
    lines += [
        "    net.start()",
        "    results = {}",
    ]
    for t in tests:
        if t["type"] == "ping":
            count = int(t.get("count", 5)); deadline = int(t.get("deadline_s", 5))
            lines += [
                f"    out = {t['src']}.cmd('ping -c {count} -w {deadline} ' + {t['dst']}.IP())",
                f"    sent={count}; recv=out.count('time=')",
                f"    results['{t['name']}'] = {{'sent': sent, 'recv': recv, 'loss_pct': 100.0*(sent-recv)/sent}}",
            ]
        elif t["type"] == "iperf_tcp":
            dur = int(t["duration_s"])
            lines += [
                f"    {t['dst']}.cmd('iperf3 -s -D')",
                f"    out = {t['src']}.cmd('iperf3 -c ' + {t['dst']}.IP() + ' -t {dur}')",
                f"    results['{t['name']}'] = {{'raw': out}}",
                f"    {t['dst']}.cmd('pkill -f iperf3')",
            ]
        elif t["type"] == "iperf_udp":
            dur = int(t["duration_s"]); rate = float(t['rate_mbps'])
            lines += [
                f"    {t['dst']}.cmd('iperf3 -s -D')",
                f"    out = {t['src']}.cmd('iperf3 -u -b {rate}M -t {dur} -c ' + {t['dst']}.IP())",
                f"    results['{t['name']}'] = {{'raw': out}}",
                f"    {t['dst']}.cmd('pkill -f iperf3')",
            ]
    lines += [
        "    net.stop()",
        "    print(json.dumps({'ok': True, 'metrics': results}))",
        "",
        "if __name__ == '__main__':",
        "    main()",
    ]
    return "\n".join(lines) + "\n"


#CLI#
def parse_args():
    ap = argparse.ArgumentParser()
    ap.add_argument("config", help="input config JSON")
    ap.add_argument("--emit-spec", help="path to write normalized spec.json")
    ap.add_argument("--emit-topo", help="path to write generated topo.py")
    ap.add_argument("--include-spec", action="store_true",
                    help="include normalized spec in stdout summary")
    return ap.parse_args()

def main():
    args = parse_args()

    try:
        data = json.load(open(args.config))
        spec = Spec(**data)  # Pydantic validation
    except ValidationError as ve:
        out = {
            "ok": False,
            "errors": [f"{'.'.join(map(str, e['loc']))}: {e['msg']}" for e in ve.errors()],
            "warnings": []
        }
        print(json.dumps(out, indent=2))
        sys.exit(1)
    except Exception as e:
        out = {"ok": False, "errors": [str(e)], "warnings": []}
        print(json.dumps(out, indent=2))
        sys.exit(1)

    res = validate_semantics(spec)
    ok = len(res["errors"]) == 0

    # Write artifacts only if valid
    if ok and args.emit_spec:
        with open(args.emit_spec, "w") as f:
            json.dump(spec.model_dump(), f, indent=2)
    if ok and args.emit_topo:
        with open(args.emit_topo, "w") as f:
            f.write(generate_topo_py(spec))

    # stdout summary
    out = {"ok": ok, "errors": res["errors"], "warnings": res["warnings"]}
    if args.include_spec and ok:
        out["spec"] = spec.model_dump()
    print(json.dumps(out, indent=2))
    sys.exit(0 if ok else 1)

if __name__ == "__main__":
    main()