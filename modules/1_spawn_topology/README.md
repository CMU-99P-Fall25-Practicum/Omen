# Spawn Mininet Topology over SSH

## Usage

1. Create input json file.
  - Example:
    ```json
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
2. Build the binary. From the top level `omen/` directory call `mage`. A binary will be placed in `Omen/artefacts/`.
  - You can also build just this binary with `go build -C modules/1_spawn_topology/ -o ../../artefacts/2_output_processing`
3. Run the binary, providing the path to the input json.
  - Pass `-h` or usage details.
  - `artefacts/2_output_processing path/to/input.json`


4. The script should establish ssh connection and run automatically
5. <font style="color : lightskyblue">[Optional]</font> Run with the ```-h``` flag to see more information of how to use the flags.

## Module Workflow

1. Slurp input json, using the ssh info to connect to the mininet vm.

2. Upload our [Mininet-WiFi Python script](./mininet-script.py) and the [input JSON file](./input-topo.json) to the VM, placing them in `/tmp`.
3. Run the script: `sudo python3 /tmp/mininet-script.py /tmp/input-topo.json`
4. Download the raw output files for further processing in the next module ([mn_raw_output_processing](../2_mn_raw_output_processing/)).


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
