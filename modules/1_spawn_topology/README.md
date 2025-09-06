# Spawn Mininet Topology over SSH
## How to start
1. Navigate to the "/1_spawn_topology" directory
2. Make sure the input topo.json file contains sufficient information (see below for more details)
3. Execute ```$ go run .``` in the terminal
4. The script should establish ssh connection and run automatically
5. <font style="color : lightskyblue">[Optional]</font> Add ```--cli``` flag to enable manual Mininet control (run ```exit``` to end the script)
6. <font style="color : lightskyblue">[Optional]</font> Run with the ```-h``` flag to see more information of how to use the flags

## Workflow
1. Read Mininet topology and input param from JSON file.

```json
// JSON example
{
  "hosts":    ["h1", "h2", "h3"],
  "switches": ["s1", "s2"],
  "links":    [["h1","s1"], ["h2","s1"], ["h3","s2"], ["s1","s2"]],
  "username": "<vm_username>",
  "password": "<ssh/sudo_password>",
  "host": "<vm_ip_address>" // ssh into <username>@<host>
}
```

2. Generate a Python file that establish topology in mininet.
3. Upload that file to the remote VM (default remote path: /tmp/topo_from_json.py).
4. Run Mininet on the VM with either:
```c
// Default (pingAll)

$ sudo mn --custom /tmp/topo_from_json.py --topo fromjson --test pingall
```
```c
// Mininet CLI (run interactively with -cli flag)

$ sudo mn --custom /tmp/topo_from_json.py --topo fromjson
```
5.	Cleanup: attempt to delete the remote Python file (rm -f /tmp/topo_from_json.py) before exiting.
## Requirements
- Local machine
  - Go (1.20+ recommended)
  - ssh and scp available in PATH
- Remote VM (prefer version above <font style="color : aquamarine">Linux Ubuntu 20.x.x</font>)
  - Mininet installed (mn in PATH)
  - Python (Mininet depends on it)
  - Enable remote ssh
    - Run ```systemctl start ssh``` to start ssh auth
    - Run ```systemctl status ssh``` to check ssh status
    - Run ```systemctl stop ssh``` to stop ssh auth
    - Run ```ifconfig``` to find host IPv4 address under ```ens160``` interface -> ```inet```
  - Ability to run ```sudo mn```


## TODO
- This is currently a stand-alone runner
- No custom test script yet (cannot feed a .cli file automatically)
