# Spawn Mininet Topology over SSH
## How to start
1. Navigate to the "/1_spawn_topology" directory
2. Make sure the input topo.json file contains sufficient information (see below for more details)
3. Execute ```$ go run .``` in the terminal
4. The script should establish ssh connection and run automatically
5. <font style="color : lightskyblue">[Optional]</font> Run with the ```-h``` flag to see more information of how to use the flags

## Workflow
1. Read Mininet topology and input param from JSON file.

```json
// JSON example
{
  "schemaVersion": "1.0",
  "meta": {
    "backend": "mininet" ,
    "name": "campus-demo",  
    "duration_s": 60
  },
  "topo": {
    "nets": {
      "noise_th": -91,
      "propagation_model":{
        "model": "logDistance",
        "exp": 4
      }
    },
    "aps": [
      {
        "id": "ap1",
        "mode": "a",
        "channel": 36,
        "ssid": "test-ssid1",
        "position": "0,0,0"
      }
    ],
    "stations": [
      {
        "id": "sta1",
        "position": "0,10,0"
      },
    ]
  },
  "tests": [
    {
        "name":"1: move sta1",
        "type":"node movements",
        "node": "sta1",
        "position": "0,5,0"
    }
  ],
  "username": "<vm_username>",
  "password": "<ssh/sudo_password>",
  "host": "<vm_ip_address>" // ssh into <username>@<host>
}
```

2. Upload our [Mininet-WiFi Python script](./mininet-script.py) and the [input JSON file](./input-topo.json) to the VM.
3. The files are all under the ```/tmp``` directory.
4. Run the script with the following command:
```c
$ sudo python3 /tmp/mininet-script.py /tmp/input-topo.json
```

5. Download the raw output files for further processing in the next module ([mn_raw_output_processing](../2_mn_raw_output_processing/)).


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
- No test script customization yet (cannot feed a .cli file automatically)
