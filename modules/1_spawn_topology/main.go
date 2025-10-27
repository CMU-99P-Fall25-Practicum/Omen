/*
Package main implements the test runner module, capable of executing topologies and tests against a remote mininet host.

# Workflow

The internal logic of the module is as follows:

1. Slurp input json, using the ssh info to connect to the mininet vm.

2. Upload the driver script and input json files to the vm.

3. Run the script via `sudo python3 /tmp/mininet-script.py /tmp/input-topo.json`.

4. Download the raw output files for further processing in the [next (output handler)](../2_mn_raw_output_processing) module.

# Dependencies

- Go 1.24.4+

- ssh and scp in client PATH

- Remote vm with the following items in their path:
Mininet
Python (3.11+)
Sudo (required to run mininet)

- Remote vm must also have an ssh server available for connection and superuser permissions (to run mininet).
*/
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/netip"
	"os"
	"strings"

	"github.com/CMU-99P-Fall25-Practicum/Omen/modules/spawn_topology/models"
)

/* TODO
- This is currently a stand-alone runner
- No test script customization yet (cannot feed a .cli file automatically)
*/

// Configuration - Set these to hardcode values, leave empty for prompting
var (
	defaultHost         = ""
	defaultUsername     = ""
	defaultPassword     = ""
	defaultTopoFile     = "input-topo.json"   // default topology filename
	defaultPythonScript = "mininet-script.py" // default python script filename
)

// flag values
var (
	remote string
	config models.Config
)

// generate flags for later parsing
func init() {
	// Define flags
	flag.StringVar(&remote, "remote", "", "remote target to run on, e.g. username@192.168.64.5")
	flag.BoolVar(&config.UseCLI, "cli", false, "enter Mininet CLI instead of running pingall")
	flag.StringVar(&config.RemotePathPython, "remote-path-python", "/tmp/"+defaultPythonScript, "remote path for the generated Python file")
	flag.StringVar(&config.RemotePathJSON, "remote-path-json", "/tmp/"+defaultTopoFile, "remote path for the generated JSON file")
	flag.BoolVar(&config.Interactive, "interactive", true, "enables prompting for missing information. If false, this module will fail out on missing information rather than prompting for it.")

	// set default values
	config.TopoFile = defaultTopoFile

	// Custom usage function
	flag.Usage = func() {
		var bin = "UNKNOWN"
		if len(os.Args) > 1 {
			bin = os.Args[0]
		}

		fmt.Fprintf(os.Stderr,
			`Mininet Topology Manager

Usage: %[1]v [OPTIONS] [topo.json]

Description:
  This program creates and runs Mininet topologies from JSON files on remote VMs.
  It handles SSH connections, uploads topology scripts, and manages Mininet sessions.

Options:
  --remote=USER@HOST            Remote VM to connect to (e.g., gavinliao89@192.168.1.100)
  --cli                         Enter interactive Mininet CLI (default: run pingall and exit)
  --remote-path-python=PATH     Remote path for generated Python file (default: /tmp/mininet-script.py)
  --remote-path-json=PATH       Remote path for generated JSON file (default: /tmp/input-topo.json)
  -h, --help                    Show this help message

Arguments:
  topo.json          JSON file containing network topology (default: topo.json)

JSON Format example:
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

Examples:
  %[1]v --remote=user@192.168.1.100 --cli topo.json
  %[1]v --cli --remote=user@192.168.1.100
  %[1]v topo.json  # Prompts for connection info

Note: If connection info is not provided via --remote flag, the program will
      check the JSON file, then hardcoded defaults, then prompt for input.`, bin)
	}
}

// loadInputTopoFile slurps the given file and attempts to unmarshal it (from JSON) to a Topo struct.
func loadInputTopoFile(filename string) (*models.Input, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read topo file: %w", err)
	}

	var input models.Input
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}

	return &input, nil
}

