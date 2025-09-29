package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/netip"
	"os"
	"strings"

	"github.com/CMU-99P-Fall25-Practicum/Omen/modules/spawn_topology/models"
)

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
    "topo": {
      "nets":{},
	  "aps":{}
    },
    "tests": [
	  { <testing info> }
    ],
    "username": "gavinliao89",    // Optional: SSH username
    "password": "mypassword",    // Optional: SSH/sudo password
    "host":     "192.168.1.100"  // Optional: VM IP address
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
		} else {
			config.Username = getInput("Enter username: ")
		}
	} else {
		fmt.Printf("Using username from --remote flag: %s\n", config.Username)
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

		// pull from stdin
		var ap netip.AddrPort
		var err error
		for ap, err = netip.ParseAddrPort(getInput("Enter a valid target of the form '<host>:<port>':")); err != nil; {
		}
		return ap

	}()

	// Resolve password
	if config.Password == "" {
		if js.Password != "" {
			config.Password = js.Password
			fmt.Println("Using password from JSON: [hidden]")
		} else if defaultPassword != "" {
			config.Password = defaultPassword
			fmt.Println("Using hardcoded password: [hidden]")
		} else {
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
