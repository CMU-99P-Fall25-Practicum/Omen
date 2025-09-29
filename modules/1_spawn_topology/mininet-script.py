#!/usr/bin/env python
# -*- coding: utf-8 -*-

"""
Build Mininet-WiFi from a JSON spec and run tests.
Usage:
    sudo python run_from_json.py topo.json

Expected JSON:
{
  "topo": {
    "net": {
      "noise_th": -91,
      "propagation": {"model": "logDistance", "exp": 4}
    },
    "ap": [
      {"id": "ap1", "ssid": "new-ssid", "mode": "a", "channel": 36, "position": "0,0,0"}
    ],
    "stationt": [
      {"id": "sta1", "position": "10,0,0"},
      {"id": "sta2", "position": "-10,0,0"}
    ],
    "test": [
      {"name": "ping_sta1_sta2", "type": "ping", "src": "sta1", "dst": "sta2", "count": 3}
    ]
  }
}
"""

import sys, json
import os
import time
from datetime import datetime
from mininet.log import setLogLevel, info, error
from mn_wifi.net import Mininet_wifi
from mn_wifi.cli import CLI
from mn_wifi.link import wmediumd
from mn_wifi.wmediumdConnector import interference

def make_results_dir():
    base = "test_results"
    # folder named by current time
    ts = datetime.now().strftime("%Y%m%d_%H%M%S")
    path = os.path.join(base, ts)
    os.makedirs(path, exist_ok=True)
    return path


def build_from_spec(spec):
    # net options
    net_cfg = spec["nets"]
    noise_th = net_cfg["noise_th"]

    net = Mininet_wifi(link=wmediumd, wmediumd_mode=interference, noise_th=noise_th)
    info("*** Creating nodes\n")

    # single default controller
    c1 = net.addController("c1")

    # APs
    ap_objs = {}
    for ap in spec["aps"]:
        ap_id = ap["id"]
        params = {
            "ssid": ap["ssid"],
            "mode": ap["mode"],
            "channel": ap["channel"],
            "position": ap["position"]
        }
        ap_objs[ap_id] = net.addAccessPoint(ap_id, **params)

    # Stations
    sta_objs = {}
    for s in spec["stations"]:
        sid = s["id"]
        params = {
            "position":s["position"]
        }
        sta_objs[sid] = net.addStation(sid, **params)

    # Propagation Model
    prop = net_cfg["propagation_model"]
    model = prop["model"]
    kwargs = {k: v for k, v in prop.items() if k != "model"}
    info("*** Propagation: %s %s\n" % (model, kwargs))
    net.setPropagationModel(model=model, **kwargs)

    info("*** Configuring nodes\n")
    net.configureNodes()

    info("*** Building & starting\n")
    net.build()
    c1.start()
    for ap in ap_objs.values():
        ap.start([c1])

    return net, sta_objs, ap_objs

def run_tests(sta_objs, ap_objs, tests, results_dir):
    info("*** Running tests\n")
    time.sleep(15)
    for idx, t in enumerate(tests, 1):
        ttype = t["type"]
        name = t['name']
        outfile = os.path.join(results_dir, f"test{idx}.txt")

        if ttype == "ping":
            src = t["src"]
            dst = t["dst"]
            count = int(t["count"])
            msg = f"\n[ping] {name}: {src} -> {dst} (-c {count})\n"
            info(msg)

            src_node = sta_objs[src]
            dst_node = sta_objs.get(dst) or ap_objs.get(dst)
            target_ip = dst_node.IP()

            out = msg + src_node.cmd("ping -c {} {}".format(count, target_ip))

        elif ttype == "iw":
            node = sta_objs[t["node"]]
            cmd = t["cmd"]
            msg = f"\n[iw] {name}: {node.name} {cmd}\n"
            info(msg)
            out = msg + node.cmd(cmd)

        elif ttype == "node movements":
            node = sta_objs[t["node"]]
            pos = t["position"]
            msg = f"\n[node movements] {name}: moving {node.name} -> {pos}\n"
            info(msg)
            node.setPosition(pos)
            time.sleep(5)
            out = msg + f"Moved {node.name} to {pos}\n"
        else:
            msg = f"\n[skip] unsupported test type: {ttype}\n"
            info(msg)
            out = msg
        # Write output to file
        with open(outfile, "w") as f:
            f.write(out)
    info("\n*** All tests are complete\n")
def main():

    with open(sys.argv[1], "r") as f:
        raw = json.load(f)

    spec = raw["topo"]
    tests = raw["tests"]

    net, sta_objs, ap_objs = build_from_spec(spec)

    results_dir = make_results_dir()
    run_tests(sta_objs, ap_objs, tests, results_dir)

    # info("*** CLI\n")
    # CLI(net)

    info("*** Stopping network\n")
    net.stop()

if __name__ == "__main__":
    setLogLevel('info')
    main()