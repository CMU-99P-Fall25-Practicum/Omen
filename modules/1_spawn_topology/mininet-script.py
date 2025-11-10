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
      "propagation": {"model": "logDistance", "exp": 4, "std_dev": 2.0}
    },
    "ap": [
      {"id": "ap1", "ssid": "new-ssid", "mode": "a", "channel": 36, "position": "0,0,0"}
    ],
    "stations": [
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
    """
    Create a new results directory under /tmp/test_results named by the current timestamp.

    The folder name is generated using the current date and time 
    (e.g., 20251110_104530). If the base directory or timestamped 
    folder does not exist, it will be created automatically.

    Returns:
        str: Path to the created results directory.
    """
    base = "/tmp/test_results"
    # folder named by current time
    ts = datetime.now().strftime("%Y%m%d_%H%M%S")
    path = os.path.join(base, ts)
    os.makedirs(path, exist_ok=True)
    return path


def build_from_spec(spec):
    """
    Build and start a Mininet-WiFi network based on a given specification.

    The function reads network parameters, access points, and station settings 
    from the input dictionary 'spec', creates corresponding nodes, configures 
    the propagation model, and starts the network.

    Args:
        spec (dict): Network specification containing:
            - "nets": global settings (e.g., noise_th, propagation_model)
            - "aps": list of access point definitions
            - "stations": list of station definitions

    Returns:
        tuple: (net, sta_objs, ap_objs)
            - net: the Mininet-WiFi network instance
            - sta_objs: dictionary of station objects
            - ap_objs: dictionary of access point objects
    """
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

def run_pingall_full(all_nodes, count=1, test_name="pingall_full"):
    """
    Run a full pairwise ping matrix test between all nodes.
    Returns the formatted output string with CSV-style results.
    """
    msg = ""
    
    # Get all positions before ping tests
    info(f"*** Node Movement output: Timeframe {test_name} ***\n")
    for node in all_nodes:
        # Get the position using the position attribute or params dict
        current_pos = getattr(node, 'position', node.params.get('position', 'unknown'))
        info(f"Node {node.name} position before pingall_full: {current_pos}\n")
        msg += f"\n[node movements] {test_name}: move {node.name}: moving {node.name} -> {current_pos}\n"
        msg += f"Moved {node.name} to {current_pos}\n"
    
    msg += f"\n[pingall_full] {test_name}: pairwise matrix (-c {count})\n"
    info(msg)
    header = "src,dst,tx,rx,loss_pct,avg_rtt_ms\n"
    lines = [msg, header]
    
    for s in all_nodes:
        for d in all_nodes:
            if s is d:
                continue
            raw = s.cmd(f"ping -c {count} {d.IP()} | tail -n 2")
            tx = rx = loss = "?"
            avg = "?"
            for line in raw.splitlines():
                if "packets transmitted" in line:
                    # "X packets transmitted, Y received, Z% packet loss"
                    parts = [p.strip() for p in line.split(',')]
                    try:
                        tx = int(parts[0].split()[0])
                        rx = int(parts[1].split()[0])
                        loss = parts[2].split('%')[0]
                    except Exception:
                        pass
                if "min/avg/max" in line or "round-trip" in line:
                    try:
                        avg = line.split('=')[1].split('/')[1].strip()
                    except Exception:
                        pass
            lines.append(f"{s.name},{d.name},{tx},{rx},{loss},{avg}\n")
    
    return "".join(lines)

def run_iw_stations(sta_objs, ap_objs, cmd, test_name="iw_stations"):
    """
    Run an iw command on all stations.
    Returns the formatted output string with results from all stations.
    """
    msg = f"\n[iw_stations] {test_name}: running '{cmd}' on all stations\n"
    info(msg)
    lines = [msg]
    lines.append("=" * 60 + "\n")
    
    for station in sta_objs.values():
        station_msg = f"\n--- Station {station.name} ---\n"
        lines.append(station_msg)
        
        # Replace any placeholder in the command with actual station interface
        actual_cmd = cmd.replace("{station}", station.name)
        actual_cmd = actual_cmd.replace("{interface}", f"{station.name}-wlan0")
        

        # Retry multiple times to avoid "Not connected"
        retries = 5
        result = station.cmd(actual_cmd)
        while "Not connected" in result and retries > 0:
            info(f"Waiting for {station.name} to associate...\n")
            time.sleep(1)
            result = station.cmd(actual_cmd)
            retries -= 1

        lines.append(f"Command: {actual_cmd}\n")
        lines.append(f"Output:\n{result}\n")
    
    # APs
    for ap in ap_objs.values():
        ap_msg = f"\n--- Access Point {ap.name} ---\n"
        lines.append(ap_msg)

        # here we assume wlan1 is the main interface
        iface = f"{ap.name}-wlan1"
        cmd = f"ifconfig {iface}"
        result = ap.cmd(cmd)

        lines.append(f"Command: {ap.name} {cmd}\n")
        lines.append(f"Output:\n{result}\n")
    lines.append("=" * 60 + "\n")
    return "".join(lines)

def run_tests(sta_objs, ap_objs, spec, tests, results_dir):
    """
    Run all tests defined in 'tests' and save results by timeframe.

    Supports ping tests and node movements. After each timeframe, runs
    `pingall_full` and `iw` checks on all nodes. Each timeframe's output 
    is written to `timeframeX.txt` in `results_dir`.
    """

    info("*** Running tests\n")
    time.sleep(30)
    
    # convenience: list of all nodes we consider for full pairwise tests
    all_nodes = []
    all_nodes.extend(sta_objs.values())
    all_nodes.extend(ap_objs.values())

    tests_by_timeframe = [[]]
    for t in tests:
        cur_timeframe = t["timeframe"]
        while len(tests_by_timeframe) <= cur_timeframe:
            tests_by_timeframe.append([])
        tests_by_timeframe[cur_timeframe].append(t)

    for timeframe, sub_tests in enumerate(tests_by_timeframe):
        outfile = os.path.join(results_dir, f"timeframe{timeframe}.txt")
        info(f"Tests in timeframe{timeframe}:\n")
        out = ""

        for sub_t in sub_tests:
            ttype = sub_t["type"]
            name = sub_t['name']

            if ttype == "ping":
                src = sub_t["src"]
                dst = sub_t["dst"]
                count = int(sub_t["count"])
                msg = f"\n[ping] {name}: {src} -> {dst} (-c {count})\n"
                info(msg)

                src_node = sta_objs[src]
                dst_node = sta_objs.get(dst) or ap_objs.get(dst)
                target_ip = dst_node.IP()

                out += msg + src_node.cmd("ping -c {} {}".format(count, target_ip))

            elif ttype == "node movements":
                node = sta_objs[sub_t["node"]]
                pos = sub_t["position"]
                node.setPosition(pos)
                
            else:
                msg = f"\n[skip] unsupported test type: {ttype}\n"
                info(msg)
                out += msg
        # Run pinall_full after all tests in one timeframe have finished
        info("*** Running pingall_full after all the node movements within one timeframe\n")
        pingall_out = run_pingall_full(all_nodes, count=1, test_name=timeframe)
        out += "\n" + pingall_out

        # Run iw on all stations and access points after all tests in one timeframe have finished
        out += run_iw_stations(sta_objs, ap_objs, "iw dev {interface} link", "check_all_links")

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
    run_tests(sta_objs, ap_objs, spec, tests, results_dir)

    # info("*** CLI\n")
    # CLI(net)

    info("*** Stopping network\n")
    net.stop()

if __name__ == "__main__":
    setLogLevel('info')
    main()