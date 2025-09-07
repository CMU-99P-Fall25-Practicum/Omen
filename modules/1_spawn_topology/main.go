package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

// Configuration - Set these to hardcode values, leave empty for prompting
var (
	defaultHost     = ""
	defaultUsername = ""
	defaultPassword = ""
	defaultTopoFile = "topo.json" // default topology file name
)

// flag values
var (
	remote string
	config Config
)

type Topo struct {
	Hosts    []string    `json:"hosts"`
	Switches []string    `json:"switches"`
	Links    [][2]string `json:"links"`
	// Optional connection info in JSON
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Host     string `json:"host,omitempty"`
}

type Config struct {
	Host       string
	Username   string
	Password   string
	TopoFile   string
	UseCLI     bool
	RemotePath string
}

// generate flags for later parsing
func init() {
	// Define flags
	flag.StringVar(&remote, "remote", "", "remote target to run on, e.g. username@192.168.64.5")
	flag.BoolVar(&config.UseCLI, "cli", false, "enter Mininet CLI instead of running pingall")
	flag.StringVar(&config.RemotePath, "remote-path", "/tmp/topo_from_json.py", "remote path for the generated Python file")

	// set default values
	config.TopoFile = defaultTopoFile

	// Custom help flag
	//showHelp := flag.Bool("h", false, "show help")
	//flag.BoolVar(showHelp, "help", false, "show help")

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
}

func loadTopology(filename string) (*Topo, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read topo file: %w", err)
	}

	var topo Topo
	if err := json.Unmarshal(data, &topo); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}

	return &topo, nil
}

func resolveConfig(config *Config, topo *Topo) error {
	// Priority: command line flags > JSON file > hardcoded defaults > user input

	// Resolve username
	if config.Username == "" {
		if topo.Username != "" {
			config.Username = topo.Username
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
	if config.Host == "" {
		if topo.Host != "" {
			config.Host = topo.Host
			fmt.Printf("Using host from JSON: %s\n", config.Host)
		} else if defaultHost != "" {
			config.Host = defaultHost
			fmt.Printf("Using hardcoded host: %s\n", config.Host)
		} else {
			config.Host = getInput("Enter host IP address: ")
		}
	} else {
		fmt.Printf("Using host from --remote flag: %s\n", config.Host)
	}

	// Resolve password
	if config.Password == "" {
		if topo.Password != "" {
			config.Password = topo.Password
			fmt.Println("Using password from JSON: [hidden]")
		} else if defaultPassword != "" {
			config.Password = defaultPassword
			fmt.Println("Using hardcoded password: [hidden]")
		} else {
			config.Password = getPassword("Enter password (SSH/sudo): ")
		}
	}

	// Validate required fields
	if config.Username == "" || config.Host == "" || config.Password == "" {
		return fmt.Errorf("username, host, and password are required")
	}

	return nil
}

// runRemoteCommand executes the given command against the client.
//
// ! Both stdout and stderr are swallowed.
func runRemoteCommand(client *ssh.Client, command string) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return fmt.Errorf("command failed: %w (output: %s)", err, string(output))
	}

	return nil
}

func main() {
	if err := parseFlags(); err != nil {
		if err.Error() == "help requested" {
			return // Normal exit for help
		}
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	// Load topology from JSON file
	fmt.Printf("Loading topology from: %s\n", config.TopoFile)
	topo, err := loadTopology(config.TopoFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	// Resolve configuration (merge flags, JSON, defaults, and user input)
	if err := resolveConfig(&config, topo); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	// Display final configuration
	fmt.Printf("\n"+`Final Configuration:
	Host       : %s
	Username   : %s
	Password   : [hidden]
	Topology   : %s
	Mode       : %s\n
	Remote path: %s
	Hosts      : %v
	Switches   : %v
	Links      : %v`+"\n",
		config.Host,
		config.Username,

		config.TopoFile,
		map[bool]string{true: "Interactive CLI", false: "Automated pingall"}[config.UseCLI],
		config.RemotePath,
		topo.Hosts,
		topo.Switches,
		topo.Links)

	// Execute the remote Mininet session
	if err := runRemoteMininet(&config, topo); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: run remote mininet: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Program completed successfully!")
}

// Parses the default flagset and sets SSH information iff --remote was specified.
func parseFlags() error {
	flag.Parse()

	// Show help if requested
	/*if *showHelp {
		flag.Usage()
		return nil, fmt.Errorf("help requested")
	}*/

	// Parse remote flag if provided
	if remote != "" {
		parts := strings.Split(remote, "@")
		if len(parts) != 2 {
			return fmt.Errorf("invalid remote format, expected username@host")
		}
		config.Username = parts[0]
		config.Host = parts[1]
	}

	// Get topology file (either from args or default)
	if flag.NArg() > 0 {
		config.TopoFile = flag.Arg(0)
	}

	return nil
}
