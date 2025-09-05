package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
		fmt.Fprintf(os.Stderr, "Mininet Topology Manager\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] [topology.json]\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "Description:\n")
		fmt.Fprintf(os.Stderr, "  This program creates and runs Mininet topologies from JSON files on remote VMs.\n")
		fmt.Fprintf(os.Stderr, "  It handles SSH connections, uploads topology scripts, and manages Mininet sessions.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  --remote=USER@HOST     Remote VM to connect to (e.g., gavinliao89@192.168.1.100)\n")
		fmt.Fprintf(os.Stderr, "  --cli                  Enter interactive Mininet CLI (default: run pingall and exit)\n")
		fmt.Fprintf(os.Stderr, "  --remote-path=PATH     Remote path for generated Python file (default: /tmp/topo_from_json.py)\n")
		fmt.Fprintf(os.Stderr, "  -h, --help            Show this help message\n\n")
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  topology.json          JSON file containing network topology (default: topo.json)\n\n")
		fmt.Fprintf(os.Stderr, "JSON Format:\n")
		fmt.Fprintf(os.Stderr, "  {\n")
		fmt.Fprintf(os.Stderr, "    \"hosts\":    [\"h1\", \"h2\", \"h3\"],\n")
		fmt.Fprintf(os.Stderr, "    \"switches\": [\"s1\", \"s2\"],\n")
		fmt.Fprintf(os.Stderr, "    \"links\":    [[\"h1\",\"s1\"], [\"h2\",\"s1\"], [\"h3\",\"s2\"], [\"s1\",\"s2\"]],\n")
		fmt.Fprintf(os.Stderr, "    \"username\": \"gavinliao89\",    // Optional: SSH username\n")
		fmt.Fprintf(os.Stderr, "    \"password\": \"mypassword\",    // Optional: SSH/sudo password\n")
		fmt.Fprintf(os.Stderr, "    \"host\":     \"192.168.1.100\"  // Optional: VM IP address\n")
		fmt.Fprintf(os.Stderr, "  }\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  %s --remote=user@192.168.1.100 --cli topology.json\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "  %s --cli --remote=user@192.168.1.100\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "  %s topology.json  # Uses hardcoded or prompts for connection info\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "\nNote: If connection info is not provided via --remote flag, the program will\n")
		fmt.Fprintf(os.Stderr, "      check the JSON file, then hardcoded defaults, then prompt for input.\n")
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