/*
*
Hierarchical priority: command line flags > JSON file > hardcoded defaults > user input

resolveConfig() fetches requires configuration information (username, host, and password) hierarchically.

Note: Keep the "user input" functionality for now. Opt to remove when future pipeline is complete
*/
func resolveConfig(config *models.Config, js *models.Input) error {
	// Resolve username
	if config.Username == "" {
		if js.Username != "" {
			config.Username = js.Username
			fmt.Printf("Using username from JSON: %s\n", config.Username)
		} else if defaultUsername != "" {
			config.Username = defaultUsername
			fmt.Printf("Using hardcoded username: %s\n", config.Username)
		} else if config.Interactive {
			config.Username = getInput("Enter username: ")
		}
	} else {
		fmt.Printf("Using username from --remote flag: %s\n", config.Username)
	}

	if config.Username == "" {
		return errors.New("username must be supplied")
	}

	// Resolve host
	config.Host = func() netip.AddrPort {
		// if it was set by cli, we are done
		if config.Host.IsValid() {
			fmt.Printf("Using host from --remote flag: %v\n", config.Host)
			return config.Host
		}

		// Pull VM address from input JSON
		// Check if default port exists
		if js.AP != "" && !strings.Contains(js.AP, ":") {
			fmt.Printf("No port detected -> Using default port 22\n")
			js.AP = js.AP + ":22"
		}
		if ap, err := netip.ParseAddrPort(js.AP); err == nil {
			fmt.Printf("Using host from JSON: %v\n", ap)
			return ap
		}

		// Pull hosts from input JSON
		if ap, err := netip.ParseAddrPort(defaultHost); err == nil {
			fmt.Printf("Using hardcoded host: %v\n", ap)
			return ap
		}

		if config.Interactive {
			// pull from stdin
			var ap netip.AddrPort
			var err error
			for ap, err = netip.ParseAddrPort(getInput("Enter a valid target of the form '<host>:<port>':")); err != nil; {
			}
			return ap
		}
		return netip.AddrPort{}
	}()
	if !config.Host.IsValid() {
		return errors.New("a valid host/target must be supplied")
	}

	// Resolve password
	if config.Password == "" {
		if js.Password != "" {
			config.Password = js.Password
			fmt.Println("Using password from JSON: [hidden]")
		} else if defaultPassword != "" {
			config.Password = defaultPassword
			fmt.Println("Using hardcoded password: [hidden]")
		} else if config.Interactive {
			config.Password = getInput("Enter password (SSH/sudo): ")
		}
	}

	// Validate required fields
	if config.Username == "" || !config.Host.IsValid() || config.Password == "" {
		return fmt.Errorf("username, host, and password are required")
	}

	return nil
}

func main() {
	if err := parseFlags(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse arguments: %v\n", err)
		os.Exit(1)
	}

	// Load topology from JSON file
	fmt.Printf("Loading topology from: %s\n", config.TopoFile)
	inputTopo, err := loadInputTopoFile(config.TopoFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	// Resolve configuration (merge flags, JSON, defaults, and user input)
	if err := resolveConfig(&config, inputTopo); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	// Display final configuration
	fmt.Printf("\n"+`Final Configuration:
	Host               : %s
	Username           : %s
	Password           : [hidden]
	Topology           : %s
	Py Script          : %s
	Mode               : %s
	Remote Python path : %s
	Remote JSON path   : %s
	Hosts              : %v
	Stations           : %v
	Switches           : %v
	Aps                : %v
	Links              : %v`+"\n",
		config.Host,
		config.Username,

		config.TopoFile,
		defaultPythonScript,
		map[bool]string{true: "Interactive CLI", false: "Automated pingall"}[config.UseCLI],
		config.RemotePathPython,
		config.RemotePathJSON,
		inputTopo.Topo.Hosts,
		inputTopo.Topo.Stations,
		inputTopo.Topo.Switches,
		inputTopo.Topo.Aps,
		inputTopo.Topo.Links)

	// Execute the remote Mininet session
	if err := runRemoteMininet(&config, defaultPythonScript); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: run remote mininet: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Program completed successfully!")
}

// Parses the default flagset and sets SSH information if --remote was specified.
func parseFlags() error {
	flag.Parse()

	// Parse remote flag if provided
	if remote != "" {
		parts := strings.Split(remote, "@")
		if len(parts) != 2 {
			return fmt.Errorf("invalid remote format, expected username@host")
		}
		config.Username = parts[0]
		config.Host, _ = netip.ParseAddrPort(parts[1]) // throw away error; validity is checked later
	}

	// Get topology file (either from args or default)
	if flag.NArg() > 0 {
		config.TopoFile = flag.Arg(0)
	}

	return nil
}
