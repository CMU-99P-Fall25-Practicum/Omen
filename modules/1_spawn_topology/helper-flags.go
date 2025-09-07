package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func parseFlags() (*Config, error) {
	config := &Config{}

	// Define flags
	var remote string
	flag.StringVar(&remote, "remote", "", "remote target to run on, e.g. username@192.168.64.5")
	flag.BoolVar(&config.UseCLI, "cli", false, "enter Mininet CLI instead of running pingall")
	flag.StringVar(&config.RemotePath, "remote-path", "/tmp/topo_from_json.py", "remote path for the generated Python file")

	// Custom help flag
	showHelp := flag.Bool("h", false, "show help")
	flag.BoolVar(showHelp, "help", false, "show help")

	// Custom usage function
	flag.Usage = func() {
		var bin string = "UNKNOWN"
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
  --remote=USER@HOST     Remote VM to connect to (e.g., gavinliao89@192.168.1.100)
  --cli                  Enter interactive Mininet CLI (default: run pingall and exit)
  --remote-path=PATH     Remote path for generated Python file (default: /tmp/topo_from_json.py)
  -h, --help             Show this help message

Arguments:
  topo.json          JSON file containing network topology (default: topo.json)

JSON Format example:
  {
    "hosts":    ["h1", "h2", "h3"],
    "switches": ["s1", "s2"],
    "links":    [["h1","s1"], ["h2","s1"], ["h3","s2"], ["s1","s2"]],
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

	flag.Parse()

	// Show help if requested
	if *showHelp {
		flag.Usage()
		return nil, fmt.Errorf("help requested")
	}

	// Parse remote flag if provided
	if remote != "" {
		parts := strings.Split(remote, "@")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid remote format, expected username@host")
		}
		config.Username = parts[0]
		config.Host = parts[1]
	}

	// Get topology file (either from args or default)
	if flag.NArg() > 0 {
		config.TopoFile = flag.Arg(0)
	} else {
		config.TopoFile = defaultTopoFile
	}

	return config, nil
}
