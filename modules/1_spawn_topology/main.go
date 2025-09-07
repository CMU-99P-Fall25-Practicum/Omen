package main

import (
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
)

// Configuration - Set these to hardcode values, leave empty for prompting
var (
	defaultHost     = ""
	defaultUsername = ""
	defaultPassword = ""
	defaultTopoFile = "topo.json" // default topology file name
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
	// Parse command line flags
	config, err := parseFlags()
	if err != nil {
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
	if err := resolveConfig(config, topo); err != nil {
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
	if err := runRemoteMininet(config, topo); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: run remote mininet: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Program completed successfully!")
}
