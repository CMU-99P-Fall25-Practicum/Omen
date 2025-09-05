# Spawn Mininet Topology over SSH
## How it works (high level)
1. Read JSON topology (hosts, switches, links).
2. Generate a small Python file that defines class FromJSON(Topo) and registers it as topos['fromjson'].
3. Upload that file to the remote VM with scp (default remote path: /tmp/topo_from_json.py).
4. Run Mininet on the VM with either:
   - default (pingall)
```
sudo mn --custom /tmp/topo_from_json.py --topo fromjson --test pingall
```
   - or enter the Mininet CLI
```
sudo mn --custom /tmp/topo_from_json.py --topo fromjson
```
5.	Cleanup: attempt to delete the remote Python file (rm -f /tmp/topo_from_json.py) before exiting.
## Requirements
- Local machine
  - Go (1.20+ recommended)
  - ssh and scp available in PATH
- Remote VM
  - Mininet installed (mn in PATH)
  - Python (Mininet depends on it)
  - An account you can ssh into (e.g., mininet@<ip>)
  - Ability to run sudo mn (you may be prompted for a password)
## Usage
> ⚠️ Flag order matters: due to Go’s flag package, all flags must come before the positional topo.json.
### Flags
- --remote (required): remote target, e.g. mininet@192.168.64.5
- --cli (optional): open the Mininet CLI instead of running pingall
- --remote-path (optional): remote path for the generated Python file (default /tmp/topo_from_json.py)
### Examples
Run auto test (pingall) and exit:
```bash
./main_tiger --remote=mininet@10.0.0.53 topo.json
```
Open interactive Mininet CLI (you must type exit to quit):
```bash
./main_tiger --remote=mininet@10.0.0.53 --cli topo.json
```
Use a custom remote path:
```bash
./main_tiger --remote=mininet@10.0.0.53 --remote-path=/tmp/my_topo.py topo.json
```
### Topology JSON schema
```JSON
{
  "hosts":    ["h1", "h2", "h3"],
  "switches": ["s1", "s2"],
  "links":    [["h1","s1"], ["h2","s1"], ["s1","s2"], ["h3","s2"]]
}
```
## Known limitations / TODO
- Not integrated with Rory’s skeleton yet (this is a stand-alone runner).
- Flag order: topo.json must appear after all flags (Go flag package behavior).
- No custom test script yet (cannot feed a .cli file automatically).
- Multiple password prompts may occur (ssh, sudo).
